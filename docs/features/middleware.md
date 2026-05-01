# Middleware

Middleware can be registered globally, per-module, or per-route.

## Middleware Signature

```go
type Middleware func(HandlerFunc) HandlerFunc
```

## Global Middleware

Applied to all routes:

```go
app := ligo.New(
    ligo.WithRouter(router),
    ligo.WithMiddleware(
        RecoveryMiddleware,
        LoggingMiddleware,
        CORSMiddleware,
    ),
)
```

## Module-Level Middleware

Applied to all routes in a module, with dependency injection:

```go
func AuthModule() ligo.Module {
    return ligo.NewModule("auth",
        ligo.Providers(
            ligo.Factory[*AuthService](NewAuthService),
        ),
        ligo.Middlewares(
            // Receives *AuthService via DI
            func(auth *AuthService) ligo.Middleware {
                return AuthMiddleware(auth)
            },
        ),
        ligo.Controllers(...),
    )
}
```

## Route-Level Middleware

Applied to specific routes using groups:

```go
func (c *UserController) Routes(r ligo.Router) {
    // Public routes
    r.Handle("GET", "/users", c.List)

    // Protected routes
    protected := r.Group("/users")
    protected.Use(AuthMiddleware)
    protected.Handle("POST", "/", c.Create)
    protected.Handle("PUT", "/:id", c.Update)
}
```

## Middleware Examples

### Logging Middleware

```go
func LoggingMiddleware(next ligo.HandlerFunc) ligo.HandlerFunc {
    return func(ctx ligo.Context) error {
        start := time.Now()
        err := next(ctx)
        log.Printf("%s %s %v", ctx.Request().Method, ctx.Request().URL.Path, time.Since(start))
        return err
    }
}
```

### Recovery Middleware

```go
func RecoveryMiddleware(next ligo.HandlerFunc) ligo.HandlerFunc {
    return func(ctx ligo.Context) error {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("panic recovered: %v", r)
                ctx.String(500, "Internal Server Error")
            }
        }()
        return next(ctx)
    }
}
```

### Auth Middleware with DI

```go
func AuthMiddleware(auth *AuthService) ligo.Middleware {
    return func(next ligo.HandlerFunc) ligo.HandlerFunc {
        return func(ctx ligo.Context) error {
            token := ctx.Request().Header.Get("Authorization")
            if token == "" {
                return ctx.String(401, "Missing authorization header")
            }

            user, err := auth.Validate(token)
            if err != nil {
                return ctx.String(401, "Invalid token")
            }

            // Store user in context for downstream handlers
            ctx.Set("user", user)
            return next(ctx)
        }
    }
}
```

## Request-Scoped Data

Share data across middleware chain:

```go
// Middleware sets data
ctx.Set("user", user)
ctx.Set("requestId", uuid.New().String())

// Handler retrieves data
user := ctx.Get("user").(*User)
requestId := ctx.Get("requestId").(string)
```

## Middleware Execution Order

Middleware is applied in reverse order (last middleware wraps first):

```go
// This order:
WithMiddleware(MW1, MW2, MW3)

// Executes as:
Request -> MW3 -> MW2 -> MW1 -> Handler
Response <- MW3 <- MW2 <- MW1 <- Handler
```
