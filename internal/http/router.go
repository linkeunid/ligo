package http

import "context"

// Router abstracts the HTTP router implementation.
type Router interface {
	Group(prefix string) Router
	Use(middleware ...Middleware)
	Handle(method, path string, handler HandlerFunc)
	Serve(addr string) error
}

// GracefulServer extends Router with graceful shutdown capability.
type GracefulServer interface {
	Router
	Shutdown(ctx context.Context) error
}
