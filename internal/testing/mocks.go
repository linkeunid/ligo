package testing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"

	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/di"
	httpifc "github.com/linkeunid/ligo/internal/http"
)

// Compile-time assertion that MockContext satisfies the Context interface.
// If the interface drifts, this fails the build instead of failing silently
// the moment a test tries to use MockContext as a Context.
var _ httpifc.Context = (*MockContext)(nil)

// MockContext is a mock implementation of ligo.Context for testing.
//
// Bind/BindQuery defaults are no-ops. To drive realistic binding in tests:
//   - SetBody(v) stashes a value that Bind copies into its target via reflect.
//   - SetQueryBody(v) does the same for BindQuery.
//   - WithBindError(err) / WithBindQueryError(err) inject failures.
type MockContext struct {
	values        map[string]any
	req           *http.Request
	resp          http.ResponseWriter
	reqCont       *di.Container
	bindBody      any
	bindErr       error
	bindQueryBody any
	bindQueryErr  error
}

// NewMockContext creates a new mock context for testing.
func NewMockContext() *MockContext {
	return &MockContext{
		values: make(map[string]any),
		req:    httptest.NewRequest("GET", "/", nil),
		resp:   httptest.NewRecorder(),
	}
}

// Request returns the mock HTTP request.
func (m *MockContext) Request() *http.Request {
	return m.req
}

// Response returns the mock HTTP response writer.
func (m *MockContext) Response() http.ResponseWriter {
	return m.resp
}

// RequestContext returns the embedded request's context, or Background if no request is set.
func (m *MockContext) RequestContext() context.Context {
	if m.req != nil {
		return m.req.Context()
	}
	return context.Background()
}

// Param returns a mock path parameter (always empty string).
func (m *MockContext) Param(key string) string {
	return ""
}

// Query delegates to the mock request's URL.Query.
func (m *MockContext) Query(key string) string {
	if m.req == nil {
		return ""
	}
	return m.req.URL.Query().Get(key)
}

// QueryDefault delegates to the mock request's URL.Query with a fallback.
func (m *MockContext) QueryDefault(key, def string) string {
	if v := m.Query(key); v != "" {
		return v
	}
	return def
}

