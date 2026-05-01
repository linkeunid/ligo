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
	// Resolve module middleware
	var modMw []Middleware
	for _, mc := range mod.Middlewares {
		mw, err := b.resolveMiddleware(mc)
		if err != nil {
			return err
		}
		modMw = append(modMw, mw)
	}

	// Apply module middleware if present
	if len(modMw) > 0 {
		moduleRouter := b.router.Group("/" + mod.Name)
		for _, mw := range modMw {
			moduleRouter.Use(mw)
		}
		// Bind controllers to module-scoped router
		for _, cc := range mod.Controllers {
			if err := b.bindControllerTo(cc, moduleRouter, mod.Name); err != nil {
				return err
			}
		}
	} else {
		// No module middleware, bind to root router
		for _, cc := range mod.Controllers {
			if err := b.bindController(cc, mod.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Binder) resolveMiddleware(mc module.MiddlewareConstructor) (Middleware, error) {
	fnValue := reflect.ValueOf(mc.Fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("ligo: middleware must be a function")
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
			return nil, fmt.Errorf("ligo: missing dependency %s for middleware", argType.String())
		}
		args[i] = reflect.ValueOf(resolved)
	}

	// Call constructor
	out := fnValue.Call(args)
	if len(out) == 0 {
		return nil, fmt.Errorf("ligo: middleware constructor must return a Middleware")
	}

	mw, ok := out[0].Interface().(Middleware)
	if !ok {
		return nil, fmt.Errorf("ligo: constructor must return Middleware")
	}
	return mw, nil
}

func (b *Binder) bindController(cc module.ControllerConstructor, modName string) error {
	fnValue := reflect.ValueOf(cc.Fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("ligo: controller must be a function")
	}

	argTypes := make([]reflect.Type, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		argTypes[i] = fnType.In(i)
	}

	args := make([]reflect.Value, len(argTypes))
	for i, argType := range argTypes {
		resolved := container.ResolveByType(b.container, argType)
		if resolved == nil {
			return fmt.Errorf("ligo: missing dependency %s for controller", argType.String())
		}
		args[i] = reflect.ValueOf(resolved)
	}

	out := fnValue.Call(args)
	if len(out) == 0 {
		return fmt.Errorf("ligo: controller constructor must return a Controller")
	}

	ctrl, ok := out[0].Interface().(Controller)
	if !ok {
		return fmt.Errorf("ligo: constructor must return Controller")
	}

	if ctrl != nil {
		ctrl.Routes(b.router)
	}

	// Log controller registration
	if b.logger != nil {
		ctrlName := b.extractControllerName(cc)
		if ctrlName == "" {
			ctrlName = "controller"
		}
		b.logger.LogWithContext(logger.ContextRoutes, fmt.Sprintf("%s controller registered", ctrlName),
			logger.Field{Key: "module", Value: modName},
		)
	}

	return nil
}

func (b *Binder) bindControllerTo(cc module.ControllerConstructor, r Router, modName string) error {
	fnValue := reflect.ValueOf(cc.Fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("ligo: controller must be a function")
	}

	argTypes := make([]reflect.Type, fnType.NumIn())
	for i := 0; i < fnType.NumIn(); i++ {
		argTypes[i] = fnType.In(i)
	}

	args := make([]reflect.Value, len(argTypes))
	for i, argType := range argTypes {
		resolved := container.ResolveByType(b.container, argType)
		if resolved == nil {
			return fmt.Errorf("ligo: missing dependency %s for controller", argType.String())
		}
		args[i] = reflect.ValueOf(resolved)
	}

	out := fnValue.Call(args)
	if len(out) == 0 {
		return fmt.Errorf("ligo: controller constructor must return a Controller")
	}

	ctrl, ok := out[0].Interface().(Controller)
	if !ok {
		return fmt.Errorf("ligo: constructor must return Controller")
	}

	if ctrl != nil {
		ctrl.Routes(r)
	}

	// Log controller registration
	if b.logger != nil {
		ctrlName := b.extractControllerName(cc)
		if ctrlName == "" {
			ctrlName = "controller"
		}
		b.logger.LogWithContext(logger.ContextRoutes, fmt.Sprintf("%s controller registered", ctrlName),
			logger.Field{Key: "module", Value: modName},
		)
	}

	return nil
}

// extractControllerName extracts the controller name from the constructor.
func (b *Binder) extractControllerName(cc module.ControllerConstructor) string {
	fnType := reflect.TypeOf(cc.Fn)
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