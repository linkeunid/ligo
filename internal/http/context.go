package http

// Controller defines how HTTP routes are registered for a module.
type Controller interface {
	Routes(r Router)
}

// HandlerFunc is the standard handler signature. The ctx pointer carries
// both the adapter plumbing and the response helpers on a single value.
type HandlerFunc func(ctx *Context) error

// Middleware wraps a handler.
type Middleware func(HandlerFunc) HandlerFunc

// Context wraps an Adapter and exposes the framework's response helpers.
// Handlers receive *Context; the underlying Adapter is implementation
// detail. Method calls on the embedded Adapter (Request, JSON, Bind, ...)
// are promoted so ctx.JSON(200, v), ctx.Request(), ctx.Set(...) all work
// transparently.
type Context struct {
	Adapter
}

// NewContext wraps an Adapter for handler dispatch. Adapters call this
// once per request and pass the result to the wrapped HandlerFunc.
func NewContext(a Adapter) *Context {
	return &Context{Adapter: a}
}
