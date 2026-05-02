# Exception Filters

Exception filters handle errors and convert them to HTTP responses. They catch errors from guards, pipes, handlers, and interceptors.

## Exception Filter Signature

```go
type ExceptionFilter func(error, Context) error
```

## Creating an Exception Filter

```go
func HttpExceptionFilter(err error, ctx ligo.Context) error {
    if err != nil {
        // Convert error to HTTP response
        return ctx.JSON(500, map[string]string{
            "error": err.Error(),
        })
    }
    return err
}
```

## Using Exception Filters

```go
func (c *UserController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r.Group("/users"))

    // Single filter
    cr.GET("/", c.List).
        Filter(ExceptionFilter)

    // Multiple filters - executed in order
    cr.POST("/", c.Create).
        Guard(AuthGuard()).
        Pipe(ValidationPipe()).
        Filter(UnauthorizedFilter, ValidationErrorFilter, ExceptionFilter)
}
```

## Exception Filter Execution

Filters execute in the order specified. If a filter returns an error, the chain stops:

```go
// Execution order on error:
// 1. UnauthorizedFilter - if guard denied, returns 401, chain stops
// 2. ValidationErrorFilter - if pipe failed, returns 400, chain stops
// 3. ExceptionFilter - catches all other errors, returns 500
cr.POST("/", c.Create).
    Filter(UnauthorizedFilter, ValidationErrorFilter, ExceptionFilter)
```

## Common Exception Filter Patterns

### HTTP Status Code Filter

```go
func HttpExceptionFilter(err error, ctx ligo.Context) error {
    if err == nil {
        return nil
    }
    return ctx.JSON(500, map[string]string{"error": err.Error()})
}
```

### Guard Denial Filter

```go
func UnauthorizedFilter(err error, ctx ligo.Context) error {
    if err != nil && err.Error() == "guard denied access" {
        return ctx.JSON(401, map[string]string{"error": "Unauthorized"})
    }
    return err // Pass to next filter
}
```

### Validation Error Filter

Use `errors.Is` and `errors.As` to detect error types from pipes:

```go
import (
    "errors"
    "github.com/go-playground/validator/v10"
    "github.com/linkeunid/ligo"
)

func ValidationErrorFilter(err error, ctx ligo.Context) error {
    if err == nil {
        return nil
    }
    var ve validator.ValidationErrors
    switch {
    case errors.Is(err, ligo.ErrBadRequest):
        // Pipe parameter parsing failed (UUIDPipe, ParseIntPipe, etc.)
        return ctx.JSON(400, map[string]string{"error": "Bad Request"})
    case errors.As(err, &ve):
        // ValidationPipe struct tag validation failed
        return ctx.JSON(422, map[string]string{"error": "Unprocessable Entity"})
    }
    return err // Pass to next filter
}
```

## ErrBadRequest Sentinel

`ligo.ErrBadRequest` is a sentinel error wrapped by all built-in param-parsing pipes
(`UUIDPipe`, `ParseIntPipe`, `ParseBoolPipe`) when a path parameter is invalid.

Detect it with `errors.Is`:

```go
case errors.Is(err, ligo.ErrBadRequest):
    return ctx.JSON(400, map[string]string{"error": "Bad Request"})
```

`ValidationPipe` also wraps `ligo.ErrBadRequest` on bind failures, but produces a
`validator.ValidationErrors` on struct tag failures — check `errors.As` for that case
and map it to 422 (Unprocessable Entity).

### Custom Error Types

```go
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s %s not found", e.Resource, e.ID)
}

func NotFoundFilter(err error, ctx ligo.Context) error {
    if notFound, ok := err.(*NotFoundError); ok {
        return ctx.JSON(404, map[string]string{
            "error": fmt.Sprintf("%s not found", notFound.Resource),
        })
    }
    return err
}
```

## Exception Filter vs Middleware

| Aspect | Middleware | Exception Filter |
|--------|-----------|------------------|
| Wraps | Before handler only | After error occurs |
| Use case | Pre-processing | Error handling |
| Access to errors | Limited | Full access |
| Execution | Always | Only on error |

## Combining with Guards, Pipes, Interceptors

```go
func (c *UserController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r.Group("/users"))

    cr.POST("/", c.Create).
        Guard(AuthGuard()).                    // Can return error
        Pipe(ValidationPipe()).                 // Can return error
        Intercept(LoggingInterceptor()).        // Can return error
        Filter(                                // Catches all errors
            UnauthorizedFilter,                 // Handles guard errors
            ValidationErrorFilter,              // Handles pipe errors
            ExceptionFilter,                    // Handles everything else
        )
}
```

## Error Flow

```
Request → Guard (deny?) → Pipe (validate?) → Interceptor → Handler
           ↓ (error)        ↓ (error)           ↓ (error)    ↓ (error)
           ExceptionFilter ←─────────────────────────────────────
           ↓ (converts to HTTP response)
           Response
```
