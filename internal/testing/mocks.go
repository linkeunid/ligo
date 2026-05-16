package testing

import (
	"net/http"
	"net/http/httptest"

	"github.com/linkeunid/ligo/internal/di"
	"github.com/linkeunid/ligo/internal/core/logger"
)

// MockContext is a mock implementation of ligo.Context for testing.
type MockContext struct {
	values    map[string]any
	req       *http.Request
	resp      http.ResponseWriter
	reqCont   *di.Container
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

// Param returns a mock path parameter (always empty string).
func (m *MockContext) Param(key string) string {
	return ""
}

// Bind is a mock implementation that always returns nil.
func (m *MockContext) Bind(v any) error {
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

func (m *MockContext) OK(v any) error                         { return nil }
func (m *MockContext) Created(v any) error                    { return nil }
func (m *MockContext) Accepted(v any) error                   { return nil }
func (m *MockContext) NoContent() error                       { return nil }
func (m *MockContext) List(items any) error                                   { return nil }
func (m *MockContext) Paginated(items any, page, perPage int, total int64) error { return nil }
func (m *MockContext) BadRequest(msg ...string) error         { return nil }
func (m *MockContext) Unauthorized(msg ...string) error       { return nil }
func (m *MockContext) Forbidden(msg ...string) error          { return nil }
func (m *MockContext) NotFound(msg ...string) error           { return nil }
func (m *MockContext) MethodNotAllowed(msg ...string) error   { return nil }
func (m *MockContext) NotAcceptable(msg ...string) error      { return nil }
func (m *MockContext) RequestTimeout(msg ...string) error     { return nil }
func (m *MockContext) Conflict(msg ...string) error           { return nil }
func (m *MockContext) Gone(msg ...string) error               { return nil }
func (m *MockContext) PreconditionFailed(msg ...string) error { return nil }
func (m *MockContext) PayloadTooLarge(msg ...string) error    { return nil }
func (m *MockContext) UnsupportedMediaType(msg ...string) error { return nil }
func (m *MockContext) UnprocessableEntity(msg ...string) error { return nil }
func (m *MockContext) TooManyRequests(msg ...string) error      { return nil }
func (m *MockContext) InternalServerError(msg ...string) error  { return nil }
func (m *MockContext) NotImplemented(msg ...string) error      { return nil }
func (m *MockContext) BadGateway(msg ...string) error          { return nil }
func (m *MockContext) ServiceUnavailable(msg ...string) error  { return nil }
func (m *MockContext) GatewayTimeout(msg ...string) error      { return nil }
func (m *MockContext) HTTPVersionNotSupported(msg ...string) error { return nil }
func (m *MockContext) Stream(reader any) error                 { return nil }

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
