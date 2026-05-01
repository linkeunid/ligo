package http

import (
	"context"
	"fmt"
	"time"
)

// TimeoutInterceptor creates an interceptor that enforces a timeout on request handling.
// Usage: Intercept(TimeoutInterceptor(5 * time.Second))
func TimeoutInterceptor(timeout time.Duration) Interceptor {
	return func(ctx Context, next HandlerFunc) error {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan error, 1)

		go func() {
			done <- next(ctx)
		}()

		select {
		case err := <-done:
			return err
		case <-timeoutCtx.Done():
			return fmt.Errorf("request timeout after %v", timeout)
		}
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
