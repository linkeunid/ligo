# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Note**: This is internal documentation for AI assistance. For user-facing documentation, see [README.md](README.md).

## Project Overview

**Ligo** is a modular Go framework with lightweight dependency injection, inspired by NestJS.

- **Module**: `github.com/linkeunid/ligo`
- **Go version**: 1.25.9
- **License**: MIT
- **Documentation**: [README.md](../README.md) (user-facing), [docs/features/](../docs/features/) (detailed)

## Commands

```bash
# Build
go build ./...

# Run tests (152 tests passing, 39.8% coverage)
go test ./...
go test -v ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. -benchmem ./...

# Run integration tests
go test -run TestAppLifecycle -v ./...
go test -run TestDIResolution -v ./...
go test -run TestMultipleModules -v ./...

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

## Documentation

**User-facing docs:**
- [README.md](README.md) - Project overview and quick start
- [docs/features/](docs/features/) - Detailed feature documentation
- [docs/examples.md](docs/examples.md) - Examples guide with API usage
- [docs/roadmaps/](docs/roadmaps/) - Release roadmap and future proposals
- [docs/migration.md](docs/migration.md) - Migration guide (0.x → 1.0)
- [docs/best-practices.md](docs/best-practices.md) - Development best practices
- [docs/performance-tuning.md](docs/performance-tuning.md) - Performance optimization guide
- [docs/deployment.md](docs/deployment.md) - Deployment guide (Docker, Kubernetes, Cloud)
- [docs/stability.md](docs/stability.md) - Stability policy and semver

## Architecture

```
ligo/
├── Public API (root level)
├── app.go                  # App struct and public methods (New, Register, Provide, Run)
├── router.go               # HTTP re-exports + built-in guards/pipes/interceptors
├── module.go               # Module re-exports (Module, NewModule, ModuleOptions)
├── provider.go             # Provider types (Value, Factory, Transient, Export)
├── options.go              # App options (WithRouter, WithAddr, WithMiddleware, etc.)
├── errors.go               # Error types
├── *_test.go               # Unit tests (app_test.go, module_test.go, etc.)
├── integration_test.go     # Integration tests for full app lifecycle
├── bench_test.go           # Performance benchmarks
├── internal/
│   ├── app/                # App implementation details
│   │   ├── app.go          # DI registration, module building
│   │   ├── app_test.go     # App tests
│   │   └── server.go       # Server startup, graceful shutdown, port retry
│   ├── core/               # Core DI, module system, logger, lifecycle, resolver
│   │   ├── container/      # DI container
│   │   ├── logger/         # Structured logging
│   │   ├── lifecycle/      # Lifecycle management
│   │   ├── module/         # Module system
│   │   └── resolver/       # Interface-based dependency resolution
│   ├── http/               # HTTP interfaces + chain/builder + built-ins
│   │   ├── guards.go       # Built-in guards (RolesGuard, ThrottleGuard, etc.)
│   │   ├── interceptors.go # Built-in interceptors (Timeout, Logging)
│   │   ├── pipes.go        # Built-in pipes (Validation, ParseInt, etc.)
│   │   ├── binder.go       # Controller registration with DI
│   │   ├── builder.go      # RouteBuilder for chain pattern
│   │   ├── chain.go        # ChainRouter for fluent API
│   │   ├── context.go      # Context interface (with Stream method)
│   │   └── router.go       # Router interface
│   ├── testing/            # Test helpers
│   └── adapters/           # Concrete implementations
│       └── echo/           # Echo v5 adapter
├── adapters/               # Public adapters
│   └── echo/               # Echo v5 adapter
├── docs/
│   ├── examples.md         # Examples guide
│   ├── migration.md        # Migration guide (0.x → 1.0)
│   ├── best-practices.md   # Development best practices
│   ├── performance-tuning.md  # Performance optimization
│   ├── deployment.md       # Deployment guide
│   ├── stability.md        # Stability policy
│   ├── features/           # Feature documentation
│   └── roadmaps/           # Release roadmap and future proposals
└── CLAUDE.md               # This file
```

### Structure Principles
- **Root files**: Minimal public API (11 files)
- **internal/app/**: App implementation details (DI, server logic)
- **internal/core/**: Framework core (DI container, module system, logger)
- **internal/http/**: HTTP abstractions (adapter-agnostic interfaces + built-ins)
- **internal/adapters/**: Concrete HTTP router implementations

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
- Request-scoped data via `ctx.Set(key, val)` / `ctx.Get(key)` - use constants for keys
- Logger uses NestJS-style context levels (ContextApp, ContextDIContainer, ContextRoutes, ContextLifecycle, ContextMiddleware)
- Middleware chaining is applied in reverse order (last middleware wraps first)
- Echo adapter's `wrapHandlerWithMiddleware` is shared between Adapter and groupAdapter
- Guards, Pipes, Interceptors, and Exception Filters use Go-idiomatic builder pattern (no decorators)
- Logger is automatically registered as a provider and injectable
- No hardcoded string keys or fmt.Printf in core code - use structured logging

## Testing

- **152 tests passing** with 39.8% coverage
- **Integration tests** (`integration_test.go`): Full app lifecycle, DI resolution, multiple modules, guards, pipes, interceptors
- **Benchmarks** (`bench_test.go`): App creation, module creation, provider types, route registration, guards, pipes, interceptors
- **Unit tests**: Comprehensive tests for internal packages (logger, module, lifecycle, resolver, container, app)

## Release Status

- **Current version**: 1.0 ✅ Ready for release
- All requirements completed:
  - ✅ API documentation (godoc comments)
  - ✅ Getting started guide
  - ✅ Migration guide (0.x → 1.0)
  - ✅ Best practices guide
  - ✅ Performance tuning guide
  - ✅ Deployment guide
  - ✅ Stability policy (semver, backward compatibility)
  - ✅ Integration tests (22 tests)
  - ✅ Performance benchmarks (16 benchmarks)

## Context Interface Methods

- `Request()` - Get HTTP request
- `Response()` - Get HTTP response writer
- `Param(key)` - Get path parameter
- `Bind(v)` - Bind request body to struct
- `JSON(code, v)` - Send JSON response
- `String(code, s)` - Send string response
- `Set/Get(key, val)` - Request-scoped data storage
- `OK(v), Created(v), NoContent()` - HTTP response helpers
- `BadRequest/Unauthorized/Forbidden/NotFound(msg)` - Error responses
- `Stream(reader)` - Stream file download

## Built-in Utilities

**Guards:** `RolesGuard`, `AdminGuard`, `ThrottleGuard`
**Pipes:** `ValidationPipe`, `ParseIntPipe`, `ParseBoolPipe`, `UUIDPipe`, `TrimPipe`
**Interceptors:** `TimeoutInterceptor`, `LoggingInterceptor`