// QueryInt parses a query value as int with a fallback default.
func (m *MockContext) QueryInt(key string, def int) int {
	v := m.Query(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// BindQuery copies the stashed query body (see SetQueryBody) into v, or
// returns the injected error (see WithBindQueryError). With neither set, it
// is a no-op for backwards compatibility.
func (m *MockContext) BindQuery(v any) error {
	if m.bindQueryErr != nil {
		return m.bindQueryErr
	}
	if m.bindQueryBody != nil {
		return copyInto(v, m.bindQueryBody)
	}
	return nil
}

// SetBody stashes a value that subsequent calls to Bind copy into their
// target. v should be the same struct type the handler binds.
func (m *MockContext) SetBody(v any) { m.bindBody = v }

// SetQueryBody stashes a value that subsequent calls to BindQuery copy into
// their target.
func (m *MockContext) SetQueryBody(v any) { m.bindQueryBody = v }

// WithBindError configures Bind to return err instead of binding.
func (m *MockContext) WithBindError(err error) { m.bindErr = err }

// WithBindQueryError configures BindQuery to return err instead of binding.
func (m *MockContext) WithBindQueryError(err error) { m.bindQueryErr = err }

// copyInto round-trips src through JSON into dst (a pointer). Reflection
// would require src and dst to share concrete types; JSON tolerates field
// subsets and is sufficient for test ergonomics.
func copyInto(dst, src any) error {
	if dst == nil {
		return nil
	}
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return nil
	}
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// Paginate returns a zero-value ListQuery; tests can swap the request to
// drive realistic pagination behavior.
func (m *MockContext) Paginate(defaultPerPage, maxPerPage int) httpifc.ListQuery {
	if m.req == nil {
		return httpifc.ListQuery{}
	}
	q := httpifc.ParseListQuery(m.req)
	q.Normalize(defaultPerPage, maxPerPage)
	return q
}

// Bind copies the stashed body (see SetBody) into v, or returns the
// injected error (see WithBindError). With neither set, it is a no-op for
// backwards compatibility.
func (m *MockContext) Bind(v any) error {
	if m.bindErr != nil {
		return m.bindErr
	}
	if m.bindBody != nil {
		return copyInto(v, m.bindBody)
	}
	return nil
}

// JSON is a mock implementation that always returns nil.
func (m *MockContext) JSON(code int, v any) error {
	return nil
}

// String is a mock implementation that always returns nil.
func (m *MockContext) String(code int, s string) error {
	return nil
}

// Set stores a value in the mock context.
func (m *MockContext) Set(key string, val any) {
	m.values[key] = val
}

// Get retrieves a value from the mock context.
func (m *MockContext) Get(key string) any {
	return m.values[key]
}

// SetRequestContainer sets the request-scoped DI di.
func (m *MockContext) SetRequestContainer(c *di.Container) {
	m.reqCont = c
}

// GetRequestContainer returns the request-scoped DI di.
func (m *MockContext) GetRequestContainer() *di.Container {
	return m.reqCont
}

// HTTP response helpers (mock implementations)

func (m *MockContext) OK(v any) error                                            { return nil }
func (m *MockContext) Created(v any) error                                       { return nil }
func (m *MockContext) Accepted(v any) error                                      { return nil }
func (m *MockContext) NoContent() error                                          { return nil }
func (m *MockContext) List(items any) error                                      { return nil }
func (m *MockContext) Paginated(items any, page, perPage int, total int64) error { return nil }
func (m *MockContext) BadRequest(msg ...string) error                            { return nil }
func (m *MockContext) Unauthorized(msg ...string) error                          { return nil }
func (m *MockContext) Forbidden(msg ...string) error                             { return nil }
func (m *MockContext) NotFound(msg ...string) error                              { return nil }
func (m *MockContext) MethodNotAllowed(msg ...string) error                      { return nil }
func (m *MockContext) NotAcceptable(msg ...string) error                         { return nil }
func (m *MockContext) RequestTimeout(msg ...string) error                        { return nil }
func (m *MockContext) Conflict(msg ...string) error                              { return nil }
func (m *MockContext) Gone(msg ...string) error                                  { return nil }
func (m *MockContext) PreconditionFailed(msg ...string) error                    { return nil }
func (m *MockContext) PayloadTooLarge(msg ...string) error                       { return nil }
func (m *MockContext) UnsupportedMediaType(msg ...string) error                  { return nil }
func (m *MockContext) UnprocessableEntity(msg ...string) error                   { return nil }
func (m *MockContext) TooManyRequests(msg ...string) error                       { return nil }
func (m *MockContext) ImATeapot(msg ...string) error                             { return nil }
func (m *MockContext) InternalServerError(msg ...string) error                   { return nil }
func (m *MockContext) NotImplemented(msg ...string) error                        { return nil }
func (m *MockContext) BadGateway(msg ...string) error                            { return nil }
func (m *MockContext) ServiceUnavailable(msg ...string) error                    { return nil }
func (m *MockContext) GatewayTimeout(msg ...string) error                        { return nil }
func (m *MockContext) HTTPVersionNotSupported(msg ...string) error               { return nil }
func (m *MockContext) Stream(reader io.Reader) error                             { return nil }

// MockLogger is a mock implementation of logger.Logger for testing.
type MockLogger struct {
	logs []string
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		logs: make([]string, 0),
	}
}

// Debug logs a debug message.
func (m *MockLogger) Debug(msg string, fields ...logger.Field) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}

// Info logs an info message.
func (m *MockLogger) Info(msg string, fields ...logger.Field) {
	m.logs = append(m.logs, "INFO: "+msg)
}

// Warn logs a warning message.
func (m *MockLogger) Warn(msg string, fields ...logger.Field) {
	m.logs = append(m.logs, "WARN: "+msg)
}

// Error logs an error message.
func (m *MockLogger) Error(msg string, fields ...logger.Field) {
	m.logs = append(m.logs, "ERROR: "+msg)
}

// LogWithContext logs a message with context.
func (m *MockLogger) LogWithContext(ctx logger.Context, msg string, fields ...logger.Field) {
	m.logs = append(m.logs, string(ctx)+": "+msg)
}

// SetDebug sets the debug level (no-op for mock).
func (m *MockLogger) SetDebug(debug bool) {}

// GetLogs returns all logged messages.
func (m *MockLogger) GetLogs() []string {
	return m.logs
}

// Clear clears all logged messages.
func (m *MockLogger) Clear() {
	m.logs = make([]string, 0)
}
