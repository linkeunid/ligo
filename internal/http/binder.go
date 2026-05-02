package http

import (
	"fmt"
	"reflect"

	"github.com/linkeunid/ligo/internal/core/container"
	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/core/module"
)

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
func (b *Binder) BindControllers(modules []module.Module) error {
	for _, mod := range modules {
		if err := b.bindModuleControllers(mod); err != nil {
			return err
		}
	}
	return nil
}

func (b *Binder) bindModuleControllers(mod module.Module) error {
	var modMw []Middleware
	for _, mc := range mod.Middlewares {
		mw, err := b.resolveMiddleware(mc)
		if err != nil {
			return err
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

	for _, cc := range mod.Controllers {
		if err := b.bindController(cc, router, mod.Name); err != nil {
			return err
		}
	}

	for _, child := range mod.Imports {
		if err := b.bindModuleControllers(child); err != nil {
			return err
		}
	}

	return nil
}

func (b *Binder) resolveMiddleware(mc module.MiddlewareConstructor) (Middleware, error) {
	return b.resolveConstructor(mc.Fn, "Middleware", func(v reflect.Value) (Middleware, error) {
		mw, ok := v.Interface().(Middleware)
		if !ok {
			return nil, fmt.Errorf("ligo: constructor must return Middleware")
		}
		return mw, nil
	})
}

func (b *Binder) resolveConstructor(fn any, typeName string, validate func(reflect.Value) (Middleware, error)) (Middleware, error) {
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
		resolved := container.ResolveByType(b.container, argType)
		if resolved == nil {
			return nil, fmt.Errorf("ligo: missing dependency %s for %s", argType.String(), typeName)
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

func (b *Binder) bindController(cc module.ControllerConstructor, router Router, modName string) error {
	_, err := b.resolveConstructor(cc.Fn, "Controller", func(v reflect.Value) (Middleware, error) {
		ctrl, ok := v.Interface().(Controller)
		if !ok {
			return nil, fmt.Errorf("ligo: constructor must return Controller")
		}

		if ctrl != nil {
			ctrl.Routes(router)
		}

		if b.logger != nil {
			ctrlName := b.extractControllerName(cc.Fn)
			if ctrlName == "" {
				ctrlName = "controller"
			}
			b.logger.LogWithContext(logger.ContextRoutes, fmt.Sprintf("%s controller registered", ctrlName),
				logger.Field{Key: "module", Value: modName},
			)
		}

		return nil, nil
	})
	return err
}

// extractControllerName extracts the controller name from the constructor.
func (b *Binder) extractControllerName(fn any) string {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return "Controller"
	}

	// Try to get name from return type
	if fnType.NumOut() > 0 {
		retTyp := fnType.Out(0)
		if retTyp.Kind() == reflect.Ptr {
			retTyp = retTyp.Elem()
		}
		if retTyp.Name() != "" && retTyp.Name() != "Controller" {
			return retTyp.Name()
		}
	}

	// Fallback: extract from function name (NewUserController -> UserController)
	fnName := fnType.Name()
	if len(fnName) > 3 && fnName[:3] == "New" {
		return fnName[3:]
	}
	return fnName
}