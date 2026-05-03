# Ligo Best Practices

This guide covers recommended patterns and practices for building applications with Ligo.

## Project Structure

### Recommended Directory Layout

```
my-app/
├── cmd/
│   └── app/
│       └── main.go           # Application entry point
├── internal/
│   ├── user/
│   │   ├── module.go         # User module definition
│   │   ├── controller.go     # User controller
│   │   ├── service.go        # User service
│   │   └── repository.go     # User repository
│   ├── auth/
│   │   ├── module.go
│   │   ├── service.go
│   │   └── guard.go          # Auth guards
│   └── common/
│       └── middleware.go     # Shared middleware
├── go.mod
└── go.sum
```

### Module Organization

**Create one module per domain:**

```go
// internal/user/module.go
package user

func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(
            ligo.Factory[*Repository](NewRepository),
            ligo.Factory[*Service](NewService),
        ),
        ligo.Controllers(
            NewController,
        ),
    )
}
```

**Use module imports for dependencies:**

```go
// internal/user/module.go
func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Imports(auth.Module()), // Import auth module
        ligo.Providers(...),
        ligo.Controllers(
            func(svc *Service, auth *auth.AuthService) ligo.Controller {
                return &Controller{service: svc, auth: auth}
            },
        ),
    )
}
```

## Dependency Injection

### Prefer Factory over Value

**Good:**
```go
ligo.Factory[*Database](NewDatabase)
```

**Use Value only for config:**
```go
ligo.Value(&Config{Port: 8080})
```

### Use Interfaces for Abstractions

**Define interfaces in your module:**

```go
// internal/user/repository.go
type Repository interface {
    FindByID(id string) (*User, error)
    Create(user *User) error
}

// internal/user/repository_impl.go
type PostgresRepository struct {
    db *sql.DB
}

func (r *PostgresRepository) FindByID(id string) (*User, error) {
    // implementation
}

// internal/user/module.go
func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(
            ligo.Factory[*Repository](func(db *sql.DB) Repository {
                return &PostgresRepository{db: db}
            }),
        ),
    )
}
```

### Use Transient for Request-Scoped Services

```go
// Create new instance per request
ligo.Transient[*RequestContext](func() *RequestContext {
    return NewRequestContext()
})
```

## Controllers

### Keep Controllers Thin

Controllers should only handle HTTP concerns. Business logic goes in services.

```go
// Good: Thin controller
func (c *Controller) GetByID(ctx ligo.Context) error {
    id := ctx.Param("id")
    user, err := c.service.FindByID(id)
    if err != nil {
        return err // Let exception filter handle it
    }
    return ctx.OK(user)
}

// Bad: Business logic in controller
func (c *Controller) GetByID(ctx ligo.Context) error {
    id := ctx.Param("id")
    // Don't do this - logic should be in service
    rows, err := c.db.Query("SELECT * FROM users WHERE id = ?", id)
    // ...
}
```

### Use Validation Pipes

```go
type CreateUserInput struct {
    Name  string `validate:"required,min=3,max=50"`
    Email string `validate:"required,email"`
}

func (c *Controller) Create(ctx ligo.Context) error {
    input := ligo.ValidatedBody[CreateUserInput](ctx)
    user, err := c.service.Create(*input)
    if err != nil {
        return err
    }
    return ctx.Created(user)
}

func (c *Controller) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r)
    cr.POST("", c.Create).
        Pipe(ligo.ValidationPipe(&CreateUserInput{})).
        Handle()
}
```

### Use Typed Pipes for Parameters

```go
func (c *Controller) GetByID(ctx ligo.Context) error {
    id := ligo.Get[int](ctx, "id") // set by ParseIntPipe, already an int
    user, err := c.service.FindByID(id)
    if err != nil {
        return err
    }
    return ctx.OK(user)
}

func (c *Controller) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r)
    cr.GET("/:id", c.GetByID).
        Pipe(ligo.ParseIntPipe("id")).
        Handle()
}
```

### Return Errors, Don't Handle Them

Let exception filters handle errors:

