# App & Lifecycle

The `App` is the main application instance that orchestrates modules, providers, and the HTTP server.

## Creating an App

```go
import "github.com/linkeunid/ligo"
import "github.com/linkeunid/ligo/adapters/echo"

router := echo.NewAdapter()
app := ligo.New(
    ligo.WithRouter(router),
    ligo.WithAddr(":8080"),
)
```

## App Options

| Option | Description |
|--------|-------------|
| `WithRouter(router)` | Set the HTTP router adapter |
| `WithAddr(addr)` | Set the server address (default: `:8080`) |
| `WithMiddleware(mw...)` | Add global middleware |
| `WithLogger(logger)` | Set the logger |
| `WithDebug(debug)` | Enable debug logging |
| `WithJSON()` | Enable JSON logging mode |
| `OnStart(fn)` | Add startup hook |
| `OnStop(fn)` | Add shutdown hook |
| `WithGracefulShutdown(timeout)` | Enable graceful shutdown on SIGINT/SIGTERM |

## Lifecycle Hooks

Ligo supports two types of lifecycle hooks:

### Module-Level Hooks (Functional)

Module-level hooks run during application startup and shutdown:

```go
app := ligo.New(
    ligo.WithRouter(router),
    ligo.OnStart(func(ctx any) error {
        log.Println("Connecting to database...")
        // Initialize resources
        return nil
    }),
    ligo.OnStop(func(ctx any) error {
        log.Println("Closing connections...")
        // Cleanup resources
        return nil
    }),
)
```

### Provider-Level Hooks (Interface-Based)

Providers and controllers can implement lifecycle interfaces for more granular control:

```go
type DatabaseService struct {
    db *sql.DB
}

func (s *DatabaseService) OnModuleInit() error {
    var err error
    s.db = sql.Open("postgres", "dsn")
    return err
}

func (s *DatabaseService) OnApplicationBootstrap() error {
    return s.db.Ping()
}

func (s *DatabaseService) OnApplicationShutdown() error {
    return s.db.Close()
}

func (s *DatabaseService) OnModuleDestroy() error {
    return nil
}
```

**Available hooks:**
- `OnModuleInit()` — Called when module initializes
- `OnApplicationBootstrap()` — Called after all modules initialize, before serving
- `BeforeApplicationShutdown()` — Called before shutdown begins (drain-stop)
- `OnApplicationShutdown()` — Called during shutdown
- `OnModuleDestroy()` — Called when module destroys

### Compile-Time Safe Hook Registration (HookedFactory/HookedController)

For compile-time safety (catching typos in hook method names), use the `HookedFactory` pattern for providers and `HookedController` for controllers with explicit hook registration:

#### Providers (HookedFactory)

```go
type Database struct {
    db *sql.DB
}

func (d *Database) Connect() error {
    var err error
    d.db = sql.Open("postgres", "dsn")
    return err
}

func (d *Database) Close() error {
    return d.db.Close()
}

// Register implements the Registerable interface for compile-time safe hook registration.
// Method expressions like d.Connect are type-checked by the compiler.
func (d *Database) Register(r *ligo.HookRegistry) {
    r.OnInit(d.Connect)     // If Connect doesn't exist → compile error
    r.OnShutdown(d.Close)   // Typo "Conenct" → compile error
}

// Provider registration with HookedFactory
ligo.Providers(
    ligo.HookedFactory[*Database](NewDatabase),
    // OR with Value:
    ligo.Value(database, ligo.WithHooks()),
)
```

#### Controllers (HookedController)

Controllers can also use compile-time safe hook registration:

```go
type UserController struct {
    userService *UserService
    log         ligo.Logger
}

func (c *UserController) Initialize() error {
    c.log.Info("User controller initializing")
    return nil
}

func (c *UserController) Ready() error {
    c.log.Info("User controller ready to handle requests")
    return nil
}

func (c *UserController) Draining() error {
    c.log.Info("User controller draining - completing in-flight requests")
    return nil
}

func (c *UserController) Shutdown() error {
    c.log.Info("User controller shutting down")
    return nil
}

// Register implements the Registerable interface for compile-time safe hook registration.
func (c *UserController) Register(registry *ligo.HookRegistry) {
    registry.OnInit(c.Initialize)      // Compile-time checked
    registry.OnBootstrap(c.Ready)      // Compile-time checked
    registry.BeforeShutdown(c.Draining) // Compile-time checked
    registry.OnShutdown(c.Shutdown)    // Compile-time checked
}

// Controller registration with HookedController
ligo.Controllers(ligo.HookedController(NewUserController))
```

**Benefits of HookedFactory/HookedController:**
- **Compile-time safety**: Method typos are caught by the compiler
- **Explicit registration**: Clear what hooks are registered via the `Register` method
- **Meaningful names**: Use descriptive method names (`Initialize` vs `OnModuleInit`)
- **Same flexibility**: Only implement the hooks you need
- **Works for providers and controllers**

**Execution order:**
1. Module-level `OnStart` hooks
2. Provider `OnModuleInit` hooks (in registration order)
3. Provider `OnApplicationBootstrap` hooks
4. Application runs (HTTP server or signal wait)
5. Provider `BeforeApplicationShutdown` hooks (reverse order)
6. Provider `OnApplicationShutdown` hooks (reverse order)
7. Provider `OnModuleDestroy` hooks (reverse order)
8. Module-level `OnStop` hooks

**Works for both HTTP and non-HTTP apps:** Bots, CLI runners, and background workers can use the same lifecycle hooks — just create the app without `WithRouter()`.

**Non-HTTP mode details:**
- Controllers are still instantiated and their lifecycle hooks are executed
- A `NullRouter` (no-op router) is used internally to satisfy the `Controller` interface
- Controller `Routes()` methods are called but do nothing
- `OnApplicationBootstrap` can start background goroutines
- `Run()` blocks waiting for SIGINT/SIGTERM signals
- Perfect for: bots, CLI runners, scheduled tasks, message queue consumers

See [Controllers](controllers.md#non-http-controllers-background-workers) for detailed examples.

## Graceful Shutdown

Enable graceful shutdown to handle SIGINT and SIGTERM:

```go
app := ligo.New(
    ligo.WithRouter(router),
    ligo.WithGracefulShutdown(10 * time.Second),
)

// OnStop hooks will be called before shutdown
```

## Registering Modules

```go
app.Register(
    user.Module(),
    auth.Module(),
    posts.Module(),
)
```

## Running the App

```go
if err := app.Run(); err != nil {
    if err != http.ErrServerClosed {
        log.Fatal(err)
    }
}
```

## Accessing the Container

For advanced use cases, you can access the DI container after `Run()`:

```go
err := app.Run()

// After Run(), access the container
container := app.Container()
userRepo := container.Resolve[*UserRepo]()
```

> **Note**: Calling `Container()` before `Run()` will panic.
