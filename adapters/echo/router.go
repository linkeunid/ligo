package echo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"

	echo "github.com/labstack/echo/v5"
	httpifc "github.com/linkeunid/ligo/internal/http"
	"github.com/linkeunid/ligo/internal/di"
	"github.com/linkeunid/ligo/internal/core/logger"
	app "github.com/linkeunid/ligo/internal/app"
)

const errorMsgKey = "error"

// Adapter implements httpifc.Router using Echo v5.
type Adapter struct {
	e          *echo.Echo
	middleware []httpifc.Middleware
	logger     logger.Logger
	server     *http.Server
	container  *di.Container
}

// NewAdapter creates a new Echo v5 adapter.
func NewAdapter() *Adapter {
	e := echo.New()
	e.Logger = slog.New(slog.NewTextHandler(io.Discard, nil)) // Suppress Echo's default logs
	return &Adapter{
		e:         e,
		container: nil, // Will be set by SetContainer
	}
}

// SetContainer sets the root DI container for request-scoped DI.
func (a *Adapter) SetContainer(c *di.Container) {
	a.container = c
	// Add request scope middleware if container is set
	if c != nil {
		a.middleware = append([]httpifc.Middleware{a.requestScopeMiddleware()}, a.middleware...)
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
		prefix:     prefix,
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
	return wrapHandlerWithMiddleware(a.middleware, handler)
}

// Serve starts the HTTP server.
func (a *Adapter) Serve(addr string) error {
	a.server = &http.Server{Addr: addr, Handler: a.e}
	err := a.server.ListenAndServe()
	if err != nil {
		// Check for "address already in use" errors
		var opErr *net.OpError
		if errors.As(err, &opErr) && (opErr.Op == "listen" || strings.Contains(opErr.Error(), "address already in use") || strings.Contains(opErr.Error(), "EADDRINUSE")) {
			return fmt.Errorf("%w: %v", app.ErrAddrInUse, err)
		}
	}
	return err
}

// Shutdown gracefully shuts down the server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	if a.server != nil {
		return a.server.Shutdown(ctx)
	}
	return nil
}

// requestScopeMiddleware creates a per-request child container and sets it on the context.
func (a *Adapter) requestScopeMiddleware() httpifc.Middleware {
	return func(next httpifc.HandlerFunc) httpifc.HandlerFunc {
		return func(ctx httpifc.Context) error {
			child := a.container.NewChild()
			ctx.SetRequestContainer(child)
			return next(ctx)
		}
	}
}

type groupAdapter struct {
	g          *echo.Group
	prefix     string
	middleware []httpifc.Middleware
	logger     logger.Logger
}

func (g *groupAdapter) Group(prefix string) httpifc.Router {
	return &groupAdapter{
		g:          g.g.Group(prefix),
		prefix:     g.prefix + prefix,
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
		g.logger.LogWithContext(logger.ContextRoutes, fmt.Sprintf("Mapped {%s, %s} route", method, g.prefix+path))
	}
}

func (g *groupAdapter) wrapHandler(handler httpifc.HandlerFunc) echo.HandlerFunc {
	return wrapHandlerWithMiddleware(g.middleware, handler)
}

func (g *groupAdapter) Serve(addr string) error {
	return nil
}

// wrapHandlerWithMiddleware applies middleware chain to handler.
func wrapHandlerWithMiddleware(middleware []httpifc.Middleware, handler httpifc.HandlerFunc) echo.HandlerFunc {
	wrapped := httpifc.ApplyMiddleware(middleware, handler)
	return func(c *echo.Context) error {
		return wrapped(newContextAdapter(c))
	}
}

type contextAdapter struct {
	c         *echo.Context
	values    map[string]any
	reqCont   *di.Container
}

var contextPool = sync.Pool{
	New: func() any {
		return &contextAdapter{
			values: make(map[string]any),
		}
	},
}

