# Ligo Documentation

Welcome to the Ligo documentation. Ligo is a modular Go framework with lightweight dependency injection, inspired by NestJS.

## Topics

- [App & Lifecycle](app.md) - Application configuration and lifecycle hooks
- [Modules](modules.md) - Creating and organizing modules
- [Providers](providers.md) - Dependency injection with providers
- [Controllers](controllers.md) - Handling HTTP requests
- [Middleware](middleware.md) - Global, module-level, and route-level middleware
- [Guards](guards.md) - Authorization with guards
- [Pipes](pipes.md) - Validation and transformation with pipes
- [Interceptors](interceptors.md) - Logging, caching, and transformation with interceptors
- [Exception Filters](exception-filters.md) - Error handling and HTTP response conversion
- [DI Container](di-container.md) - How the DI container works

## Quick Reference

### Basic App Setup

```go
import "github.com/linkeunid/ligo"
import "github.com/linkeunid/ligo/adapters/echo"

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
```

### Common Patterns

| Pattern | Where to Learn |
|---------|----------------|
| Create a service with dependencies | [Providers](providers.md) |
| Organize related code | [Modules](modules.md) |
| Handle HTTP requests | [Controllers](controllers.md) |
| Add authentication | [Middleware](middleware.md) |
| Understand DI resolution | [DI Container](di-container.md) |
