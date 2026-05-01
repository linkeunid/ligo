# Guards

Guards determine if a request should proceed. They're typically used for authorization.

## Guard Signature

```go
type Guard func(ctx Context) (bool, error)
```

Return `true` to allow the request, `false` to deny it. Return an error to stop processing with an error.

## Creating a Guard

```go
import "errors"

const ContextKeyUser = "user"

var ErrUnauthorized = errors.New("unauthorized")

type User struct {
    ID   string
    Name string
    Role string
}

func AuthGuard(auth *AuthService) ligo.Guard {
    return func(ctx ligo.Context) (bool, error) {
        token := ctx.Request().Header.Get("Authorization")
        if token == "" {
            return false, nil
        }

        user, err := auth.Validate(token)
        if err != nil {
            return false, nil
        }

        // Store user in context for downstream handlers
        ctx.Set(ContextKeyUser, user)
        return true, nil
    }
}
```

## Using Guards

```go
func (c *UserController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r.Group("/users"))

    // Public route - no guards
    cr.GET("/public", c.PublicHandler)

    // Protected route - requires auth
    cr.GET("/:id", c.Get).
        Guard(AuthGuard(authService))

    // Multiple guards - all must pass
    cr.DELETE("/:id", c.Delete).
        Guard(AuthGuard(authService), AdminGuard())
}
```

## Guard Execution

Guards execute in the order specified. All guards must return `true` for the request to proceed.

```go
// Both AuthGuard and AdminGuard must pass
cr.POST("/admin", c.AdminAction).
    Guard(AuthGuard(authService), AdminGuard())
```

If any guard returns `false` or an error, the chain stops and the error is returned.

## Storing Context Data

Define context keys as constants to avoid typos:

```go
const (
    ContextKeyUser   = "user"
    ContextKeyUserID = "user_id"
)

func AuthGuard() ligo.Guard {
    return func(ctx ligo.Context) (bool, error) {
        user := validateToken(...)
        ctx.Set(ContextKeyUser, user)
        ctx.Set(ContextKeyUserID, user.ID)
        return true, nil
    }
}

// Later in the handler
func (c *Controller) Get(ctx ligo.Context) error {
    user, ok := ctx.Get(ContextKeyUser).(*User)
    if !ok {
        return ctx.JSON(401, map[string]string{"error": "unauthorized"})
    }
    // ...
}
```

## Error Handling with Guards

Use custom error types for different guard failures:

```go
var (
    ErrUnauthorized = errors.New("unauthorized")
    ErrForbidden   = errors.New("forbidden")
)

func AdminGuard() ligo.Guard {
    return func(ctx ligo.Context) (bool, error) {
        user, ok := ctx.Get(ContextKeyUser).(*User)
        if !ok || user.Role != "admin" {
            return false, ErrForbidden
        }
        return true, nil
    }
}

// In exception filter
func ExceptionFilter(err error, ctx ligo.Context) error {
    if errors.Is(err, ErrUnauthorized) {
        return ctx.JSON(401, map[string]string{"error": "Unauthorized"})
    }
    if errors.Is(err, ErrForbidden) {
        return ctx.JSON(403, map[string]string{"error": "Forbidden"})
    }
    return ctx.JSON(500, map[string]string{"error": "Internal Server Error"})
}
```
