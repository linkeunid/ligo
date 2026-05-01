package ligo

import "reflect"

// Module represents a self-contained unit of functionality.
type Module struct {
	name        string
	providers   []Provider
	controllers []controllerConstructor
	imports     []Module
}

// controllerConstructor holds a controller constructor with its dependency types
type controllerConstructor struct {
	fn       interface{}     // constructor function
	argTypes []reflect.Type  // dependency types for auto-injection
}

// ModuleOption configures a Module.
type ModuleOption func(*Module)

// NewModule creates a new module with the given name and options.
func NewModule(name string, opts ...ModuleOption) Module {
	m := Module{name: name}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// Providers adds providers to the module.
func Providers(providers ...Provider) ModuleOption {
	return func(m *Module) {
		m.providers = append(m.providers, providers...)
	}
}

// Imports adds child modules.
func Imports(modules ...Module) ModuleOption {
	return func(m *Module) {
		m.imports = append(m.imports, modules...)
	}
}

// Controllers adds controller constructors that receive dependencies via DI.
// Controllers are constructed with auto-injection after the container is built.
// Example: ligo.Controllers(func(svc *UserService) Controller { return NewController(svc) })
func Controllers(constructors ...any) ModuleOption {
	return func(m *Module) {
		for _, c := range constructors {
			fnValue := reflect.ValueOf(c)
			fnType := fnValue.Type()
			if fnType.Kind() != reflect.Func {
				panic("ligo: Controllers expects a function")
			}
			argTypes := make([]reflect.Type, fnType.NumIn())
			for i := 0; i < fnType.NumIn(); i++ {
				argTypes[i] = fnType.In(i)
			}
			m.controllers = append(m.controllers, controllerConstructor{
				fn:       c,
				argTypes: argTypes,
			})
		}
	}
}
