package ligo

import "net/http"

// HandlerFunc is the standard handler signature.
type HandlerFunc func(ctx Context) error

// Middleware is a function that wraps a handler.
type Middleware func(HandlerFunc) HandlerFunc

// Router abstracts the HTTP router implementation.
type Router interface {
	Group(prefix string) Router
	Use(middleware ...Middleware)
	Handle(method, path string, handler HandlerFunc)
	Serve(addr string) error
}

// Context wraps HTTP request/response for handlers.
type Context interface {
	Request() *http.Request
	Response() http.ResponseWriter
	Param(key string) string
	Bind(v any) error
	JSON(code int, v any) error
	String(code int, s string) error
}