```go
// Good: Return errors
func (c *Controller) GetByID(ctx ligo.Context) error {
    user, err := c.service.FindByID(id)
    if err != nil {
        return err // Exception filter will convert to HTTP response
    }
    return ctx.OK(user)
}

// Bad: Handle errors in controller
func (c *Controller) GetByID(ctx ligo.Context) error {
    user, err := c.service.FindByID(id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return ctx.NotFound("user not found")
        }
        return ctx.InternalServerError("database error")
    }
    return ctx.OK(user)
}
```

## Guards

### Use Guards for Authorization

```go
// Create reusable guards
func AuthGuard(auth *AuthService) ligo.Guard {
    return func(ctx ligo.Context) (bool, error) {
        token := ctx.Request().Header.Get("Authorization")
        if token == "" {
            return false, nil // Deny without error
        }
        user, err := auth.ValidateToken(token)
        if err != nil {
            return false, err
        }
        ctx.Set("user", user)
        return true, nil
    }
}

// Use guards in routes
func (c *Controller) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r)

    // Public endpoint
    cr.GET("/health", c.Health).Handle()

    // Protected endpoint
    cr.GET("/profile", c.Profile).
        Guard(AuthGuard(c.auth)).
        Handle()

    // Admin only
    cr.DELETE("/users/:id", c.DeleteUser).
        Guard(AuthGuard(c.auth), ligo.RolesGuard("admin")).
        Handle()
}
```

### Compose Guards

```go
// Combine multiple guards
cr.DELETE("/:id", c.Delete).
    Guard(
        AuthGuard(c.auth),
        ligo.RolesGuard("admin"),
        ligo.ThrottleGuard(10, time.Minute), // Rate limit
    ).
    Handle()
```

## Pipes

### Use Pipes for Input Transformation

```go
// Trim whitespace from input
cr.POST("", c.Create).
    Pipe(ligo.TrimPipe("name"), ligo.TrimPipe("email")).
    Pipe(ligo.ValidationPipe(&CreateUserInput{})).
    Handle()
```

### Chain Pipes for Complex Validation

```go
cr.PUT("/:id", c.Update).
    Pipe(ligo.ParseIntPipe("id")).
    Pipe(ligo.UUIDPipe("account_id")).
    Pipe(ligo.ValidationPipe(&UpdateUserInput{})).
    Handle()
```

### Do Not Use ValidationPipe Twice on the Same Route

`ValidationPipe[T]` reads the HTTP body via `ctx.Bind`. The body stream can only be read once, so a second `ValidationPipe` on the same route receives an empty body. Both would also overwrite each other's context value.

Instead, use a single composite DTO:

```go
// Wrong: body consumed on first pipe, second gets nothing
cr.POST("", c.Create).
    Pipe(ligo.ValidationPipe(&dto.CreateUserInput{})).
    Pipe(ligo.ValidationPipe(&dto.ExtraInput{})).
    Handle()

// Right: one composite DTO covers all fields
type CreateUserInput struct {
    dto.BaseUserInput
    Role string `json:"role" validate:"required"`
}

cr.POST("", c.Create).
    Pipe(ligo.ValidationPipe(&dto.CreateUserInput{})).
    Handle()
```

For path params alongside body validation, use `ParseIntPipe` / `UUIDPipe` before `ValidationPipe` — they read from the path, not the body:

```go
cr.PUT("/:id", c.Update).
    Pipe(ligo.ParseIntPipe("id")).
    Pipe(ligo.ValidationPipe(&dto.UpdateUserInput{})).
    Handle()
```

## Interceptors

### Use Interceptors for Cross-Cutting Concerns

```go
// Logging interceptor
func LoggingInterceptor(logger ligo.Logger) ligo.Interceptor {
    return func(ctx ligo.Context, next ligo.HandlerFunc) error {
        start := time.Now()
        path := ctx.Request().URL.Path

        err := next(ctx)

        logger.Info("request",
            ligo.LoggerField{Key: "path", Value: path},
            ligo.LoggerField{Key: "duration", Value: time.Since(start)},
            ligo.LoggerField{Key: "error", Value: err},
        )
        return err
    }
}

// Use in routes
cr.GET("", c.GetAll).
    Intercept(LoggingInterceptor(c.logger)).
    Handle()
```

### Add Timeouts to Slow Operations

```go
cr.GET("/export", c.ExportData).
    Intercept(ligo.TimeoutInterceptor(5 * time.Minute)).
    Handle()
```

