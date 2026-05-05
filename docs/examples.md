# Examples Guide

This document describes the example applications demonstrating Ligo framework features.

## Table of Contents

- [Boilerplate Repository](#boilerplate-repository)
- [REST API (CRUD Operations)](#1-rest-api-crud-operations-)
- [Authentication/Authorization](#2-authenticationauthorization-)
- [File Upload](#3-file-upload-)
- [Lifecycle Hooks](#4-lifecycle-hooks-)
- [Module Structure in Examples](#module-structure-in-examples)
- [Common Patterns Demonstrated](#common-patterns-demonstrated)
- [Contributing Examples](#contributing-examples)
- [Running Individual Examples](#running-individual-examples)

## Boilerplate Repository

All examples are in the [ligo-boilerplate](https://github.com/linkeunid/ligo-boilerplate) repository.

### Running the Examples

```bash
git clone https://github.com/linkeunid/ligo-boilerplate.git
cd ligo-boilerplate
go run ./cmd/example
```

The example server starts on `http://localhost:8080`.

---

## Available Examples

### 1. REST API (CRUD Operations) ✅

**Location:** `internal/user/`

**Endpoints:**
- `GET /users` - List all users
- `GET /users/:id` - Get user by ID
- `POST /users` - Create new user
- `PUT /users/:id` - Update user
- `DELETE /users/:id` - Delete user

**Features demonstrated:**
- CRUD operations with in-memory repository
- HTTP response helpers (`ctx.OK`, `ctx.NotFound`, `ctx.Created`, etc.)
- Exception filter for error handling
- Logging interceptor

**Example:**
```bash
# List all users
curl http://localhost:8080/users

# Get user by ID
curl http://localhost:8080/users/1

# Create user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer valid-token" \
  -d '{"name":"John Doe","email":"john@example.com"}'

# Update user
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer valid-token" \
  -d '{"name":"Jane Doe","email":"jane@example.com"}'

# Delete user
curl -X DELETE http://localhost:8080/users/1 \
  -H "Authorization: Bearer valid-admin-token"
```

---

### 2. Authentication/Authorization ✅

**Location:** `internal/auth/`

**Features demonstrated:**
- Custom guards (`AuthGuard`, `AdminGuard`)
- Role-based access control (`RolesGuard`)
- JWT-style token validation
- Context-based user storage
- Protected routes with guard chains

**Example:**
```bash
# Public endpoint (no auth required)
curl http://localhost:8080/

# Protected endpoint (requires valid token)
curl http://localhost:8080/users/1 \
  -H "Authorization: Bearer valid-token"

# Admin-only endpoint (requires admin role)
curl -X DELETE http://localhost:8080/users/1 \
  -H "Authorization: Bearer valid-admin-token"
```

**Code:**
```go
// AuthGuard validates JWT tokens
func AuthGuard(auth *AuthService) ligo.Guard {
    return func(ctx ligo.Context) (bool, error) {
        token := ctx.Request().Header.Get("Authorization")
        if !strings.HasPrefix(token, "Bearer ") {
            return false, common.ErrUnauthorized
        }
        // Validate token and store user in context
        user, err := auth.Validate(token)
        if err != nil {
            return false, err
        }
        ctx.Set(auth.ContextKeyUser, user)
        return true, nil
    }
}

// AdminGuard checks for admin role
cr.DELETE("/:id", c.DeleteUser).
    Guard(auth.AuthGuard(c.auth), ligo.RolesGuard(auth.ContextKeyUser, "admin")).
    Handle()
```

---

### 3. File Upload ✅

**Location:** `internal/file/`

**Features demonstrated:**
- Multipart file upload handling
- File size validation (10MB max)
- File type detection (using `http.DetectContentType` and extension fallback)
- In-memory file storage for demo
- Streaming file downloads
- File metadata (ID, name, size, content-type)

**Example:**
```bash
# Upload file
curl -X POST http://localhost:8080/files/upload \
  -F "file=@/path/to/file.jpg" \
  -H "Authorization: Bearer valid-token"

# Upload with metadata
curl -X POST http://localhost:8080/files/upload \
  -F "file=@/path/to/file.jpg" \
  -F "metadata={\"title\":\"My File\"}" \
  -H "Authorization: Bearer valid-token"
```

---

### 4. Lifecycle Hooks ✅

**Features demonstrated:**
- Provider-level lifecycle interfaces
- Database connection initialization
- Cache warming on startup
- Graceful shutdown handling

**HTTP App Example:**
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
    return s.db.Ping() // Verify connection
}

func (s *DatabaseService) BeforeApplicationShutdown() error {
    // Stop accepting new connections, finish in-flight requests
    return s.db.Close()
}

func (s *DatabaseService) OnApplicationShutdown() error {
    // Final cleanup after all connections are drained
    return nil
}

type CacheService struct {
    cache *sync.Map
}

func (s *CacheService) OnApplicationBootstrap() error {
    // Warm cache on startup
    s.cache.Store("key", "value")
    return nil
}
```

**Non-HTTP App Example (Bot):**
```go
type BotService struct {
    client *discord.Client
}

func (s *BotService) OnModuleInit() error {
    s.client = discord.New("token")
    return nil
}

func (s *BotService) OnApplicationBootstrap() error {
    s.client.Open()
    return nil
}

func (s *BotService) BeforeApplicationShutdown() error {
    // Stop accepting new messages
    s.client.Disconnect()
    return nil
}

func (s *BotService) OnApplicationShutdown() error {
    s.client.Close()
    return nil
}

// No router needed - app waits for SIGINT/SIGTERM
app := ligo.New()
app.Register(ligo.NewModule("bot",
    ligo.Providers(ligo.Factory[*BotService](NewBotService)),
))
app.Run() // Blocks until Ctrl+C
```

**Non-HTTP App Example (Background Worker with Controller):**
```go
// Worker controller manages background task execution
type WorkerController struct {
    log    ligo.Logger
    cancel context.CancelFunc
    running bool
}

func NewWorkerController(log ligo.Logger) *WorkerController {
    return &WorkerController{log: log}
}

// No Routes() method needed for non-HTTP mode!
// The framework handles controllers without HTTP routes automatically.

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
    c.running = true
    return nil
}

func (c *WorkerController) DrainWork() error {
    c.log.Info("Worker draining - stopping new work")
    c.running = false
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

    // Execute immediately on start
    c.doWork()

    for {
        select {
        case <-ctx.Done():
            c.log.Info("Worker stopped")
            return
        case <-ticker.C:
            c.doWork()
        }
    }
}

func (c *WorkerController) doWork() {
    c.log.Info("Executing background task...")
    // Do work here
}

// Register the worker module with HookedController
app := ligo.New()
app.Register(ligo.NewModule("worker",
    ligo.Controllers(ligo.HookedController(NewWorkerController)),
))
app.Run() // Blocks until Ctrl+C, worker runs in background
```

**Compile-Time Safe Hook Registration (HookedFactory/HookedController):**

For compile-time safety (catching typos in hook method names at compile time), use the `HookedFactory` pattern for providers and `HookedController` for controllers:

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

// Controller registration with HookedController
ligo.Controllers(ligo.HookedController(NewWorkerController))
```

**Benefits of HookedFactory:**
- **Compile-time safety**: Method typos like `OnModulInit` instead of `OnModuleInit` are caught by the compiler
- **Explicit registration**: Clear what hooks are registered via the `Register` method
- **Same flexibility**: Only implement the hooks you need
- **Works for both Factory and Value providers**: Use `HookedFactory[T]()` or `Value(v, WithHooks())`

---

## Package Ecosystem

Advanced features like database integration and microservices are provided as separate packages, similar to NestJS's `@nestjs/typeorm` and `@nestjs/microservices` approach.

See [Package Ecosystem](../roadmaps/ecosystem.md) for:
- Database integration packages (planned)
- Microservices packages (planned)
- WebSocket, GraphQL, Scheduling packages (planned)

---

## Module Structure in Examples

The boilerplate demonstrates clean module architecture:

```
ligo-boilerplate/
├── cmd/
│   └── example/
│       └── main.go           # Application entry point
├── internal/
│   ├── auth/                 # Authentication module
│   │   ├── guard.go          # Auth guards
│   │   ├── module.go         # Module definition
│   │   └── user.go           # User entity
│   ├── user/                 # User CRUD module
│   │   ├── controller.go     # HTTP handlers
│   │   ├── service.go        # Business logic
│   │   ├── repository.go     # Data access (in-memory)
│   │   ├── filter.go         # Exception filters
│   │   ├── interceptor.go    # Logging interceptor
│   │   └── module.go         # Module definition
│   ├── health/               # Health check module
│   └── root/                 # Root/info endpoint
└── go.mod
```

---

## Common Patterns Demonstrated

### 1. Guard Composition
```go
cr.DELETE("/:id", c.DeleteUser).
    Guard(
        auth.AuthGuard(c.auth),           // Must be authenticated
        ligo.RolesGuard("user", "admin"),  // Must have one of these roles
    ).
    Handle()
```

### 2. Exception Handling
```go
// Global exception filter handles all errors
cr.Filter(common.GlobalExceptionFilter)
```

### 3. Logging
```go
cr.Intercept(ligo.LoggingInterceptor(func(start time.Time, ctx Context, err error) {
    duration := time.Since(start)
    log.Info("Request completed",
        ligo.LoggerField{Key: "duration", Value: duration},
    )
}))
```

### 4. Lifecycle Hooks
```go
type DatabaseService struct {
    db *sql.DB
}

func (s *DatabaseService) OnModuleInit() error {
    var err error
    s.db = sql.Open("postgres", "dsn")
    return err
}

func (s *DatabaseService) BeforeApplicationShutdown() error {
    // Stop accepting new connections
    return s.db.Close()
}

func (s *DatabaseService) OnApplicationShutdown() error {
    // Final cleanup
    return nil
}
```

### 5. Module Dependencies
```go
func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Imports(auth.Module()),  // Import auth module
        ligo.Providers(...),
        ligo.Controllers(...),
    )
}
```

---

## Contributing Examples

To add a new example:

1. Create a new directory under `internal/` or `examples/`
2. Follow the existing module structure
3. Include comprehensive comments
4. Add curl examples in this document
5. Update the main.go to register the module

---

## Running Individual Examples

To run specific examples, modify `cmd/example/main.go`:

```go
app.Register(
    // Comment out modules you don't need
    // auth.Module(),
    // user.Module(),
    // health.Module(),
    // root.Module(),
)
```
