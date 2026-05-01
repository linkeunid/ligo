# Interceptors

Interceptors wrap the entire request/response cycle. They're used for logging, caching, response transformation, and performance monitoring.

## Interceptor Signature

```go
type Interceptor func(ctx Context, next HandlerFunc) error
```

## Creating an Interceptor

```go
import (
    "github.com/linkeunid/ligo/internal/core/logger"
)

func LoggingInterceptor(log logger.Logger) ligo.Interceptor {
    return func(ctx ligo.Context, next ligo.HandlerFunc) error {
        log.LogWithContext(logger.ContextRoutes, "Request started",
            logger.Field{Key: "method", Value: ctx.Request().Method},
            logger.Field{Key: "path", Value: ctx.Request().URL.Path},
        )

        err := next(ctx)

        if err != nil {
            log.Error("Request completed with error",
                logger.Field{Key: "method", Value: ctx.Request().Method},
                logger.Field{Key: "path", Value: ctx.Request().URL.Path},
                logger.Field{Key: "error", Value: err.Error()},
            )
        } else {
            log.LogWithContext(logger.ContextRoutes, "Request completed",
                logger.Field{Key: "method", Value: ctx.Request().Method},
                logger.Field{Key: "path", Value: ctx.Request().URL.Path},
            )
        }

        return err
    }
}
```

## Using Interceptors

```go
func (c *UserController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r.Group("/users"))

    // Apply interceptor
    cr.GET("/", c.List).
        Intercept(LoggingInterceptor(log))

    // Multiple interceptors
    cr.GET("/:id", c.Get).
        Intercept(LoggingInterceptor(log), CachingInterceptor(cache))
}
```

## Common Interceptor Patterns

### Caching Interceptor

```go
func CachingInterceptor(cache *Cache, log logger.Logger) ligo.Interceptor {
    return func(ctx ligo.Context, next ligo.HandlerFunc) error {
        key := ctx.Request().URL.String()

        // Check cache
        if cached, found := cache.Get(key); found {
            log.Debug("Cache hit",
                logger.Field{Key: "path", Value: ctx.Request().URL.Path},
            )
            return ctx.JSON(200, cached)
        }

        log.Debug("Cache miss",
            logger.Field{Key: "path", Value: ctx.Request().URL.Path},
        )

        // Capture response
        err := next(ctx)

        // Cache successful responses
        if err == nil {
            // Store response (implementation depends on context)
        }

        return err
    }
}
```

### Performance Monitoring Interceptor

```go
func MetricsInterceptor(metrics *Metrics, log logger.Logger) ligo.Interceptor {
    return func(ctx ligo.Context, next ligo.HandlerFunc) error {
        start := time.Now()

        err := next(ctx)

        duration := time.Since(start)
        metrics.Record(ctx.Request().URL.Path, duration, err != nil)

        log.Debug("Request metrics",
            logger.Field{Key: "path", Value: ctx.Request().URL.Path},
            logger.Field{Key: "duration_ms", Value: duration.Milliseconds()},
            logger.Field{Key: "error", Value: err != nil},
        )

        return err
    }
}
```

### Request ID Interceptor

```go
func RequestIDInterceptor(log logger.Logger) ligo.Interceptor {
    return func(ctx ligo.Context, next ligo.HandlerFunc) error {
        requestID := ctx.Request().Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = generateUUID()
        }

        ctx.Set("request_id", requestID)

        log.LogWithContext(logger.ContextRoutes, "Request with ID",
            logger.Field{Key: "request_id", Value: requestID},
            logger.Field{Key: "path", Value: ctx.Request().URL.Path},
        )

        return next(ctx)
    }
}
```

## Interceptor vs Middleware

| Aspect | Middleware | Interceptor |
|--------|-----------|-------------|
| Wraps | Before handler only | Entire request/response cycle |
| Use case | Pre-processing | Logging, caching, transformation |
| Access to response | Limited | Full access |
| Order | First = outermost | Last = outermost |

## Execution Order

Interceptors execute in reverse order (last = outermost):

```go
cr.GET("/", c.List).
    Intercept(A, B, C)

// Executes as: A -> B -> C -> handler -> C -> B -> A
```

## Combining with Guards and Pipes

```go
func (c *UserController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r.Group("/users"))

    cr.GET("/:id", c.Get).
        Guard(AuthGuard(authService)).            // Authorization
        Pipe(ParseIntPipe("id")).                   // Transform ID
        Intercept(LoggingInterceptor(log))         // Logging
}
```
