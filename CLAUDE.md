# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Ligo** is a modular Go framework with lightweight dependency injection, inspired by NestJS.

- **Module**: `github.com/linkeunid/ligo`
- **Go version**: 1.25.9
- **License**: MIT

## Commands

```bash
# Build
go build ./...

# Run tests (all 40 pass with race detector)
go test ./...
go test -v ./...

# Run a single test
go test -run TestName ./...

# Run with race detector
go test -race ./...

# Lint
golangci-lint run

# Format
go fmt ./...

# Tidy dependencies
go mod tidy
```

## Architecture

```
ligo/
├── app.go                  # Core App struct, Run(), DI setup
├── router.go               # Re-exports: Router, HandlerFunc, Middleware, Context, Controller
├── module.go               # Re-exports: Module, NewModule, Providers, Controllers, Middlewares, Imports
├── provider.go             # Provider types: Value(), Factory(), Transient(), Export()
├── options.go              # App options: WithRouter, WithAddr, WithMiddleware, WithLogger, OnStart, OnStop
├── errors.go               # Error types: ErrAppAlreadyStarted, ErrMissingDependency, ErrCircularDependency, etc.
├── adapters/
│   └── echo/router.go      # Echo v5 adapter implementation
├── internal/
│   ├── core/
│   │   ├── container/      # DI container with thread-safe singletons and cycle detection
│   │   ├── module/         # Module definition with Middlewares support
│   │   ├── lifecycle/      # App lifecycle management
│   │   └── resolver/       # Interface-based dependency resolution
│   ├── http/
│   │   ├── router.go      # Router interface
│   │   ├── context.go     # Context interface (with Set/Get for request-scoped data)
│   │   └── binder.go      # Controller registration with DI
│   └── testing/
│       └── app.go         # Test helpers: NewTestApp, NewTestContainer, NewTestAppWithOverrides
└── examples/
    ├── basic-api/          # Simple inline module example
    └── api/                # Full modular example with user module and middleware
```

## Key Components

### App
```go
app := ligo.New(
    ligo.WithRouter(echo.NewAdapter()),
    ligo.WithAddr(":8080"),
    ligo.WithMiddleware(RecoveryMiddleware, LoggingMiddleware),
)
app.Register(user.Module())
app.Run()
```

### Module
```go
func Module() ligo.Module {
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

### Provider Types
- `Value[T](instance)` - Pre-built singleton
- `Factory[T](fn)` - Factory function (auto-injected deps)
- `Transient[T](fn)` - New instance per resolve
- `Export(p)` - Make visible to sibling modules

### Middleware
```go
func AuthMiddleware(auth *AuthService) ligo.Middleware {
    return func(next ligo.HandlerFunc) ligo.HandlerFunc {
        return func(ctx ligo.Context) error {
            user, _ := auth.Validate(ctx.Header("Authorization"))
            ctx.Set("user", user)
            return next(ctx)
        }
    }
}

// In handler
func (c *Controller) Get(ctx ligo.Context) error {
    user := ctx.Get("user").(*User)  // Type-assert from context
    // ...
}
```

### DI Container Features
- Thread-safe singleton creation (per-type locks via `sync.Map`)
- Transient (new instance per resolve)
- Chain-based cycle detection (prevents deadlock)
- Auto-injection via reflection
- Error types: `ErrCircularDependency`, `ErrMissingDependency`, `ErrDuplicateProvider`

## Development Notes

- Root-level files are thin re-exports from `internal/` packages
- HTTP abstractions in `internal/http/` are adapter-agnostic
- Module middleware is resolved via DI and applied per module group
- Request-scoped data via `ctx.Set(key, val)` / `ctx.Get(key)`