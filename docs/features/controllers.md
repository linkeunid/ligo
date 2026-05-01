# Controllers

Controllers handle HTTP requests. They receive dependencies via constructor injection.

## Creating a Controller

```go
type UserController struct {
    svc *UserService
}

func NewUserController(svc *UserService) *UserController {
    return &UserController{svc: svc}
}

func (c *UserController) Routes(r ligo.Router) {
    r.Handle("GET", "/users", c.List)
    r.Handle("GET", "/users/:id", c.Get)
    r.Handle("POST", "/users", c.Create)
}
```

## Registering Controllers

```go
func UserModule() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(
            ligo.Factory[*UserService](NewUserService),
        ),
        ligo.Controllers(func(svc *UserService) ligo.Controller {
            return NewUserController(svc)
        }),
    )
}
```

## Context Methods

The `Context` interface provides access to request and response:

| Method | Description |
|--------|-------------|
| `Request() *http.Request` | Get the request |
| `Response() http.ResponseWriter` | Get the response writer |
| `Param(key string) string` | Get URL parameter (e.g., `:id`) |
| `Bind(v any) error` | Bind request body to struct |
| `JSON(code int, v any) error` | Send JSON response |
| `String(code int, s string) error` | Send string response |
| `Set(key string, val any)` | Set request-scoped value |
| `Get(key string) any` | Get request-scoped value |

## Handler Examples

### JSON Response

```go
func (c *UserController) Get(ctx ligo.Context) error {
    id := ctx.Param("id")
    user, err := c.svc.Find(id)
    if err != nil {
        return ctx.JSON(404, map[string]string{"error": "not found"})
    }
    return ctx.JSON(200, user)
}
```

### Request Body Binding

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (c *UserController) Create(ctx ligo.Context) error {
    var req CreateUserRequest
    if err := ctx.Bind(&req); err != nil {
        return ctx.JSON(400, map[string]string{"error": err.Error()})
    }

    user, err := c.svc.Create(req)
    if err != nil {
        return ctx.JSON(500, map[string]string{"error": err.Error()})
    }

    return ctx.JSON(201, user)
}
```

### String Response

```go
func (c *HealthController) Check(ctx ligo.Context) error {
    return ctx.String(200, "OK")
}
```

### Using Request-Scoped Data

```go
func (c *UserController) Current(ctx ligo.Context) error {
    user := ctx.Get("user").(*User) // Set by middleware
    return ctx.JSON(200, user)
}
```

## Route Groups

Create groups with shared prefixes:

```go
func (c *UserController) Routes(r ligo.Router) {
    api := r.Group("/api/v1")
    api.Handle("GET", "/users", c.List)
    api.Handle("GET", "/users/:id", c.Get)
}
```
