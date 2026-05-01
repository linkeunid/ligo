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

Lifecycle hooks run during application startup and shutdown:

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