## Middleware

### Global Middleware

Apply middleware at the app level for cross-cutting concerns:

```go
func main() {
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
        ligo.WithMiddleware(
            RecoveryMiddleware,
            CORSMiddleware,
            LoggingMiddleware,
        ),
    )
    app.Register(user.Module())
    app.Run()
}
```

### Module-Level Middleware

Apply middleware within modules:

```go
func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Middlewares(
            func(logger *Logger) ligo.Middleware {
                return UserLoggingMiddleware(logger)
            },
        ),
        ligo.Controllers(NewController),
    )
}
```

## Error Handling

### Create Custom Error Types

```go
// internal/user/errors.go
package user

import "errors"

var (
    ErrUserNotFound = errors.New("user not found")
    ErrInvalidEmail = errors.New("invalid email")
    ErrUserExists   = errors.New("user already exists")
)
```

### Use Exception Filters

```go
// Global exception filter
func GlobalExceptionFilter(logger ligo.Logger) ligo.ExceptionFilter {
    return func(ctx ligo.Context, err error) error {
        if err == nil {
            return nil
        }

        // Log the error
        logger.Error("request error",
            ligo.LoggerField{Key: "error", Value: err},
            ligo.LoggerField{Key: "path", Value: ctx.Request().URL.Path},
        )

        // Convert to HTTP response
        switch {
        case errors.Is(err, user.ErrUserNotFound):
            return ctx.NotFound(err.Error())
        case errors.Is(err, user.ErrInvalidEmail):
            return ctx.BadRequest(err.Error())
        case errors.Is(err, user.ErrUserExists):
            return ctx.Conflict(err.Error())
        default:
            return ctx.InternalServerError("internal server error")
        }
    }
}

// Use in routes
cr.GET("/:id", c.GetByID).
    Filter(GlobalExceptionFilter(logger)).
    Handle()
```

## Lifecycle Hooks

### Use OnModuleInit for Setup

```go
func Module() ligo.Module {
    return ligo.NewModule("database",
        ligo.Providers(
            ligo.Factory[*Database](NewDatabase),
        ),
        ligo.OnModuleInit(func() error {
            // Run migrations or setup
            return Migrate()
        }),
        ligo.OnModuleDestroy(func() error {
            // Cleanup
            return Close()
        }),
    )
}
```

### Use OnStart/OnStop for App-Level Hooks

```go
func main() {
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.OnStart(func(ctx any) error {
            log.Println("Application starting")
            return nil
        }),
        ligo.OnStop(func(ctx any) error {
            log.Println("Application stopping")
            return nil
        }),
    )
    app.Run()
}
```

### Implement Provider-Level Hooks

For service-level initialization and cleanup, implement lifecycle interfaces on your providers:

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
    // Verify connection before serving requests
    return s.db.Ping()
}

func (s *DatabaseService) OnApplicationShutdown() error {
    // Close connection gracefully
    return s.db.Close()
}
```

**When to use which:**
- **Module-level hooks** (`ligo.OnModuleInit(fn)`) — One-time setup like migrations
- **Provider-level hooks** (interface methods) — Service-specific initialization like DB connections
- **App-level hooks** (`ligo.OnStart(fn)`) — Cross-cutting concerns like global logging setup

**Execution order:**
1. Module `OnModuleInit` functions
2. Provider `OnModuleInit` methods
3. Provider `OnApplicationBootstrap` methods
4. App runs (HTTP or signals)
5. Provider `OnApplicationShutdown` methods
6. Provider `OnModuleDestroy` methods
7. Module `OnModuleDestroy` functions

### Non-HTTP Controllers for Background Workers

For bots, CLI runners, and scheduled tasks, use controllers without HTTP routes:

```go
type WorkerController struct {
    log    ligo.Logger
    cancel context.CancelFunc
}

func NewWorkerController(log ligo.Logger) *WorkerController {
    return &WorkerController{log: log}
}

// No Routes() method needed! Framework handles non-HTTP controllers automatically.

func (c *WorkerController) OnApplicationBootstrap() error {
    ctx, cancel := context.WithCancel(context.Background())
    c.cancel = cancel
    go c.run(ctx)
    return nil
}

