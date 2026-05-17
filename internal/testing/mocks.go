package testing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"

	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/di"
	httpifc "github.com/linkeunid/ligo/internal/http"
)

// MockContext is a mock Adapter implementation used by tests.
//
// Bind/BindQuery defaults are no-ops. To drive realistic binding in tests:
//   - SetBody(v) stashes a value that Bind copies into its target via reflect.
//   - SetQueryBody(v) does the same for BindQuery.
//   - WithBindError(err) / WithBindQueryError(err) inject failures.
//
// Wrap returns a *httpifc.Context backed by this MockContext for tests
// that need to invoke a HandlerFunc / Pipe / Guard.
type MockContext struct {
	values        map[string]any
	req           *http.Request
	resp          http.ResponseWriter
	reqCont       *di.Container
	bindBody      any
	bindErr       error
	bindQueryBody any
	bindQueryErr  error

	// LastJSONCode/LastJSONBody record the most recent JSON call.
	LastJSONCode int
	LastJSONBody any
	JSONErr      error

	// LastStringCode/LastStringBody record the most recent String call.
	LastStringCode int
	LastStringBody string
	StringErr      error

	// LastStreamBody captures bytes passed to Stream (read via io.ReadAll).
	LastStreamBody string
	StreamErr      error
}

// NewMockContext creates a new mock context for testing.
func NewMockContext() *MockContext {
	return &MockContext{
		values: make(map[string]any),
		req:    httptest.NewRequest("GET", "/", nil),
		resp:   httptest.NewRecorder(),
	}
}

// Wrap returns a *Context whose Adapter is this MockContext. Use this
// when calling a HandlerFunc / Pipe / Guard that expects *Context.
func (m *MockContext) Wrap() *httpifc.Context {
	return httpifc.NewContext(m)
}

// SetRequest replaces the embedded request. Useful for tests that need
// to drive query/body parsing through the helper methods on *Context.
func (m *MockContext) SetRequest(r *http.Request) { m.req = r }

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

// JSON records the call and returns m.JSONErr.
func (m *MockContext) JSON(code int, v any) error {
	m.LastJSONCode = code
	m.LastJSONBody = v
	return m.JSONErr
}

// String records the call and returns m.StringErr.
func (m *MockContext) String(code int, s string) error {
	m.LastStringCode = code
	m.LastStringBody = s
	return m.StringErr
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

// Stream records the bytes from reader and returns m.StreamErr.
func (m *MockContext) Stream(reader io.Reader) error {
	if reader != nil {
		b, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		m.LastStreamBody = string(b)
	}
	return m.StreamErr
}

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
