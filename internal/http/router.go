package http

import (
	"context"

	"github.com/linkeunid/ligo/internal/core/logger"
)

// Router abstracts the HTTP router implementation.
type Router interface {
	Group(prefix string) Router
	Use(middleware ...Middleware)
	Handle(method, path string, handler HandlerFunc)
	Serve(addr string) error
}

// SetLoggerRouter extends Router with optional logger support.
type SetLoggerRouter interface {
	Router
	SetLogger(logger.Logger)
}

// GracefulServer extends Router with graceful shutdown capability.
type GracefulServer interface {
	Router
	Shutdown(ctx context.Context) error
}
