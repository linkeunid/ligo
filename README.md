# Ligo

A modular Go framework with lightweight dependency injection, inspired by NestJS.

[![Go Version](https://img.shields.io/badge/go-1.25.9-blue)](https://go.dev/dl)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Features

- **Modular Architecture** - Organize code into self-contained modules
- **Dependency Injection** - Automatic dependency resolution with zero boilerplate
- **HTTP Routing** - Adapter-agnostic router interface with Echo v5 adapter
- **Middleware Support** - Global, module-level, and route-level middleware
- **Request-Scoped Data** - Share data across middleware chain
- **Lifecycle Hooks** - OnStart and OnStop hooks for graceful shutdown
- **Type-Safe** - Full type safety with generics
- **Production Ready** - Thread-safe with cycle detection and comprehensive error handling

## Installation

```bash
go get github.com/linkeunid/ligo
```

## Quick Start

```go
package main

import (
    "github.com/linkeunid/ligo"
    "github.com/linkeunid/ligo/adapters/echo"
)

func main() {
    router := echo.NewAdapter()
    app := ligo.New(
        ligo.WithRouter(router),
        ligo.WithAddr(":8080"),
    )

    app.Register(
        ligo.NewModule("hello",
            ligo.Providers(ligo.Value("Hello, World!")),
            ligo.Controllers(func(msg string) ligo.Controller {
                return &helloController{msg: msg}
            }),
        ),
    )

    app.Run()
}

type helloController struct { msg string }

func (c *helloController) Routes(r ligo.Router) {
    r.Handle("GET", "/", func(ctx ligo.Context) error {
        return ctx.String(200, c.msg)
    })
}
```

Run it:

```bash
curl http://localhost:8080/
# Hello, World!
```

## Core Concepts

### App

The `App` is the main application instance that orchestrates modules, providers, and the HTTP server.

```go
app := ligo.New(
    ligo.WithRouter(router),
    ligo.WithAddr(":8080"),
    ligo.WithMiddleware(RecoveryMiddleware, LoggingMiddleware),
    ligo.OnStart(func(ctx context.Context) error {
        log.Println("Application starting...")
        return nil
    }),
    ligo.OnStop(func(ctx context.Context) error {
        log.Println("Application stopping...")
        return nil
    }),
)

app.Register(module1, module2)
app.Run()
```

### Module

A `Module` is a self-contained unit of functionality that bundles providers, controllers, and middleware.

```go
func UserModule() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(
            ligo.Factory[*UserRepo](NewUserRepo),
            ligo.Factory[*UserService](NewUserService),
        ),
        ligo.Middlewares(func(auth *AuthService) ligo.Middleware {
            return AuthMiddleware(auth)
        }),
        ligo.Controllers(func(svc *UserService) ligo.Controller {
            return NewUserController(svc)
        }),
    )
}
```

### Providers

Providers define how dependencies are created and injected.

**Value** - Pre-built singleton:

```go
ligo.Value("config string")
```

**Factory** - Factory function with auto-injected dependencies:

```go
ligo.Factory[*UserService](NewUserService)
// NewUserService receives *UserRepo automatically
```

**Transient** - New instance per resolve:

```go
ligo.Transient[*RequestContext](NewRequestContext)
```

**Export** - Make provider visible to sibling modules:

```go
ligo.Export(ligo.Factory[*AuthService](NewAuthService))
```

### Controllers

Controllers handle HTTP requests. They receive dependencies via constructor injection.

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

func (c *UserController) Get(ctx ligo.Context) error {
    id := ctx.Param("id")
    user, err := c.svc.Find(id)
    if err != nil {
        return ctx.JSON(404, map[string]string{"error": "not found"})
    }
    return ctx.JSON(200, user)
}
```

### Middleware

Middleware can be registered globally, per-module, or per-route.

```go
// Global middleware
app := ligo.New(
    ligo.WithRouter(router),
    ligo.WithMiddleware(RecoveryMiddleware, LoggingMiddleware),
)

// Module-level middleware with DI
func AuthModule() ligo.Module {
    return ligo.NewModule("auth",
        ligo.Providers(
            ligo.Factory[*AuthService](NewAuthService),
        ),
        ligo.Middlewares(func(auth *AuthService) ligo.Middleware {
            return AuthMiddleware(auth)
        }),
        ligo.Controllers(func(svc *UserService) ligo.Controller {
            return NewUserController(svc)
        }),
    )
}

// Middleware implementation
func AuthMiddleware(auth *AuthService) ligo.Middleware {
    return func(next ligo.HandlerFunc) ligo.HandlerFunc {
        return func(ctx ligo.Context) error {
            token := ctx.Request().Header.Get("Authorization")
            user, err := auth.Validate(token)
            if err != nil {
                return ctx.String(401, "Unauthorized")
            }
            ctx.Set("user", user) // Request-scoped data
            return next(ctx)
        }
    }
}
```

### Request-Scoped Data

Share data across the middleware chain using `Set` and `Get`:

```go
// Middleware
ctx.Set("user", user)

// Handler
user := ctx.Get("user").(*User)
```

## API Reference

### App Options

| Option | Description |
|--------|-------------|
| `WithRouter(router)` | Set the HTTP router |
| `WithAddr(addr)` | Set the server address |
| `WithMiddleware(mw...)` | Add global middleware |
| `WithLogger(logger)` | Set the logger |
| `OnStart(fn)` | Add startup hook |
| `OnStop(fn)` | Add shutdown hook |

### Module Options

| Option | Description |
|--------|-------------|
| `Providers(providers...)` | Add providers to the module |
| `Controllers(constructors...)` | Add controller constructors |
| `Middlewares(constructors...)` | Add middleware constructors |
| `Imports(modules...)` | Import child modules |

### Provider Types

| Type | Description |
|------|-------------|
| `Value[T](instance)` | Pre-built singleton |
| `Factory[T](fn)` | Factory with auto-injection |
| `Transient[T](fn)` | New instance per resolve |
| `Export(provider)` | Make visible to sibling modules |

### Context Methods

| Method | Description |
|--------|-------------|
| `Request() *http.Request` | Get the request |
| `Response() http.ResponseWriter` | Get the response writer |
| `Param(key string) string` | Get URL parameter |
| `Bind(v any) error` | Bind request body to struct |
| `JSON(code int, v any) error` | Send JSON response |
| `String(code int, s string) error` | Send string response |
| `Set(key string, val any)` | Set request-scoped value |
| `Get(key string) any` | Get request-scoped value |

## Examples

See the [examples](examples/) directory for complete examples:

- [Basic API](examples/basic-api/) - Simple inline module example
- [Full API](examples/api/) - Modular example with user service and middleware

## DI Container Features

- **Thread-safe singleton creation** - Per-type locks via `sync.Map`
- **Transient providers** - New instance per resolve
- **Cycle detection** - Chain-based detection prevents deadlock
- **Auto-injection** - Dependencies resolved via reflection
- **Error handling** - `ErrCircularDependency`, `ErrMissingDependency`, `ErrDuplicateProvider`

## Architecture

```
ligo/
├── app.go                  # Core App struct
├── module.go               # Module re-exports
├── provider.go             # Provider types
├── router.go               # Router re-exports
├── errors.go               # Error types
├── options.go              # App options
├── adapters/
│   └── echo/router.go      # Echo v5 adapter
├── internal/
│   ├── core/
│   │   ├── container/      # DI container
│   │   ├── module/         # Module definition
│   │   ├── logger/         # NestJS-style logger
│   │   ├── lifecycle/      # App lifecycle
│   │   └── resolver/       # Interface-based resolution
│   ├── http/
│   │   ├── router.go       # Router interface
│   │   ├── context.go      # Context interface
│   │   └── binder.go       # Controller registration
│   └── testing/
│       └── app.go          # Test helpers
└── examples/
    ├── basic-api/
    └── api/
```

## License

MIT License - see [LICENSE](LICENSE) for details.
