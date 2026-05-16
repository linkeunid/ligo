package http

import (
	"net/http"

	"github.com/linkeunid/ligo/internal/di"
)

// Controller defines how HTTP routes are registered for a module.
type Controller interface {
	Routes(r Router)
}

// HandlerFunc is the standard handler signature.
type HandlerFunc func(ctx Context) error

// Middleware is a function that wraps a handler.
type Middleware func(HandlerFunc) HandlerFunc

// Context wraps HTTP request/response for handlers.
type Context interface {
	Request() *http.Request
	Response() http.ResponseWriter
	Param(key string) string
	// Query returns a single query-string value or "" when absent.
	Query(key string) string
	// QueryDefault returns the query value or def when absent/empty.
	QueryDefault(key, def string) string
	// QueryInt parses a query value as int, returning def on missing/invalid.
	QueryInt(key string, def int) int
	// BindQuery decodes query parameters into v (struct pointer) using
	// `query:"name"` field tags. See ligo.BindQuery for supported types.
	BindQuery(v any) error
	// Paginate reads ?page= and ?per_page= and applies the normalization
	// rules of ListQuery.Normalize (page<1→1; per_page absent→default;
	// per_page<0→0; capped at max). per_page=0 is honored as LIMIT 0.
	Paginate(defaultPerPage, maxPerPage int) ListQuery
	Bind(v any) error
	JSON(code int, v any) error
	String(code int, s string) error

	// Request-scoped data storage
	Set(key string, val any)
	Get(key string) any

	// Request-scoped DI container
	SetRequestContainer(c *di.Container)
	GetRequestContainer() *di.Container

	// HTTP response helpers
	OK(v any) error
	Created(v any) error
	Accepted(v any) error
	NoContent() error
	// List writes 200 with a { "data": [...], "meta": { "count": N } } envelope.
	// A nil or typed-nil slice is coerced to [] so the body never renders null.
	List(items any) error
	// Paginated writes 200 with a { "data": [...], "meta": { page, per_page,
	// total, total_pages } } envelope. Items follow the same nil rule as List.
	Paginated(items any, page, perPage int, total int64) error
	BadRequest(msg ...string) error
	Unauthorized(msg ...string) error
	Forbidden(msg ...string) error
	NotFound(msg ...string) error
	MethodNotAllowed(msg ...string) error
	NotAcceptable(msg ...string) error
	RequestTimeout(msg ...string) error
	Conflict(msg ...string) error
	Gone(msg ...string) error
	PreconditionFailed(msg ...string) error
	PayloadTooLarge(msg ...string) error
	UnsupportedMediaType(msg ...string) error
	UnprocessableEntity(msg ...string) error
	TooManyRequests(msg ...string) error
	ImATeapot(msg ...string) error
	InternalServerError(msg ...string) error
	NotImplemented(msg ...string) error
	BadGateway(msg ...string) error
	ServiceUnavailable(msg ...string) error
	GatewayTimeout(msg ...string) error
	HTTPVersionNotSupported(msg ...string) error
	Stream(reader any) error
}
