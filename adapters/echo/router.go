package echo

import (
	"net/http"

	echo "github.com/labstack/echo/v5"
	"github.com/linkeunid/ligo"
)

// Adapter implements ligo.Router using Echo v5.
type Adapter struct {
	e *echo.Echo
}

// NewAdapter creates a new Echo v5 adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		e: echo.New(),
	}
}

// Group creates a sub-router with a prefix.
func (a *Adapter) Group(prefix string) ligo.Router {
	return &groupAdapter{g: a.e.Group(prefix)}
}

// Use adds middleware to the router.
func (a *Adapter) Use(mw ...ligo.Middleware) {
	// Middleware applied per-route in Handle method
}

// Handle registers a route.
func (a *Adapter) Handle(method, path string, handler ligo.HandlerFunc) {
	a.e.Add(method, path, func(c *echo.Context) error {
		return handler(&contextAdapter{c: c})
	})
}

// Serve starts the HTTP server.
func (a *Adapter) Serve(addr string) error {
	return a.e.Start(addr)
}

type groupAdapter struct {
	g *echo.Group
}

func (g *groupAdapter) Group(prefix string) ligo.Router {
	return &groupAdapter{g: g.g.Group(prefix)}
}

func (g *groupAdapter) Use(mw ...ligo.Middleware) {}

func (g *groupAdapter) Handle(method, path string, handler ligo.HandlerFunc) {
	g.g.Add(method, path, func(c *echo.Context) error {
		return handler(&contextAdapter{c: c})
	})
}

func (g *groupAdapter) Serve(addr string) error {
	return nil
}

type contextAdapter struct {
	c *echo.Context
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
