package http

import (
	nethttp "net/http"
)

// HTTP response helpers — composed on the embedded Adapter's JSON/String.

func (c *Context) OK(v any) error       { return c.JSON(200, v) }
func (c *Context) Created(v any) error  { return c.JSON(201, v) }
func (c *Context) Accepted(v any) error { return c.JSON(202, v) }
func (c *Context) NoContent() error     { return c.String(204, "") }

// List writes 200 with a { "data": [...], "meta": { "count": N } } envelope.
// A nil or typed-nil slice is coerced to [] so the body never renders null.
func (c *Context) List(items any) error {
	return c.JSON(200, NewListResponse(items))
}

// Paginated writes 200 with a { "data": [...], "meta": { page, per_page,
// total, total_pages } } envelope. Items follow the same nil rule as List.
func (c *Context) Paginated(items any, page, perPage int, total int64) error {
	return c.JSON(200, NewPageResponse(items, page, perPage, total))
}

// errorReply writes a { "error": msg } body with the given status. msg
// defaults to net/http.StatusText(code) when omitted or empty.
func (c *Context) errorReply(code int, msg []string) error {
	m := nethttp.StatusText(code)
	if len(msg) > 0 && msg[0] != "" {
		m = msg[0]
	}
	return c.JSON(code, map[string]string{"error": m})
}

// BadRequest writes a 400 with the given (or default) message.
func (c *Context) BadRequest(msg ...string) error           { return c.errorReply(400, msg) }
func (c *Context) Unauthorized(msg ...string) error         { return c.errorReply(401, msg) }
func (c *Context) Forbidden(msg ...string) error            { return c.errorReply(403, msg) }
func (c *Context) NotFound(msg ...string) error             { return c.errorReply(404, msg) }
func (c *Context) MethodNotAllowed(msg ...string) error     { return c.errorReply(405, msg) }
func (c *Context) NotAcceptable(msg ...string) error        { return c.errorReply(406, msg) }
func (c *Context) RequestTimeout(msg ...string) error       { return c.errorReply(408, msg) }
func (c *Context) Conflict(msg ...string) error             { return c.errorReply(409, msg) }
func (c *Context) Gone(msg ...string) error                 { return c.errorReply(410, msg) }
func (c *Context) PreconditionFailed(msg ...string) error   { return c.errorReply(412, msg) }
func (c *Context) PayloadTooLarge(msg ...string) error      { return c.errorReply(413, msg) }
func (c *Context) UnsupportedMediaType(msg ...string) error { return c.errorReply(415, msg) }
func (c *Context) UnprocessableEntity(msg ...string) error  { return c.errorReply(422, msg) }
func (c *Context) TooManyRequests(msg ...string) error      { return c.errorReply(429, msg) }
func (c *Context) ImATeapot(msg ...string) error            { return c.errorReply(418, msg) }

// InternalServerError writes a 500 with the given (or default) message.
func (c *Context) InternalServerError(msg ...string) error { return c.errorReply(500, msg) }
func (c *Context) NotImplemented(msg ...string) error      { return c.errorReply(501, msg) }
func (c *Context) BadGateway(msg ...string) error          { return c.errorReply(502, msg) }
func (c *Context) ServiceUnavailable(msg ...string) error  { return c.errorReply(503, msg) }
func (c *Context) GatewayTimeout(msg ...string) error      { return c.errorReply(504, msg) }
func (c *Context) HTTPVersionNotSupported(msg ...string) error {
	return c.errorReply(505, msg)
}

// QueryDefault returns the query value or def when absent/empty.
func (c *Context) QueryDefault(key, def string) string {
	return QueryDefault(c.Request(), key, def)
}

// QueryInt parses a query value as int, returning def on missing/invalid.
func (c *Context) QueryInt(key string, def int) int {
	return QueryInt(c.Request(), key, def)
}

// Paginate reads ?page= and ?per_page= and applies ListQuery.Normalize.
func (c *Context) Paginate(defaultPerPage, maxPerPage int) ListQuery {
	q := ParseListQuery(c.Request())
	q.Normalize(defaultPerPage, maxPerPage)
	return q
}
