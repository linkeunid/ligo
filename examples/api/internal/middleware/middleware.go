package middleware

import (
	"fmt"

	"github.com/linkeunid/ligo"
)

// LoggingMiddleware logs each request
func LoggingMiddleware(next ligo.HandlerFunc) ligo.HandlerFunc {
	return func(ctx ligo.Context) error {
		fmt.Printf("Request: %s %s\n", ctx.Request().Method, ctx.Request().URL.Path)
		return next(ctx)
	}
}