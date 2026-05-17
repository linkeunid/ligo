package echo

import (
	"io"
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

func TestContextAdapter_Query(t *testing.T) {
	ca, _ := newTestContext(t, "GET", "/foo?name=alice")
	if got := ca.Query("name"); got != "alice" {
		t.Errorf("Query(name) = %q", got)
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
		return func(c *httpifc.Context) error {
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
	got := mw(func(c *httpifc.Context) error {
		if c.GetRequestContainer() == nil {
			t.Error("request container not set")
		}
		return nil
	})
	if err := got(httpifc.NewContext(ca)); err != nil {
		t.Fatalf("middleware err: %v", err)
	}
}
