# Providers

Providers define how dependencies are created and injected into your controllers and middleware.

## Provider Types

### Value - Pre-built Singleton

Register an already-created instance:

```go
config := &Config{Port: 8080, Debug: true}
ligo.Value(config)
```

### Factory - Singleton with Auto-injection

Register a factory function. Dependencies are automatically injected:

```go
// The function signature defines the dependencies
func NewUserService(repo *UserRepo, logger Logger) *UserService {
    return &UserService{repo: repo, logger: logger}
}

// Register it - dependencies are auto-resolved
ligo.Factory[*UserService](NewUserService)
```

### Transient - New Instance per Resolve

Create a new instance each time the dependency is resolved:

```go
func NewRequestContext() *RequestContext {
    return &RequestContext{startTime: time.Now()}
}

// Each call to Resolve[*RequestContext]() returns a new instance
ligo.Transient[*RequestContext](NewRequestContext)
```

### Export - Make Visible to Sibling Modules

Make a provider available to other modules:

```go
ligo.Export(ligo.Factory[*AuthService](NewAuthService))
```

## Example: Complete Module with Providers

```go
func UserModule() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(
            // Pre-built value
            ligo.Value(&Config{Debug: false}),

            // Factory with auto-injection
            ligo.Factory[*UserRepo](NewUserRepo),
            ligo.Factory[*UserService](NewUserService),

            // Transient - new instance each resolve
            ligo.Transient[*LoggerContext](NewLoggerContext),

            // Exported - available to other modules
            ligo.Export(ligo.Factory[*UserService](NewUserService)),
        ),
    )
}
```

## Provider Resolution

The DI container resolves dependencies based on type:

```go
func NewUserService(repo *UserRepo, logger Logger, config *Config) *UserService {
    // repo is automatically resolved as *UserRepo
    // logger is automatically resolved as Logger
    // config is automatically resolved as *Config
    return &UserService{
        repo:   repo,
        logger: logger,
        config: config,
    }
}
```

## Provider Scopes

| Type | Scope | When to Use |
|------|-------|-------------|
| `Value[T]` | Singleton | Configuration, constants, pre-built objects |
| `Factory[T]` | Singleton | Services, repositories (one instance per app) |
| `Transient[T]` | Transient | Request-scoped objects, stateful instances |
| `Export(p)` | Shared | Services needed by sibling modules |
