package http

import (
	"context"
	"io"
	"net/http"

	"github.com/linkeunid/ligo/internal/di"
)

// Adapter is the minimal request/response contract every HTTP adapter
// (Echo, chi, stdlib mux, ...) must implement. ligo wraps an Adapter into
// *Context, which owns the response helpers (OK, BadRequest, Paginated,
// QueryInt, ...). Adapter implementers do NOT implement those helpers
// directly — Context provides them once on top of JSON/String.
type Adapter interface {
	Request() *http.Request
	Response() http.ResponseWriter
	// RequestContext returns the per-request cancellation context.
	// Interceptors derive timeouts from it (so client disconnect and
	// graceful shutdown propagate); handlers should read it when calling
	// cancellable downstream operations (database queries, RPCs).
	RequestContext() context.Context
	Param(key string) string
	// Query returns a single query-string value or "" when absent.
	Query(key string) string
	// BindQuery decodes query parameters into v (struct pointer) using
	// `query:"name"` field tags.
	BindQuery(v any) error
	// Bind decodes the request body into v (struct pointer) using
	// content-type negotiation.
	Bind(v any) error
	JSON(code int, v any) error
	String(code int, s string) error
	// Stream writes the contents of reader to the response. If reader
	// implements io.Closer the adapter closes it when streaming completes.
	Stream(reader io.Reader) error

	// Request-scoped data storage.
	Set(key string, val any)
	Get(key string) any

	// Request-scoped DI container.
	SetRequestContainer(c *di.Container)
	GetRequestContainer() *di.Container
}
