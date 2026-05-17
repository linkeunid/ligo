package echo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	echo "github.com/labstack/echo/v5"

	"github.com/linkeunid/ligo/internal/di"
	httpifc "github.com/linkeunid/ligo/internal/http"
)

// newTestContext wires up an Echo request/response pair and returns the
// context adapter under test plus the recorder so callers can inspect the
// HTTP response.
func newTestContext(t *testing.T, method, path string) (*contextAdapter, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	c := e.NewContext(req, rec)
	return newContextAdapter(c), rec
}

func TestContextAdapter_RequestAndResponse(t *testing.T) {
	ca, rec := newTestContext(t, "GET", "/foo")
	if ca.Request() == nil {
		t.Error("Request returned nil")
	}
	if ca.Response() == nil {
		t.Error("Response returned nil")
	}
	if ca.RequestContext() == nil {
		t.Error("RequestContext returned nil")
	}
	_ = rec
}

func TestContextAdapter_QueryHelpers(t *testing.T) {
	ca, _ := newTestContext(t, "GET", "/foo?name=alice&age=30")
	if got := ca.Query("name"); got != "alice" {
		t.Errorf("Query(name) = %q", got)
	}
	if got := ca.QueryDefault("missing", "fallback"); got != "fallback" {
		t.Errorf("QueryDefault fallback = %q", got)
	}
	if got := ca.QueryInt("age", 0); got != 30 {
		t.Errorf("QueryInt(age) = %d", got)
	}
	if got := ca.QueryInt("missing", 99); got != 99 {
		t.Errorf("QueryInt fallback = %d", got)
	}
}

func TestContextAdapter_SetGet(t *testing.T) {
	ca, _ := newTestContext(t, "GET", "/")
	ca.Set("key", "value")
	if got := ca.Get("key"); got != "value" {
		t.Errorf("Get(key) = %v", got)
	}
	if got := ca.Get("missing"); got != nil {
		t.Errorf("Get(missing) = %v", got)
	}
}

func TestContextAdapter_RequestContainer(t *testing.T) {
	ca, _ := newTestContext(t, "GET", "/")
	c := di.New()
	ca.SetRequestContainer(c)
	if ca.GetRequestContainer() != c {
		t.Error("request container round-trip failed")
	}
}

// errorHelperCase covers ca.errorResponse-backed methods.
type errorHelperCase struct {
	name string
	fn   func(httpifc.Context) error
	code int
}

func TestContextAdapter_ErrorHelpers(t *testing.T) {
	cases := []errorHelperCase{
		{"BadRequest", func(c httpifc.Context) error { return c.BadRequest("x") }, http.StatusBadRequest},
		{"Unauthorized", func(c httpifc.Context) error { return c.Unauthorized() }, http.StatusUnauthorized},
		{"Forbidden", func(c httpifc.Context) error { return c.Forbidden() }, http.StatusForbidden},
		{"NotFound", func(c httpifc.Context) error { return c.NotFound() }, http.StatusNotFound},
		{"MethodNotAllowed", func(c httpifc.Context) error { return c.MethodNotAllowed() }, http.StatusMethodNotAllowed},
		{"NotAcceptable", func(c httpifc.Context) error { return c.NotAcceptable() }, http.StatusNotAcceptable},
		{"RequestTimeout", func(c httpifc.Context) error { return c.RequestTimeout() }, http.StatusRequestTimeout},
		{"Conflict", func(c httpifc.Context) error { return c.Conflict() }, http.StatusConflict},
		{"Gone", func(c httpifc.Context) error { return c.Gone() }, http.StatusGone},
		{"PreconditionFailed", func(c httpifc.Context) error { return c.PreconditionFailed() }, http.StatusPreconditionFailed},
		{"PayloadTooLarge", func(c httpifc.Context) error { return c.PayloadTooLarge() }, http.StatusRequestEntityTooLarge},
		{"UnsupportedMediaType", func(c httpifc.Context) error { return c.UnsupportedMediaType() }, http.StatusUnsupportedMediaType},
		{"UnprocessableEntity", func(c httpifc.Context) error { return c.UnprocessableEntity() }, http.StatusUnprocessableEntity},
		{"TooManyRequests", func(c httpifc.Context) error { return c.TooManyRequests() }, http.StatusTooManyRequests},
		{"ImATeapot", func(c httpifc.Context) error { return c.ImATeapot() }, http.StatusTeapot},
		{"InternalServerError", func(c httpifc.Context) error { return c.InternalServerError() }, http.StatusInternalServerError},
		{"NotImplemented", func(c httpifc.Context) error { return c.NotImplemented() }, http.StatusNotImplemented},
		{"BadGateway", func(c httpifc.Context) error { return c.BadGateway() }, http.StatusBadGateway},
		{"ServiceUnavailable", func(c httpifc.Context) error { return c.ServiceUnavailable() }, http.StatusServiceUnavailable},
		{"GatewayTimeout", func(c httpifc.Context) error { return c.GatewayTimeout() }, http.StatusGatewayTimeout},
		{"HTTPVersionNotSupported", func(c httpifc.Context) error { return c.HTTPVersionNotSupported() }, http.StatusHTTPVersionNotSupported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ca, rec := newTestContext(t, "GET", "/")
			if err := tc.fn(ca); err != nil {
				t.Fatalf("%s err: %v", tc.name, err)
			}
			if rec.Code != tc.code {
				t.Errorf("%s code = %d, want %d", tc.name, rec.Code, tc.code)
			}
		})
	}
}

