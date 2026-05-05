# Controllers

Controllers handle HTTP requests. They receive dependencies via constructor injection.

## Creating a Controller

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
```

## Registering Controllers

```go
func UserModule() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(
            ligo.Factory[*UserService](NewUserService),
        ),
        ligo.Controllers(func(svc *UserService) ligo.Controller {
            return NewUserController(svc)
        }),
    )
}
```

## Context Methods

The `Context` interface provides access to request and response:

| Method | Description |
|--------|-------------|
| `Request() *http.Request` | Get the request |
| `Response() http.ResponseWriter` | Get the response writer |
| `Param(key string) string` | Get URL parameter (e.g., `:id`) |
| `Bind(v any) error` | Bind request body to struct |
| `JSON(code int, v any) error` | Send JSON response |
| `String(code int, s string) error` | Send string response |
| `Set(key string, val any)` | Set request-scoped value |
| `Get(key string) any` | Get request-scoped value |

## Response Helpers

Prefer these over raw `ctx.JSON` — they encode the status code in the name and keep handlers readable.

### 2xx Success

| Method | Status |
|--------|--------|
| `ctx.OK(v)` | 200 |
| `ctx.Created(v)` | 201 |
| `ctx.Accepted(v)` | 202 |
| `ctx.NoContent()` | 204 |

### 4xx Client Errors

The `msg` argument is optional — omit it to use the standard HTTP status text.

| Method | Status |
|--------|--------|
| `ctx.BadRequest(msg?)` | 400 |
| `ctx.Unauthorized(msg?)` | 401 |
| `ctx.Forbidden(msg?)` | 403 |
| `ctx.NotFound(msg?)` | 404 |
| `ctx.MethodNotAllowed(msg?)` | 405 |
| `ctx.NotAcceptable(msg?)` | 406 |
| `ctx.RequestTimeout(msg?)` | 408 |
| `ctx.Conflict(msg?)` | 409 |
| `ctx.Gone(msg?)` | 410 |
| `ctx.PreconditionFailed(msg?)` | 412 |
| `ctx.PayloadTooLarge(msg?)` | 413 |
| `ctx.UnsupportedMediaType(msg?)` | 415 |
| `ctx.UnprocessableEntity(msg?)` | 422 |
| `ctx.TooManyRequests(msg?)` | 429 |
| `ctx.ImATeapot(msg?)` | 418 |

### 5xx Server Errors

| Method | Status |
|--------|--------|
| `ctx.InternalServerError(msg?)` | 500 |
| `ctx.NotImplemented(msg?)` | 501 |
| `ctx.BadGateway(msg?)` | 502 |
| `ctx.ServiceUnavailable(msg?)` | 503 |
| `ctx.GatewayTimeout(msg?)` | 504 |
| `ctx.HTTPVersionNotSupported(msg?)` | 505 |

## Handler Examples

### JSON Response

```go
func (c *UserController) Get(ctx ligo.Context) error {
    id := ctx.Param("id")
    user, err := c.svc.Find(id)
    if err != nil {
        return ctx.NotFound("user not found")
    }
    return ctx.OK(user)
}
```

### Request Body Binding

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (c *UserController) Create(ctx ligo.Context) error {
    var req CreateUserRequest
    if err := ctx.Bind(&req); err != nil {
        return ctx.BadRequest(err.Error())
    }

    user, err := c.svc.Create(req)
    if err != nil {
        return ctx.InternalServerError(err.Error())
    }

    return ctx.Created(user)
}
```

### String Response

```go
func (c *HealthController) Check(ctx ligo.Context) error {
    return ctx.String(200, "OK")
}
```

### Using Request-Scoped Data

```go
func (c *UserController) Current(ctx ligo.Context) error {
    user := ctx.Get("user").(*User) // Set by middleware
    return ctx.OK(user)
}
```

## Route Groups

Create groups with shared prefixes:

```go
func (c *UserController) Routes(r ligo.Router) {
    api := r.Group("/api/v1")
    api.Handle("GET", "/users", c.List)
    api.Handle("GET", "/users/:id", c.Get)
}
```

## Lifecycle Hooks

Controllers and providers can implement lifecycle interfaces to run code at specific application stages.

### Interface-Based Hooks (Duck-Typing)

