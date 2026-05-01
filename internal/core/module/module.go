package module

import "reflect"

// Module represents a self-contained unit of functionality.
type Module struct {
	Name        string
	Providers   []any // Provider
	Controllers []ControllerConstructor
	Imports     []Module
	Middlewares []MiddlewareConstructor
}

// MiddlewareConstructor holds a middleware constructor with its dependency types.
type MiddlewareConstructor struct {
	Fn       any
	ArgTypes []reflect.Type
}

// ControllerConstructor holds a controller constructor with its dependency types.
type ControllerConstructor struct {
	Fn       any
	ArgTypes []reflect.Type
}

// ModuleOption configures a Module.
type ModuleOption func(*Module)

// New creates a new module with the given name and options.
func New(name string, opts ...ModuleOption) Module {
	m := Module{Name: name}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// Providers adds providers to the module.
func Providers(providers ...any) ModuleOption {
	return func(m *Module) {
		m.Providers = append(m.Providers, providers...)
	}
}

// Imports adds child modules.
func Imports(modules ...Module) ModuleOption {
	return func(m *Module) {
		m.Imports = append(m.Imports, modules...)
	}
}

// Controllers adds controller constructors that receive dependencies via DI.
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
			m.Controllers = append(m.Controllers, ControllerConstructor{
				Fn:       c,
				ArgTypes: argTypes,
			})
		}
	}
}

// Middlewares adds middleware constructors that receive dependencies via DI.
func Middlewares(constructors ...any) ModuleOption {
	return func(m *Module) {
		for _, c := range constructors {
			fnValue := reflect.ValueOf(c)
			fnType := fnValue.Type()
			if fnType.Kind() != reflect.Func {
				panic("ligo: Middlewares expects a function")
			}
			argTypes := make([]reflect.Type, fnType.NumIn())
			for i := 0; i < fnType.NumIn(); i++ {
				argTypes[i] = fnType.In(i)
			}
			m.Middlewares = append(m.Middlewares, MiddlewareConstructor{
				Fn:       c,
				ArgTypes: argTypes,
			})
		}
	}
}
