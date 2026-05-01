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

type chainRouter struct {
	router Router
}

// NewChainRouter wraps a Router with chain methods.
func NewChainRouter(router Router) ChainRouter {
	return &chainRouter{router: router}
}

func (cr *chainRouter) Group(prefix string) Router {
	return cr.router.Group(prefix)
}

func (cr *chainRouter) Use(middleware ...Middleware) {
	cr.router.Use(middleware...)
}

func (cr *chainRouter) Handle(method, path string, handler HandlerFunc) {
	cr.router.Handle(method, path, handler)
}

func (cr *chainRouter) Serve(addr string) error {
	return cr.router.Serve(addr)
}

func (cr *chainRouter) GET(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.router, "GET", path, handlers...)
}

func (cr *chainRouter) POST(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.router, "POST", path, handlers...)
}

func (cr *chainRouter) PUT(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.router, "PUT", path, handlers...)
}

func (cr *chainRouter) DELETE(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.router, "DELETE", path, handlers...)
}

func (cr *chainRouter) PATCH(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.router, "PATCH", path, handlers...)
}

func (cr *chainRouter) OPTIONS(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.router, "OPTIONS", path, handlers...)
}

func (cr *chainRouter) HEAD(path string, handlers ...HandlerFunc) RouteBuilder {
	return newRouteBuilder(cr.router, "HEAD", path, handlers...)
}