func (c *WorkerController) OnApplicationShutdown() error {
    if c.cancel != nil {
        c.cancel()
    }
    return nil
}

func (c *WorkerController) run(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            c.processTask()
        }
    }
}

// No router needed - app.Run() blocks on signals
app := ligo.New()
app.Register(ligo.NewModule("worker",
    ligo.Controllers(NewWorkerController),
))
app.Run()
```

**Best practices for background workers:**
- Use `OnApplicationBootstrap` to start goroutines (not `OnModuleInit`)
- Store `context.CancelFunc` to gracefully stop goroutines
- Use `OnApplicationShutdown` to cancel and wait for cleanup
- `Routes()` method is **optional** — only needed for HTTP routes
- Use `time.Ticker` for scheduled tasks, not `time.Sleep` in loops

## Configuration

### Use Structured Config

```go
type Config struct {
    Port        int
    Environment string
    Database    DatabaseConfig
}

type DatabaseConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    Database string
}

func LoadConfig() *Config {
    return &Config{
        Port:        getEnv("PORT", 8080),
        Environment: getEnv("ENV", "development"),
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     getEnv("DB_PORT", 5432),
            Username: getEnv("DB_USER", "user"),
            Password: getEnv("DB_PASS", "pass"),
            Database: getEnv("DB_NAME", "db"),
        },
    }
}

func main() {
    config := LoadConfig()

    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(fmt.Sprintf(":%d", config.Port)),
        ligo.WithDebug(config.Environment == "development"),
        ligo.Providers(
            ligo.Value(config),
        ),
    )

    app.Register(user.Module())
    app.Run()
}
```

## Testing

### Test Controllers with httptest

```go
func TestControllerGetByID(t *testing.T) {
    // Create test service
    mockService := &MockUserService{
        users: map[string]*User{
            "1": {ID: "1", Name: "Test User"},
        },
    }

    // Create controller
    controller := NewController(mockService)

    // Create test router
    e := echo.New()
    req := httptest.NewRequest("GET", "/users/1", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Create handler
    handler := func(ctx ligo.Context) error {
        return controller.GetByID(ctx)
    }

    // Wrap with Echo context
    // ... (implementation depends on adapter)

    // Assert
    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
}
```

## Performance

### Use Connection Pooling

```go
func NewDatabase(config *Config) (*Database, error) {
    db, err := sql.Open("postgres", config.DSN())
    if err != nil {
        return nil, err
    }

    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    return &Database{db: db}, nil
}
```

### Enable JSON Logging in Production

```go
isProduction := os.Getenv("ENV") == "production"

app := ligo.New(
    ligo.WithRouter(echo.NewAdapter()),
    ligo.WithJSON(), // JSON logging
    ligo.WithDebug(!isProduction),
)
```

### Use Graceful Shutdown

```go
app := ligo.New(
    ligo.WithRouter(echo.NewAdapter()),
    ligo.WithGracefulShutdown(10 * time.Second),
)
```

## Security

### Always Validate Input

```go
type CreateUserInput struct {
    Name  string `validate:"required,min=3,max=50"`
    Email string `validate:"required,email"`
}

cr.POST("", c.Create).
    Pipe(ligo.ValidationPipe(&CreateUserInput{})).
    Handle()
```

### Use Rate Limiting

```go
cr.POST("/login", c.Login).
    Guard(ligo.ThrottleGuard(5, time.Minute)).
    Handle()
```

### Sanitize Error Messages

```go
func GlobalExceptionFilter(logger ligo.Logger) ligo.ExceptionFilter {
    return func(ctx ligo.Context, err error) error {
        // Log detailed error
        logger.Error("error", ligo.LoggerField{Key: "error", Value: err})

        // Return generic message to client
        if isProduction() {
            return ctx.InternalServerError("an error occurred")
        }
        return ctx.InternalServerError(err.Error())
    }
}
```

## Summary

- Organize code by domain with one module per feature
- Keep controllers thin, put business logic in services
- Use validation pipes for input validation
- Use guards for authorization
- Use exception filters for error handling
- Return errors from controllers, don't handle them
- Use lifecycle hooks for setup/teardown
- Test controllers with httptest
- Enable JSON logging in production
- Always validate and sanitize input
