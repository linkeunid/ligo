# Ligo Package Development Guide

This guide explains how to create new packages for Ligo, whether they're router adapters, integration packages, or utility packages.

## Package Types

There are three main ways to extend Ligo:

1. **Router Adapters** - Add support for new HTTP frameworks
2. **Internal Utilities** - Add guards, pipes, or interceptors to the core
3. **External Packages** - Separate repositories for integrations (databases, auth, etc.)

---

## Type 1: Router Adapter (HTTP Framework)

### When to Use
- You want to use a different HTTP framework (Fiber, Gin, Chi, etc.)
- You need to implement the `Router` interface

### Implementation Steps

**1. Create the adapter package:**

```bash
mkdir -p adapters/fiber
cd adapters/fiber
go mod init github.com/linkeunid/ligo/adapters/fiber
```

**2. Implement the Router interface:**

```go
// adapters/fiber/router.go
package fiber

import (
    "net/http"
    
    "github.com/gofiber/fiber/v2"
    httpifc "github.com/linkeunid/ligo/internal/http"
    "github.com/linkeunid/ligo/internal/core/container"
    "github.com/linkeunid/ligo/internal/core/logger"
)

// Adapter implements httpifc.Router using Fiber.
type Adapter struct {
    app        *fiber.App
    middleware []httpifc.Middleware
    logger     logger.Logger
    container  *container.Container
}

func NewAdapter() *Adapter {
    app := fiber.New()
    return &Adapter{
        app:       app,
        container: nil,
    }
}

func (a *Adapter) SetContainer(c *container.Container) {
    a.container = c
}

func (a *Adapter) SetLogger(log logger.Logger) {
    a.logger = log
}

func (a *Adapter) Group(prefix string) httpifc.Router {
    group := a.app.Group(prefix)
    return &groupAdapter{
        group:     group,
        adapter:   a,
        container: a.container,
    }
}

func (a *Adapter) Use(middleware ...httpifc.Middleware) {
    for _, mw := range middleware {
        a.app.Use(mw.ToFiber(a))
    }
    a.middleware = append(a.middleware, middleware...)
}

func (a *Adapter) Handle(method, path string, handler httpifc.HandlerFunc) {
    a.app.Add(method, path, func(c *fiber.Ctx) error {
        ctx := &contextAdapter{ctx: c}
        return handler(ctx)
    })
}

func (a *Adapter) Serve(addr string) error {
    return a.app.Listen(addr)
}
```

**3. Usage:**

```go
import "github.com/linkeunid/ligo/adapters/fiber"

app := ligo.New(
    ligo.WithRouter(fiber.NewAdapter()),
    ligo.WithAddr(":8080"),
)
```

---

## Type 2: Internal Utilities

### When to Use
- Adding built-in utilities to the core framework
- The utility is generic and useful for all Ligo users

### Implementation Steps

**Add to `internal/http/`:**

```go
// internal/http/myutils.go

// MyCustomPipe provides custom functionality.
//
// Example:
//
//	cr.POST("", c.Create).Pipe(ligo.MyCustomPipe())
func MyCustomPipe() Pipe {
    return Pipe{
        name: "MyCustomPipe",
        fn: func(ctx Context) error {
            // Custom logic
            return nil
        },
    }
}
```

**Re-export in root package:**

```go
// router.go

func MyCustomPipe() Pipe {
    return http.MyCustomPipe()
}
```

---

## Type 3: External Package (Separate Repository)

### When to Use
- Large feature set (database drivers, auth providers, etc.)
- Has its own dependencies
- Could be used independently
- Optional Ligo functionality

**For complete guide on creating external packages, see [External Packages Guide](external-packages.md).**

### Quick Example

```go
// External package: github.com/linkeunid/ligo-database-pgx
package databasepgx

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/linkeunid/ligo"
)

type Config struct { DSN string }

func Module(config Config) ligo.Module {
    return ligo.NewModule("database",
        ligo.Providers(
            ligo.Factory[*pgxpool.Pool](func() *pgxpool.Pool {
                pool, _ := pgxpool.New(context.Background(), config.DSN)
                return pool
            }),
        ),
    )
}
```

---

## Understanding Dynamic Modules

### When to Use Dynamic Modules

Dynamic modules are useful when:

1. **Runtime Configuration** - Module needs configuration at runtime
2. **Multiple Instances** - Same module type with different configs
3. **Conditional Registration** - Module structure depends on config

### Pattern: Dynamic vs Regular

**Regular Module:**
```go
func Module() ligo.Module {
    return ligo.NewModule("static",
        ligo.Providers(
            ligo.Value("static-value"),
        ),
    )
}
```

**Dynamic Module:**
```go
func RegisterDynamicModule(config string) ligo.Module {
    return ligo.NewModule("dynamic-wrapper",
        ligo.Dynamic(
            func(opts ...any) ligo.Module {
                // Config is passed through opts
                return ligo.NewModule("dynamic-instance",
                    ligo.Providers(
                        ligo.Value(config),
                    ),
                )
            },
            config,
        ),
    )
}
```

