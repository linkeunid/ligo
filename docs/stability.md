# Ligo Stability Policy

This document outlines the stability guarantees, versioning policy, and deprecation policy for the Ligo framework.

## Semantic Versioning

Ligo follows [Semantic Versioning 2.0.0](https://semver.org/).

- **Major version (X.0.0)**: Incompatible API changes
- **Minor version (0.X.0)**: New functionality in a backwards compatible manner
- **Patch version (0.0.X)**: Backwards compatible bug fixes

### Version Examples

- `0.5.0` - First feature-complete release
- `0.6.0` - Internal restructuring (di package, HTTP subdirectories)




## Public API Audit

The following public APIs are considered stable and will follow semantic versioning:

### Core Types

- `App` - Main application struct
- `Module` - Module system
- `Provider` - DI provider types (Value, Factory, Transient)
- `Controller` - HTTP controller interface
- `Router` - HTTP router interface
- `Context` - HTTP request/response context

### Configuration Options

- `New(...Option) *App`
- `WithRouter(r Router) Option`
- `WithAddr(addr string) Option`
- `WithDebug(debug bool) Option`
- `WithMiddleware(mw ...Middleware) Option`
- `WithLogger(l Logger) Option`
- `OnStart(hook LifecycleHook) Option`
- `OnStop(hook LifecycleHook) Option`
- `WithGracefulShutdown(timeout time.Duration) Option`
- `WithAutoPort() Option`

### Module Options

- `NewModule(name string, ...ModuleOption) Module`
- `Providers(providers ...any) ModuleOption`
- `Imports(modules ...Module) ModuleOption`
- `Controllers(constructors ...any) ModuleOption`
- `Middlewares(constructors ...any) ModuleOption`
- `OnModuleInit(fn func() error) ModuleOption`
- `OnModuleDestroy(fn func() error) ModuleOption`
- `Dynamic(factory func(...any) Module, opts ...any) ModuleOption`

### Provider Types

- `Value[T any](instance T) Provider`
- `Factory[T any](fn any) Provider`
- `Transient[T any](fn any) Provider`
- `Export(p Provider) Provider`

### Built-in Utilities

**Guards:**
- `RolesGuard(contextKey string, roles ...string) Guard`
- `AdminGuard(contextKey string) Guard`
- `ThrottleGuard(limit int, window time.Duration) Guard`

**Pipes:**
- `ValidationPipe[T any](v *T) Pipe`
- `ValidatedBody[T any](ctx Context) *T`
- `ParseIntPipe(param string) Pipe`
- `ParseBoolPipe(param string) Pipe`
- `UUIDPipe(param string) Pipe`
- `TrimPipe(param string) Pipe`

**Interceptors:**
- `TimeoutInterceptor(timeout time.Duration) Interceptor`
- `LoggingInterceptor(logger Logger) Interceptor`

### Logger

- `NewLogger(...LoggerOption) Logger`
- `WithLoggerText() LoggerOption`
- `WithLoggerJSON() LoggerOption`
- `WithLoggerProduction() LoggerOption`
- `WithLoggerDebug() LoggerOption`

### Error Types

- `ErrAppAlreadyStarted`
- `ErrMissingDependency`
- `ErrCircularDependency`
- `ErrDuplicateProvider`

## Backward Compatibility Guarantees

### What Won't Change

1. **Public API Signatures**: Function signatures for all public APIs will remain stable within a major version
2. **Behavioral Contracts**: The behavior of guards, pipes, interceptors, and exception filters will remain consistent
3. **DI Container Semantics**: Dependency injection resolution behavior will not change
4. **Module System**: The module import/export system will maintain compatibility

### What May Change

1. **Internal Implementation**: Internal code paths may be optimized without affecting public APIs
2. **Default Values**: Default configuration values may change in minor versions if they improve security or performance
3. **Error Messages**: Error messages may be improved for clarity

## Deprecation Policy

### Deprecation Process

1. **Announcement**: Deprecated features will be documented in the release notes
2. **Warning**: Using deprecated features will produce compiler warnings where possible
3. **Grace Period**: Deprecated features will be supported for at least one minor version
4. **Removal**: Deprecated features will be removed in a major version bump

### Deprecation Example

```go
// Deprecated: Use WithLoggerJSON() instead. Will be removed in a future version.
func WithProduction() Option {
    return WithLoggerJSON()
}
```

## Breaking Changes Policy

Breaking changes will only occur in major version updates (e.g., 0.5.x → 0.6.x).

### Examples of Breaking Changes

1. **Renamed public types or functions**
2. **Changed function signatures**
3. **Removed public APIs**
4. **Changed behavior of existing features**

### Migration Guide

When breaking changes are introduced:

1. A migration guide will be provided in the release notes
2. Code examples will show how to update existing code
3. Automated migration tools will be provided where feasible

## Release Process

### Pre-Release Versions

- **Alpha**: Early development, not feature-complete
- **Beta**: Feature-complete, may have bugs
- **RC**: Release Candidate, stable except for critical bugs

### Stable Release Criteria

A version is considered stable when:

1. All public APIs are documented
2. Test coverage is ≥ 80%
3. No known critical bugs
4. Performance benchmarks pass
5. Documentation is complete

## Security Updates

Critical security updates may be released as patch versions that include:

- Security fixes
- Minimal necessary changes
- Documentation of the security issue

## Long-Term Support (LTS)

For major versions:

- Security updates will be provided for 12 months after the next major version
- Bug fixes will be provided for 6 months after the next major version
- After the support period, users must upgrade to a supported version

## API Stability Timeline

| Version | Status          | Support Until |
|---------|-----------------|---------------|
| 0.5.x   | Pre-release     | N/A           |
| 0.6.x   | Stable (current)| Supported     |

## Feedback and Contributions

Users are encouraged to:

1. Report bugs through GitHub Issues
2. Suggest improvements through GitHub Discussions
3. Submit pull requests for bug fixes
4. Participate in RFC discussions for breaking changes

## Additional Resources

- [Release Roadmap](roadmaps/1.0-release.md)
- [Migration Guide](migration.md)
- [Best Practices](best-practices.md) (to be created)
