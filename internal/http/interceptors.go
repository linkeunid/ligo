package http

import (
	"time"

	"github.com/linkeunid/ligo/internal/http/interceptors"
)

// Re-exported interceptor functions

// TimeoutInterceptor creates an interceptor that enforces a timeout on request handling.
// Usage: Intercept(TimeoutInterceptor(5 * time.Second))
func TimeoutInterceptor(timeout time.Duration) Interceptor {
	i := interceptors.TimeoutInterceptor(timeout)
	return func(ctx Context, next HandlerFunc) error {
		// Wrap context to match interceptors.Context interface
		wrappedCtx := &contextWrapper{ctx: ctx}
		// Wrap handler to match interceptors.HandlerFunc
		wrappedNext := func(ic interceptors.Context) error {
			return next(ic.(*contextWrapper).ctx)
		}
		return i(wrappedCtx, wrappedNext)
	}
}

// LoggingInterceptor creates an interceptor that logs request details.
// The logFunc callback receives the start time, context, and any error.
// Usage: Intercept(LoggingInterceptor(func(start time.Time, ctx Context, err error) { ... }))
func LoggingInterceptor(logFunc func(start time.Time, ctx Context, err error)) Interceptor {
	return func(ctx Context, next HandlerFunc) error {
		start := time.Now()
		err := next(ctx)
		if logFunc != nil {
			logFunc(start, ctx, err)
		}
		return err
	}
}

// contextWrapper wraps http.Context to implement interceptors.Context
type contextWrapper struct {
	ctx Context
}

func (w *contextWrapper) Request() any {
	return w.ctx.Request()
}