---

## Best Practices

### 1. Package Naming

```
✅ Good:
- ligo-database-pgx
- ligo-cache-redis
- ligo-auth0-integration

❌ Avoid:
- ligo-fiber (Use adapters/fiber)
- ligo-redis (Add context prefix)
```

### 2. Module Structure

**DO:** Keep modules focused

```go
✅ Good: Single concern
func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(...),
        ligo.Controllers(...),
    )
}
```

**DON'T:** Monolithic modules

```go
❌ Bad: Everything
func Module() ligo.Module {
    return ligo.NewModule("everything",
        ligo.Providers(user, auth, database...),
        ligo.Controllers(user, admin...),
        ligo.Middlewares(auth, logging...),
    )
}
```

### 3. Dependency Injection

**Use Factory for dependencies:**

```go
✅ Good: Factory with DI
func Module(dbModule ligo.Module) ligo.Module {
    return ligo.NewModule("user",
        ligo.Imports(dbModule),
        ligo.Providers(
            ligo.Factory[*UserService](func(db *Database) *UserService {
                return NewUserService(db)
            }),
        ),
    )
}
```

### 4. Error Handling

**Use typed errors:**

```go
var (
    ErrConfig = errors.New("config error")
    ErrConnection = errors.New("connection failed")
)
```

### 5. Configuration

**Use structs for config:**

```go
✅ Good: Struct config
type Config struct {
    Host     string
    Port     int
    Username string
}

func Module(config Config) ligo.Module {
    // ...
}
```

---

## Testing Your Package

### 1. Unit Tests

```go
// module_test.go
package mypackage

import "testing"

func TestModule(t *testing.T) {
    module := Module(Config{
        Host: "localhost",
        Port: 8080,
    })
    
    if module.Name != "mypackage" {
        t.Errorf("expected name 'mypackage', got %s", module.Name)
    }
}
```

### 2. Integration Tests with Ligo

```go
// integration_test.go
package mypackage_test

import (
    "testing"
    
    "github.com/linkeunid/ligo"
    "github.com/linkeunid/ligo/adapters/echo"
    "github.com/linkeunid/ligo/mypackage"
)

func TestIntegration(t *testing.T) {
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":0"),
    )
    
    app.Register(
        mypackage.Module(mypackage.Config{
            Host: "localhost",
            Port: 8080,
        }),
    )
    
    // Test that app starts without error
}
```

---

## Common Patterns

### Pattern 1: Router Adapter

```go
// adapters/myframework/router.go
package myframework

import (
    "github.com/linkeunid/ligo/internal/http"
)

type Adapter struct {
    // ...
}

func NewAdapter() *Adapter {
    return &Adapter{}
}

// Implement Router interface
func (a *Adapter) Group(prefix string) http.Router { /* ... */ }
func (a *Adapter) Use(middleware ...http.Middleware) { /* ... */ }
func (a *Adapter) Handle(method, path string, handler http.HandlerFunc) { /* ... */ }
func (a *Adapter) Serve(addr string) error { /* ... */ }
```

### Pattern 2: Utility Package

```go
// utils/validation/pipe.go
package validation

import "github.com/linkeunid/ligo"

func EmailValidationPipe() ligo.Pipe {
    return ligo.Pipe{
        name: "EmailValidationPipe",
        fn: func(ctx ligo.Context) error {
            email := ctx.Param("email")
            if !isValidEmail(email) {
                return ctx.BadRequest("invalid email")
            }
            return nil
        },
    }
}
```

### Pattern 3: Integration Package

```go
// integration/stripe/module.go
package stripe

import (
    "github.com/linkeunid/ligo"
    "github.com/stripe/stripe-go"
)

type Config struct {
    APIKey string
}

func Module(config Config) ligo.Module {
    stripe.Key = config.APIKey
    
    return ligo.NewModule("stripe",
        ligo.Providers(
            ligo.Factory[*stripe.Client](func() *stripe.Client {
                return stripe.New(config.APIKey, nil)
            }),
        ),
        ligo.Controllers(
            func(client *stripe.Client) ligo.Controller {
                return &PaymentController{client: client}
            },
        ),
    )
}
```

---

## Summary

| Package Type | Location | For Core Contributors? | Example |
|--------------|----------|------------------------|---------|
| **Router Adapter** | `adapters/*` | Yes | `adapters/fiber` |
| **Internal Utility** | `internal/http/*` | Yes | `ValidationPipe` |
| **External Package** | Separate repo | No | `database-pgx` |

### Key Takeaways

1. **Router adapters** → Implement `Router` interface, place in `adapters/`
2. **Internal utilities** → Add to `internal/http/`, re-export in root
3. **External packages** → Separate repos, use only public API, provide module factory
4. **Dynamic modules** → For runtime configuration, not a replacement for regular modules

### See Also

- **External packages**: See [External Packages Guide](external-packages.md) for creating third-party integration packages
- **Microservices**: See [Microservices Documentation](microservices.md) for building distributed systems with Ligo