func newContextAdapter(c *echo.Context) *contextAdapter {
	ctx := contextPool.Get().(*contextAdapter)
	ctx.c = c
	// Reset values map to avoid leaking data between requests
	for k := range ctx.values {
		delete(ctx.values, k)
	}
	ctx.reqCont = nil
	return ctx
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

func (ca *contextAdapter) SetRequestContainer(c *di.Container) {
	ca.reqCont = c
}

func (ca *contextAdapter) GetRequestContainer() *di.Container {
	return ca.reqCont
}

// HTTP response helpers

func (ca *contextAdapter) errorResponse(code int, msg ...string) error {
	m := http.StatusText(code)
	if len(msg) > 0 && msg[0] != "" {
		m = msg[0]
	}
	return ca.c.JSON(code, map[string]string{errorMsgKey: m})
}

func (ca *contextAdapter) OK(v any) error {
	return ca.c.JSON(http.StatusOK, v)
}

func (ca *contextAdapter) Created(v any) error {
	return ca.c.JSON(http.StatusCreated, v)
}

func (ca *contextAdapter) Accepted(v any) error {
	return ca.c.JSON(http.StatusAccepted, v)
}

func (ca *contextAdapter) NoContent() error {
	return ca.c.String(http.StatusNoContent, "")
}

func (ca *contextAdapter) BadRequest(msg ...string) error {
	return ca.errorResponse(http.StatusBadRequest, msg...)
}

func (ca *contextAdapter) Unauthorized(msg ...string) error {
	return ca.errorResponse(http.StatusUnauthorized, msg...)
}

func (ca *contextAdapter) Forbidden(msg ...string) error {
	return ca.errorResponse(http.StatusForbidden, msg...)
}

func (ca *contextAdapter) NotFound(msg ...string) error {
	return ca.errorResponse(http.StatusNotFound, msg...)
}

func (ca *contextAdapter) MethodNotAllowed(msg ...string) error {
	return ca.errorResponse(http.StatusMethodNotAllowed, msg...)
}

func (ca *contextAdapter) NotAcceptable(msg ...string) error {
	return ca.errorResponse(http.StatusNotAcceptable, msg...)
}

func (ca *contextAdapter) RequestTimeout(msg ...string) error {
	return ca.errorResponse(http.StatusRequestTimeout, msg...)
}

func (ca *contextAdapter) Conflict(msg ...string) error {
	return ca.errorResponse(http.StatusConflict, msg...)
}

func (ca *contextAdapter) Gone(msg ...string) error {
	return ca.errorResponse(http.StatusGone, msg...)
}

func (ca *contextAdapter) PreconditionFailed(msg ...string) error {
	return ca.errorResponse(http.StatusPreconditionFailed, msg...)
}

func (ca *contextAdapter) PayloadTooLarge(msg ...string) error {
	return ca.errorResponse(http.StatusRequestEntityTooLarge, msg...)
}

func (ca *contextAdapter) UnsupportedMediaType(msg ...string) error {
	return ca.errorResponse(http.StatusUnsupportedMediaType, msg...)
}

func (ca *contextAdapter) UnprocessableEntity(msg ...string) error {
	return ca.errorResponse(http.StatusUnprocessableEntity, msg...)
}

func (ca *contextAdapter) TooManyRequests(msg ...string) error {
	return ca.errorResponse(http.StatusTooManyRequests, msg...)
}

func (ca *contextAdapter) ImATeapot(msg ...string) error {
	return ca.errorResponse(http.StatusTeapot, msg...)
}

func (ca *contextAdapter) InternalServerError(msg ...string) error {
	return ca.errorResponse(http.StatusInternalServerError, msg...)
}

func (ca *contextAdapter) NotImplemented(msg ...string) error {
	return ca.errorResponse(http.StatusNotImplemented, msg...)
}

func (ca *contextAdapter) BadGateway(msg ...string) error {
	return ca.errorResponse(http.StatusBadGateway, msg...)
}

func (ca *contextAdapter) ServiceUnavailable(msg ...string) error {
	return ca.errorResponse(http.StatusServiceUnavailable, msg...)
}

func (ca *contextAdapter) GatewayTimeout(msg ...string) error {
	return ca.errorResponse(http.StatusGatewayTimeout, msg...)
}

func (ca *contextAdapter) HTTPVersionNotSupported(msg ...string) error {
	return ca.errorResponse(http.StatusHTTPVersionNotSupported, msg...)
}

func (ca *contextAdapter) Stream(reader any) error {
	r, ok := reader.(io.ReadCloser)
	if !ok {
		return ca.c.JSON(http.StatusBadRequest, map[string]string{errorMsgKey: "invalid reader"})
	}
	defer r.Close()

	return ca.c.Stream(http.StatusOK, "", r)
}