```go
type DatabaseService struct {
    db *sql.DB
}

func (s *DatabaseService) OnModuleInit() error {
    var err error
    s.db = sql.Open("postgres", "dsn")
    return err
}

func (s *DatabaseService) OnApplicationShutdown() error {
    return s.db.Close()
}

func (s *DatabaseService) BeforeApplicationShutdown() error {
    // Stop accepting new connections, finish in-flight requests
    return nil
}

type UserController struct {
    db *DatabaseService
}

func (c *UserController) Routes(r ligo.Router) {
    r.Handle("GET", "/", func(ctx ligo.Context) error {
        // c.db is already connected!
        return ctx.OK(...)
    })
}

app.Register(
    ligo.NewModule("users",
        ligo.Providers(
            ligo.Factory[*DatabaseService](NewDatabaseService),
        ),
        ligo.Controllers(func(db *DatabaseService) ligo.Controller {
            return &UserController{db: db}
        }),
    ),
)
```

### Compile-Time Safe Hooks (HookedController)

For compile-time safety, use the `HookedController` pattern:

```go
type UserController struct {
    userService *UserService
    log         ligo.Logger
}

func NewUserController(svc *UserService, log ligo.Logger) *UserController {
    return &UserController{userService: svc, log: log}
}

// Hook methods with meaningful names
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

func (c *UserController) Routes(r ligo.Router) {
    r.Handle("GET", "/users", c.List)
    r.Handle("GET", "/users/:id", c.Get)
}

// Register with HookedController
app.Register(ligo.NewModule("users",
    ligo.Providers(
        ligo.Factory[*UserService](NewUserService),
    ),
    ligo.Controllers(ligo.HookedController(NewUserController)),
))
```

**Benefits of HookedController:**
- **Compile-time safety**: Method typos are caught by the compiler
- **Explicit registration**: Clear what hooks are registered via the `Register` method
- **Meaningful names**: Use descriptive method names (`Initialize` vs `OnModuleInit`)

**Available hooks:**
- `OnModuleInit` — Called when module initializes
- `OnApplicationBootstrap` — Called after all modules initialize, before app serves
- `BeforeApplicationShutdown` — Called before shutdown begins (drain-stop)
- `OnApplicationShutdown` — Called during shutdown
- `OnModuleDestroy` — Called when module destroys

Hooks run in the same order for both HTTP and non-HTTP applications. The only difference is what happens between `OnApplicationBootstrap` and shutdown (HTTP server serves vs. waiting for signals).

### Non-HTTP Controllers (Background Workers)

Controllers can also be used in non-HTTP mode (bots, CLI runners, background workers). The `Routes()` method is **optional** — the framework automatically handles controllers without HTTP routes:

```go
type WorkerController struct {
    log    ligo.Logger
    cancel context.CancelFunc
}

func NewWorkerController(log ligo.Logger) *WorkerController {
    return &WorkerController{log: log}
}

// No Routes() method needed! The framework handles non-HTTP controllers automatically.

// Hook methods with meaningful names
func (c *WorkerController) Initialize() error {
    c.log.Info("Worker initializing")
    return nil
}

func (c *WorkerController) StartBackground() error {
    c.log.Info("Worker starting background goroutine")

    ctx, cancel := context.WithCancel(context.Background())
    c.cancel = cancel

    go c.run(ctx)
    return nil
}

func (c *WorkerController) DrainWork() error {
    c.log.Info("Worker draining - stopping new work")
    // Signal to stop accepting new work
    return nil
}

func (c *WorkerController) Stop() error {
    c.log.Info("Worker stopping")
    if c.cancel != nil {
        c.cancel()
    }
    return nil
}

// Register implements the Registerable interface for compile-time safe hook registration.
func (c *WorkerController) Register(registry *ligo.HookRegistry) {
    registry.OnInit(c.Initialize)
    registry.OnBootstrap(c.StartBackground)
    registry.BeforeShutdown(c.DrainWork)
    registry.OnShutdown(c.Stop)
}

func (c *WorkerController) run(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            c.doWork()
        }
    }
}

func (c *WorkerController) doWork() {
    c.log.Info("Processing scheduled task")
    // Background work here
}

// Register with HookedController for non-HTTP mode
app := ligo.New()
app.Register(ligo.NewModule("worker",
    ligo.Controllers(ligo.HookedController(NewWorkerController)),
))
app.Run() // Blocks until SIGINT/SIGTERM, worker runs in background
```

**Key points for non-HTTP controllers:**
- `Routes()` method is **optional** — only needed if you have HTTP routes
- Use `HookedController` for compile-time safe hook registration
- `OnApplicationBootstrap` (or custom `StartBackground`) is perfect for starting background goroutines
- `OnApplicationShutdown` (or custom `Stop`) gracefully stops goroutines
- App blocks on `Run()` waiting for SIGINT/SIGTERM signals
- All lifecycle hooks execute the same as HTTP mode
- Perfect for: bots, message queue consumers, scheduled tasks, CLI runners
