package http

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// TimeoutInterceptor creates an interceptor that enforces a timeout on request
// handling. The timeout context derives from the per-request context returned
// by ctx.RequestContext(), so client disconnects, parent interceptor timeouts,
// and graceful-shutdown signals all propagate. The wrapped Context passed into
// the next handler exposes the timeoutCtx via RequestContext() and Request();
// handlers should read either when issuing cancellable downstream calls.
//
// Caveat: when the timeout fires before next returns, this interceptor returns
// immediately. The handler goroutine is best-effort: it is not forcibly
// stopped. If the handler ignores cancellation it keeps running until it
// returns naturally — captured resources stay reachable until then. Handlers
// that wrap I/O in cancellable contexts (database drivers, http.Client with
// the request context) shut down promptly; tight CPU loops do not.
//
// Usage: Intercept(TimeoutInterceptor(5 * time.Second))
func TimeoutInterceptor(timeout time.Duration) Interceptor {
	return func(ctx *Context, next HandlerFunc) error {
		parent := ctx.RequestContext()
		if parent == nil {
			parent = context.Background()
		}
		timeoutCtx, cancel := context.WithTimeout(parent, timeout)
		defer cancel()

		wrapped := NewContext(&timeoutAdapter{
			Adapter: ctx.Adapter,
			req:     ctx.Request().WithContext(timeoutCtx),
			reqCtx:  timeoutCtx,
		})

		done := make(chan error, 1)
		go func() { done <- next(wrapped) }()

		select {
		case err := <-done:
			return err
		case <-timeoutCtx.Done():
			return fmt.Errorf("request timeout after %v", timeout)
		}
	}
}

// timeoutAdapter overrides Request and RequestContext to expose the
// timeout-bound context to the handler while delegating everything else to
// the underlying Adapter.
type timeoutAdapter struct {
	Adapter
	req    *http.Request
	reqCtx context.Context
}

func (w *timeoutAdapter) Request() *http.Request          { return w.req }
func (w *timeoutAdapter) RequestContext() context.Context { return w.reqCtx }

// LoggingInterceptor creates an interceptor that logs request details.
// The logFunc callback receives the start time, context, and any error.
// Usage: Intercept(LoggingInterceptor(func(start time.Time, ctx *Context, err error) { ... }))
func LoggingInterceptor(logFunc func(start time.Time, ctx *Context, err error)) Interceptor {
	return func(ctx *Context, next HandlerFunc) error {
		start := time.Now()
		err := next(ctx)
		if logFunc != nil {
			logFunc(start, ctx, err)
		}
		return err
	}
}
