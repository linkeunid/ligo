package http

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/lifecycle"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
)

// unwrapController extracts the underlying constructor and whether it was wrapped.
// Combines type assertion and unwrapping into a single operation.
func unwrapController(fn any) (unwrapped any, isHooked bool) {
	uw, ok := fn.(interface{ Unwrap() any })
	if ok {
		return uw.Unwrap(), true
	}
	return fn, false
}

// Binder handles controller registration and dependency injection.
type Binder struct {
	container *container.Container
	router    Router
	logger    logger.Logger
}

// NewBinder creates a new binder instance.
func NewBinder(c *container.Container, r Router, log logger.Logger) *Binder {
	return &Binder{
		container: c,
		router:    r,
		logger:    log,
	}
}

// BindControllers registers all controllers from modules, applying module middleware per group.
// Returns collected lifecycle hooks from all controllers.
func (b *Binder) BindControllers(modules []module.Module) ([]lifecycle.Hooks, error) {
	var allHooks []lifecycle.Hooks
	for _, mod := range modules {
		hooks, err := b.bindModuleControllers(mod)
		if err != nil {
			return nil, err
		}
		allHooks = append(allHooks, hooks...)
	}
	return allHooks, nil
}

func (b *Binder) bindModuleControllers(mod module.Module) ([]lifecycle.Hooks, error) {
	var modMw []Middleware
	for _, mc := range mod.Middlewares {
		mw, err := b.resolveMiddleware(mc, mod.Name)
		if err != nil {
			return nil, err
		}
		modMw = append(modMw, mw)
	}

	router := b.router
	if len(modMw) > 0 {
		g := b.router.Group("/" + mod.Name)
		for _, mw := range modMw {
			g.Use(mw)
		}
		router = g
	}

	var allHooks []lifecycle.Hooks
	for _, cc := range mod.Controllers {
		hooks, err := b.bindController(cc, router, mod.Name)
		if err != nil {
			return nil, err
		}
		if hooks.OnInit != nil || hooks.OnBootstrap != nil || hooks.OnDestroy != nil || hooks.OnShutdown != nil {
			allHooks = append(allHooks, hooks)
		}
	}

	for _, child := range mod.Imports {
		hooks, err := b.bindModuleControllers(child)
		if err != nil {
			return nil, err
		}
		allHooks = append(allHooks, hooks...)
	}

	return allHooks, nil
}

func (b *Binder) resolveMiddleware(mc module.MiddlewareConstructor, modName string) (Middleware, error) {
	return b.resolveConstructor(mc.Fn, "Middleware", modName, func(v reflect.Value) (Middleware, error) {
		mw, ok := v.Interface().(Middleware)
		if !ok {
			return nil, fmt.Errorf("ligo: constructor must return Middleware")
		}
		return mw, nil
	})
}

func (b *Binder) resolveConstructor(fn any, typeName string, modName string, validate func(reflect.Value) (Middleware, error)) (Middleware, error) {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("ligo: %s must be a function", typeName)
	}

	argTypes := make([]reflect.Type, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		argTypes[i] = fnType.In(i)
	}

	// Resolve dependencies
	args := make([]reflect.Value, len(argTypes))
	for i, argType := range argTypes {
		resolved, err := container.ResolveByType(b.container, argType)
		if err != nil {
			return nil, &ErrControllerBinding{
				Module:     modName,
				TypeName:   typeName,
				Dependency: argType.String(),
				Cause:      err,
			}
		}
		args[i] = reflect.ValueOf(resolved)
	}

	// Call constructor
	out := fnValue.Call(args)
	if len(out) == 0 {
		return nil, fmt.Errorf("ligo: %s constructor must return a value", typeName)
	}

	return validate(out[0])
}

func (b *Binder) bindController(cc module.ControllerConstructor, router Router, modName string) (lifecycle.Hooks, error) {
	var capturedCtrl any

	// Unwrap controller constructor if wrapped with HookedController
	constructorFn, _ := unwrapController(cc.Fn)

	_, err := b.resolveConstructor(constructorFn, "Controller", modName, func(v reflect.Value) (Middleware, error) {
		// Capture the controller value for hook collection
		capturedCtrl = v.Interface()

		// Try to call Routes() if the controller implements it
		if ctrl, ok := capturedCtrl.(Controller); ok {
			ctrl.Routes(router)
		}

		if b.logger != nil {
			ctrlName := b.extractControllerName(constructorFn)
			if ctrlName == "" {
				ctrlName = "controller"
			}
			b.logger.LogWithContext(logger.ContextRoutes, fmt.Sprintf("%s controller registered", ctrlName),
				logger.Field{Key: "module", Value: modName},
			)
		}

		return nil, nil
	})

	// Try explicit hook registration first, then fall back to interface-based detection.
	// This works for both HookedController (with Register method) and regular controllers.
	if registerable, ok := capturedCtrl.(interface{ Register(*lifecycle.HookRegistry) }); ok {
		registry := lifecycle.NewHookRegistry()
		registerable.Register(registry)
		return registry.ToHooks(), err
	}
	// Fall back to interface-based hook detection (duck-typing)
	return lifecycle.CollectHooks(capturedCtrl), err
}

// extractControllerName extracts the controller name from the constructor.
func (b *Binder) extractControllerName(fn any) string {
	name := logger.ExtractProviderName(fn)
	if name == "unknown" || name == "" {
		return "Controller"
	}
	return name
}

// ErrControllerBinding is returned when a controller's dependency chain cannot be fully resolved.
type ErrControllerBinding struct {
	Module     string
	TypeName   string
	Dependency string
	Cause      error
}

func (e *ErrControllerBinding) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "ligo: cannot build %s in module %q\n", e.TypeName, e.Module)
	writeChain(&b, e.Dependency, e.TypeName, e.Cause, "  ")
	return b.String()
}

func (e *ErrControllerBinding) Unwrap() error { return e.Cause }

// writeChain appends the dependency chain to b.
// dep is the type that failed; requiredBy is its direct consumer.
func writeChain(b *strings.Builder, dep, requiredBy string, cause error, indent string) {
	fmt.Fprintf(b, "%s%s  <- required by %s\n", indent, dep, requiredBy)
	var next *container.ErrMissingDependency
	if errors.As(cause, &next) {
		writeChain(b, next.Type, dep, next.Cause, indent+"  ")
	} else if cause != nil {
		fmt.Fprintf(b, "%s  %s", indent, cause.Error())
	} else {
		fmt.Fprintf(b, "%s  no provider registered", indent)
	}
}
