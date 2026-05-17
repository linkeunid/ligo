package http

// ChainRouter provides fluent chain methods for building routes.
type ChainRouter interface {
	Router
	GET(path string, handlers ...HandlerFunc) RouteBuilder
	POST(path string, handlers ...HandlerFunc) RouteBuilder
	PUT(path string, handlers ...HandlerFunc) RouteBuilder
	DELETE(path string, handlers ...HandlerFunc) RouteBuilder
	PATCH(path string, handlers ...HandlerFunc) RouteBuilder
	OPTIONS(path string, handlers ...HandlerFunc) RouteBuilder
	HEAD(path string, handlers ...HandlerFunc) RouteBuilder
}

// chainRouter embeds Router so that Router methods (Group/Use/Handle/Serve)
// are forwarded automatically. Adding a method to the Router interface no
// longer silently breaks this wrapper.
type chainRouter struct {
	Router
}

// NewChainRouter wraps a Router with chain methods.
func NewChainRouter(router Router) ChainRouter {
	return &chainRouter{Router: router}
}

func (cr *chainRouter) GET(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.Router, "GET", path, handlers...)
}

func (cr *chainRouter) POST(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.Router, "POST", path, handlers...)
}

func (cr *chainRouter) PUT(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.Router, "PUT", path, handlers...)
}

func (cr *chainRouter) DELETE(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.Router, "DELETE", path, handlers...)
}

func (cr *chainRouter) PATCH(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.Router, "PATCH", path, handlers...)
}

func (cr *chainRouter) OPTIONS(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.Router, "OPTIONS", path, handlers...)
}

func (cr *chainRouter) HEAD(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.Router, "HEAD", path, handlers...)
}