func TestContextAdapter_SuccessHelpers(t *testing.T) {
	successes := []struct {
		name string
		fn   func(httpifc.Context) error
		code int
	}{
		{"OK", func(c httpifc.Context) error { return c.OK(map[string]string{"k": "v"}) }, http.StatusOK},
		{"Created", func(c httpifc.Context) error { return c.Created(map[string]string{"k": "v"}) }, http.StatusCreated},
		{"Accepted", func(c httpifc.Context) error { return c.Accepted(map[string]string{"k": "v"}) }, http.StatusAccepted},
		{"NoContent", func(c httpifc.Context) error { return c.NoContent() }, http.StatusNoContent},
		{"List", func(c httpifc.Context) error { return c.List([]string{"a", "b"}) }, http.StatusOK},
		{"Paginated", func(c httpifc.Context) error { return c.Paginated([]string{"a"}, 1, 10, 1) }, http.StatusOK},
		{"JSON", func(c httpifc.Context) error { return c.JSON(http.StatusOK, "x") }, http.StatusOK},
		{"String", func(c httpifc.Context) error { return c.String(http.StatusOK, "x") }, http.StatusOK},
	}
	for _, tc := range successes {
		t.Run(tc.name, func(t *testing.T) {
			ca, rec := newTestContext(t, "GET", "/")
			if err := tc.fn(ca); err != nil {
				t.Fatalf("%s err: %v", tc.name, err)
			}
			if rec.Code != tc.code {
				t.Errorf("%s code = %d, want %d", tc.name, rec.Code, tc.code)
			}
		})
	}
}

func TestContextAdapter_Stream(t *testing.T) {
	ca, rec := newTestContext(t, "GET", "/")
	body := strings.NewReader("hello stream")
	if err := ca.Stream(body); err != nil {
		t.Fatalf("Stream err: %v", err)
	}
	if rec.Body.String() != "hello stream" {
		t.Errorf("Stream body = %q", rec.Body.String())
	}
}

// closingReader counts close calls.
type closingReader struct {
	io.Reader
	closed bool
}

func (c *closingReader) Close() error {
	c.closed = true
	return nil
}

func TestContextAdapter_StreamClosesReadCloser(t *testing.T) {
	ca, _ := newTestContext(t, "GET", "/")
	cr := &closingReader{Reader: strings.NewReader("x")}
	if err := ca.Stream(cr); err != nil {
		t.Fatalf("Stream err: %v", err)
	}
	if !cr.closed {
		t.Error("Stream did not close the io.Closer")
	}
}

func TestAdapter_Use_AppendsMiddleware(t *testing.T) {
	a := NewAdapter()
	called := false
	a.Use(func(next httpifc.HandlerFunc) httpifc.HandlerFunc {
		return func(c httpifc.Context) error {
			called = true
			return next(c)
		}
	})
	if len(a.middleware) != 1 {
		t.Errorf("middleware len = %d, want 1", len(a.middleware))
	}
	_ = called
}

func TestAdapter_RequestScopeMiddleware_AttachesChild(t *testing.T) {
	a := NewAdapter()
	parent := di.New()
	a.SetContainer(parent)

	ca, _ := newTestContext(t, "GET", "/")
	mw := a.requestScopeMiddleware()
	got := mw(func(c httpifc.Context) error {
		if c.GetRequestContainer() == nil {
			t.Error("request container not set")
		}
		return nil
	})
	if err := got(ca); err != nil {
		t.Fatalf("middleware err: %v", err)
	}
}
