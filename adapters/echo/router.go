package echo

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	echo "github.com/labstack/echo/v5"
	httpifc "github.com/linkeunid/ligo/internal/http"
	"github.com/linkeunid/ligo/internal/core/logger"
)

// Adapter implements httpifc.Router using Echo v5.
type Adapter struct {
	e          *echo.Echo
	middleware []httpifc.Middleware
	logger     logger.Logger
}

// NewAdapter creates a new Echo v5 adapter.
func NewAdapter() *Adapter {
	e := echo.New()
	e.Logger = slog.New(slog.NewTextHandler(io.Discard, nil)) // Suppress Echo's default logs
	return &Adapter{
		e: e,
	}
}

// SetLogger sets the logger for route mapping logs.
func (a *Adapter) SetLogger(log logger.Logger) {
	a.logger = log
}

// Group creates a sub-router with a prefix.
func (a *Adapter) Group(prefix string) httpifc.Router {
	return &groupAdapter{
		g:          a.e.Group(prefix),
		middleware: a.middleware, // inherit global middleware
		logger:     a.logger,
	}
}

// Use adds middleware to the router.
func (a *Adapter) Use(mw ...httpifc.Middleware) {
	a.middleware = append(a.middleware, mw...)
}

// Handle registers a route with middleware chain.
func (a *Adapter) Handle(method, path string, handler httpifc.HandlerFunc) {
	a.e.Add(method, path, a.wrapHandler(handler))

	if a.logger != nil {
		a.logger.LogWithContext(logger.ContextRoutes, fmt.Sprintf("Mapped {%s, %s} route", method, path))
	}
}

// wrapHandler applies middleware chain to handler.
func (a *Adapter) wrapHandler(handler httpifc.HandlerFunc) echo.HandlerFunc {
	wrapped := handler
	for i := len(a.middleware) - 1; i >= 0; i-- {
		wrapped = a.middleware[i](wrapped)
	}
	return func(c *echo.Context) error {
		return wrapped(newContextAdapter(c))
	}
}

// Serve starts the HTTP server.
func (a *Adapter) Serve(addr string) error {
	return a.e.Start(addr)
}

type groupAdapter struct {
	g          *echo.Group
	middleware []httpifc.Middleware
	logger     logger.Logger
}

func (g *groupAdapter) Group(prefix string) httpifc.Router {
	return &groupAdapter{
		g:          g.g.Group(prefix),
		middleware: g.middleware,
		logger:     g.logger,
	}
}

func (g *groupAdapter) Use(mw ...httpifc.Middleware) {
	g.middleware = append(g.middleware, mw...)
}

func (g *groupAdapter) Handle(method, path string, handler httpifc.HandlerFunc) {
	g.g.Add(method, path, g.wrapHandler(handler))

	if g.logger != nil {
		g.logger.LogWithContext(logger.ContextRoutes, fmt.Sprintf("Mapped {%s, %s} route", method, path))
	}
}

func (g *groupAdapter) wrapHandler(handler httpifc.HandlerFunc) echo.HandlerFunc {
	wrapped := handler
	for i := len(g.middleware) - 1; i >= 0; i-- {
		wrapped = g.middleware[i](wrapped)
	}
	return func(c *echo.Context) error {
		return wrapped(newContextAdapter(c))
	}
}

func (g *groupAdapter) Serve(addr string) error {
	return nil
}

type contextAdapter struct {
	c      *echo.Context
	values map[string]any
}

func newContextAdapter(c *echo.Context) *contextAdapter {
	return &contextAdapter{
		c:      c,
		values: make(map[string]any),
	}
}

func (ca *contextAdapter) Request() *http.Request {
	return ca.c.Request()
}

func (ca *contextAdapter) Response() http.ResponseWriter {
	return ca.c.Response()
}

func (ca *contextAdapter) Param(key string) string {
	return ca.c.Param(key)
}

func (ca *contextAdapter) Bind(v any) error {
	return ca.c.Bind(v)
}

func (ca *contextAdapter) JSON(code int, v any) error {
	return ca.c.JSON(code, v)
}

func (ca *contextAdapter) String(code int, s string) error {
	return ca.c.String(code, s)
}

func (ca *contextAdapter) Set(key string, val any) {
	ca.values[key] = val
}

func (ca *contextAdapter) Get(key string) any {
	return ca.values[key]
}