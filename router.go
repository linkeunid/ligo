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
