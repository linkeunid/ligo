# DI Container

The dependency injection container manages provider registration and dependency resolution.

## Features

- **Thread-safe singleton creation** - Per-type locks via `sync.Map`
- **Transient providers** - New instance per resolve
- **Cycle detection** - Chain-based detection prevents deadlock
- **Auto-injection** - Dependencies resolved via reflection
- **Interface type support** - Register and resolve interface types; fallback scan finds a concrete implementor automatically
- **Interface resolution caching** - First resolution scans providers, subsequent resolutions use cached mapping (~90% faster)
- **Parallel hook execution** - Lifecycle hooks execute concurrently for faster startup/shutdown (~50% faster)
- **Error handling** - `ErrCircularDependency`, `ErrMissingDependency`, `ErrDuplicateProvider`, `ErrAmbiguousDependency`, `ErrControllerBinding`
- **Tree-format error messages** - Transitive missing dependencies produce a full chain showing exactly which type is missing and why

## How It Works

### Registration

Providers are registered with their type:

```go
container := container.New()

container.Register(reflect.TypeOf(&UserService{}), container.ProviderEntry{
    Factory: func(args []reflect.Value) (any, error) {
        return &UserService{}, nil
    },
    ArgTypes: []reflect.Type{
        reflect.TypeOf(&UserRepo{}), // Dependencies
    },
})
```

### Resolution

When resolving a dependency:
1. Check if already instantiated (singleton)
2. Check for circular dependencies
3. Resolve dependencies recursively
4. Create and cache instance

## Interface Type Support

`Factory[T]` and `Value[T]` work with interface types. When a factory function declares an interface parameter, the container scans registered providers for a concrete type that implements it:

```go
type FileRepository interface {
    Save(file []byte) error
}

// Register concrete type
ligo.Factory[*memory.FileRepository](func(cfg *Config) *memory.FileRepository {
    return memory.NewFileRepository(cfg.UploadDir)
})

// Register usecase that depends on the interface
ligo.Factory[*FileUseCase](NewFileUseCase) // func NewFileUseCase(repo FileRepository) *FileUseCase

// Container resolves: *memory.FileRepository implements FileRepository ✓
```

If two registered types implement the same interface, `ErrAmbiguousDependency` is returned listing both implementors. After first resolution, the result is cached under the interface type key for O(1) subsequent lookups.

## Error Types

### ErrMissingDependency

Thrown when a dependency cannot be found. For transitive failures, the full chain is preserved via `Cause`/`Unwrap` and surfaced through `ErrControllerBinding`:

```go
// If *UserRepo is not registered but *UserService needs it:
ligo.Factory[*UserService](NewUserService)
// ErrMissingDependency{Type: "*UserRepo", RequiredBy: "*UserService", Cause: ...}
```

### ErrCircularDependency

Thrown when dependencies form a cycle:

```go
// A depends on B, B depends on A
ligo.Factory[*A](func(b *B) *A { return &A{b} })
ligo.Factory[*B](func(a *A) *B { return &B{a} })
// Error: circular dependency detected
```

### ErrDuplicateProvider

Thrown when registering the same type twice:

```go
ligo.Factory[*UserService](NewUserService1)
ligo.Factory[*UserService](NewUserService2)
// Error: duplicate provider for *UserService
```

### ErrAmbiguousDependency

Thrown when multiple registered types implement the same interface:

```go
ligo.Factory[*PgUserRepo](NewPgUserRepo)   // implements UserRepository
ligo.Factory[*MemUserRepo](NewMemUserRepo) // also implements UserRepository

// Resolving UserRepository → Error: ambiguous dependency, lists both types
```

### ErrControllerBinding

Thrown when a controller's dependency chain cannot be fully resolved. Produces a tree-format message showing the full chain:

```
ligo: cannot build UserController in module "user"
  *usecase.UserUseCase  ← required by UserController
    repository.UserRepository  ← required by *usecase.UserUseCase
      no provider registered
```

Use `errors.As` to inspect the structured fields (`Module`, `TypeName`, `Dependency`, `Cause`).

## Thread Safety

The container uses `sync.Map` for per-type locking, allowing concurrent resolution:

```go
// Safe to call from multiple goroutines
go func() {
    svc := container.Resolve[*UserService]()
}()

go func() {
    svc := container.Resolve[*UserService]()
}()
```

## Singleton vs Transient

### Singleton (Default)

```go
ligo.Factory[*UserService](NewUserService)

// Always returns the same instance
svc1 := Resolve[*UserService]()
svc2 := Resolve[*UserService]()
// svc1 == svc2
```

### Transient

```go
ligo.Transient[*UserService](NewUserService)

// Returns a new instance each time
svc1 := Resolve[*UserService]()
svc2 := Resolve[*UserService]()
// svc1 != svc2
```

## Advanced: Manual Container Access

For advanced use cases, access the container after `Run()`:

```go
app.Run()

container := app.Container()
userService := container.Resolve[*UserService]()
```

> **Warning**: This is an "escape hatch" for advanced scenarios. Prefer using DI through constructors.
