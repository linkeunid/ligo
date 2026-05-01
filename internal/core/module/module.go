package module

// Module represents a self-contained unit of functionality.
type Module struct {
	Name        string
	Providers   []any // Provider
	Controllers []ControllerConstructor
	Imports     []Module
	Middlewares []MiddlewareConstructor
}

// MiddlewareConstructor holds a middleware constructor.
type MiddlewareConstructor struct {
	Fn any
}

// ControllerConstructor holds a controller constructor.
type ControllerConstructor struct {
	Fn any
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
			m.Controllers = append(m.Controllers, ControllerConstructor{Fn: c})
		}
	}
}

// Middlewares adds middleware constructors that receive dependencies via DI.
func Middlewares(constructors ...any) ModuleOption {
	return func(m *Module) {
		for _, c := range constructors {
			m.Middlewares = append(m.Middlewares, MiddlewareConstructor{Fn: c})
		}
	}
}
