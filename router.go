package ligo

import "github.com/linkeunid/ligo/internal/http"

// Router is the HTTP router interface.
type Router = http.Router

// HandlerFunc is the standard handler signature.
type HandlerFunc = http.HandlerFunc

// Middleware is a function that wraps a handler.
type Middleware = http.Middleware

// Context wraps HTTP request/response for handlers.
type Context = http.Context

// Controller defines how HTTP routes are registered for a module.
type Controller = http.Controller

// Guard determines if a request should proceed (authorization).
type Guard = http.Guard

// Pipe transforms input data before it reaches the handler.
type Pipe = http.Pipe

// Interceptor wraps the entire request/response cycle.
type Interceptor = http.Interceptor

// ExceptionFilter handles errors and converts them to HTTP responses.
type ExceptionFilter = http.ExceptionFilter

// RouteBuilder provides fluent API for composing routes.
type RouteBuilder = http.RouteBuilder

// ChainRouter provides fluent chain methods for building routes.
type ChainRouter = http.ChainRouter

// NewChainRouter wraps a Router with chain methods.
func NewChainRouter(r Router) ChainRouter {
	return http.NewChainRouter(r)
}
