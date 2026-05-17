package http

import (
	"context"
	"errors"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/linkeunid/ligo/internal/di"
)

// timeoutMockContext is a minimal Context for interceptor tests. We deliberately
// do not embed mockBindContext (from pipes_test) — these tests only need
// Request/RequestContext and the request-container plumbing.
type timeoutMockContext struct {
	req    *nethttp.Request
	reqCtx context.Context
	cont   *di.Container
}

func (m *timeoutMockContext) Request() *nethttp.Request          { return m.req }
func (m *timeoutMockContext) Response() nethttp.ResponseWriter   { return nil }
func (m *timeoutMockContext) RequestContext() context.Context    { return m.reqCtx }
func (m *timeoutMockContext) Param(string) string                { return "" }
func (m *timeoutMockContext) Query(string) string                { return "" }
func (m *timeoutMockContext) QueryDefault(_, def string) string  { return def }
func (m *timeoutMockContext) QueryInt(_ string, def int) int     { return def }
func (m *timeoutMockContext) BindQuery(any) error                { return nil }
func (m *timeoutMockContext) Paginate(_, _ int) ListQuery        { return ListQuery{} }
func (m *timeoutMockContext) Bind(any) error                     { return nil }
func (m *timeoutMockContext) JSON(int, any) error                { return nil }
func (m *timeoutMockContext) String(int, string) error           { return nil }
func (m *timeoutMockContext) Set(string, any)                    {}
func (m *timeoutMockContext) Get(string) any                     { return nil }
func (m *timeoutMockContext) SetRequestContainer(*di.Container)  {}
func (m *timeoutMockContext) GetRequestContainer() *di.Container { return m.cont }
func (m *timeoutMockContext) OK(any) error                       { return nil }
func (m *timeoutMockContext) Created(any) error                  { return nil }
func (m *timeoutMockContext) Accepted(any) error                 { return nil }
func (m *timeoutMockContext) NoContent() error                   { return nil }
func (m *timeoutMockContext) List(any) error                     { return nil }
func (m *timeoutMockContext) Paginated(any, int, int, int64) error {
	return nil
}
func (m *timeoutMockContext) BadRequest(...string) error              { return nil }
func (m *timeoutMockContext) Unauthorized(...string) error            { return nil }
func (m *timeoutMockContext) Forbidden(...string) error               { return nil }
func (m *timeoutMockContext) NotFound(...string) error                { return nil }
func (m *timeoutMockContext) MethodNotAllowed(...string) error        { return nil }
func (m *timeoutMockContext) NotAcceptable(...string) error           { return nil }
func (m *timeoutMockContext) RequestTimeout(...string) error          { return nil }
func (m *timeoutMockContext) Conflict(...string) error                { return nil }
func (m *timeoutMockContext) Gone(...string) error                    { return nil }
func (m *timeoutMockContext) PreconditionFailed(...string) error      { return nil }
func (m *timeoutMockContext) PayloadTooLarge(...string) error         { return nil }
func (m *timeoutMockContext) UnsupportedMediaType(...string) error    { return nil }
func (m *timeoutMockContext) UnprocessableEntity(...string) error     { return nil }
func (m *timeoutMockContext) TooManyRequests(...string) error         { return nil }
func (m *timeoutMockContext) ImATeapot(...string) error               { return nil }
func (m *timeoutMockContext) InternalServerError(...string) error     { return nil }
func (m *timeoutMockContext) NotImplemented(...string) error          { return nil }
func (m *timeoutMockContext) BadGateway(...string) error              { return nil }
func (m *timeoutMockContext) ServiceUnavailable(...string) error      { return nil }
func (m *timeoutMockContext) GatewayTimeout(...string) error          { return nil }
func (m *timeoutMockContext) HTTPVersionNotSupported(...string) error { return nil }
func (m *timeoutMockContext) Stream(io.Reader) error                  { return nil }

func newTimeoutCtx(parent context.Context) *timeoutMockContext {
	req := httptest.NewRequest("GET", "/", nil).WithContext(parent)
	return &timeoutMockContext{req: req, reqCtx: parent}
}

func TestTimeoutInterceptor_ReturnsTimeoutErrorOnSlowHandler(t *testing.T) {
	i := TimeoutInterceptor(50 * time.Millisecond)
	ctx := newTimeoutCtx(context.Background())

	err := i(ctx, func(Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout in error, got: %v", err)
	}
}

func TestTimeoutInterceptor_HandlerSeesCancelledContext(t *testing.T) {
	i := TimeoutInterceptor(50 * time.Millisecond)
	ctx := newTimeoutCtx(context.Background())

	var cancelled atomic.Bool
	_ = i(ctx, func(c Context) error {
		select {
		case <-c.RequestContext().Done():
			cancelled.Store(true)
			return c.RequestContext().Err()
		case <-time.After(500 * time.Millisecond):
			return nil
		}
	})

	// The interceptor returns as soon as the timeout fires; the handler
	// goroutine is still running. Give it up to 1s to observe cancel and
	// store the flag.
	deadline := time.Now().Add(time.Second)
	for !cancelled.Load() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !cancelled.Load() {
		t.Error("handler did not observe context cancellation")
	}
}

func TestTimeoutInterceptor_PropagatesParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	ctx := newTimeoutCtx(parent)
	i := TimeoutInterceptor(time.Minute)

	cancelObserved := make(chan struct{})
	go func() {
		_ = i(ctx, func(c Context) error {
			<-c.RequestContext().Done()
			close(cancelObserved)
			return c.RequestContext().Err()
		})
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	select {
	case <-cancelObserved:
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not propagate to handler within 1s")
	}
}

func TestTimeoutInterceptor_HandlerReturnsNormallyWhenFast(t *testing.T) {
	i := TimeoutInterceptor(time.Second)
	ctx := newTimeoutCtx(context.Background())

	want := errors.New("handler error")
	err := i(ctx, func(Context) error { return want })

	if !errors.Is(err, want) {
		t.Errorf("expected handler error, got %v", err)
	}
}
