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

// timeoutMockAdapter is a minimal Adapter for interceptor tests. We deliberately
// do not embed mockBindContext (from pipes_test) — these tests only need
// Request/RequestContext and the request-container plumbing.
type timeoutMockAdapter struct {
	req    *nethttp.Request
	reqCtx context.Context
	cont   *di.Container
}

func (m *timeoutMockAdapter) Request() *nethttp.Request          { return m.req }
func (m *timeoutMockAdapter) Response() nethttp.ResponseWriter   { return nil }
func (m *timeoutMockAdapter) RequestContext() context.Context    { return m.reqCtx }
func (m *timeoutMockAdapter) Param(string) string                { return "" }
func (m *timeoutMockAdapter) Query(string) string                { return "" }
func (m *timeoutMockAdapter) BindQuery(any) error                { return nil }
func (m *timeoutMockAdapter) Bind(any) error                     { return nil }
func (m *timeoutMockAdapter) JSON(int, any) error                { return nil }
func (m *timeoutMockAdapter) String(int, string) error           { return nil }
func (m *timeoutMockAdapter) Set(string, any)                    {}
func (m *timeoutMockAdapter) Get(string) any                     { return nil }
func (m *timeoutMockAdapter) SetRequestContainer(*di.Container)  {}
func (m *timeoutMockAdapter) GetRequestContainer() *di.Container { return m.cont }
func (m *timeoutMockAdapter) Stream(io.Reader) error             { return nil }

func newTimeoutCtx(parent context.Context) *Context {
	req := httptest.NewRequest("GET", "/", nil).WithContext(parent)
	return NewContext(&timeoutMockAdapter{req: req, reqCtx: parent})
}

func TestTimeoutInterceptor_ReturnsTimeoutErrorOnSlowHandler(t *testing.T) {
	i := TimeoutInterceptor(50 * time.Millisecond)
	ctx := newTimeoutCtx(context.Background())

	err := i(ctx, func(*Context) error {
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
	_ = i(ctx, func(c *Context) error {
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
		_ = i(ctx, func(c *Context) error {
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
	err := i(ctx, func(*Context) error { return want })

	if !errors.Is(err, want) {
		t.Errorf("expected handler error, got %v", err)
	}
}
