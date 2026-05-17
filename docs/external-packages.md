# Creating External Ligo Packages

This guide explains how to create external packages that extend Ligo, similar to NestJS packages like `@nestjs/common`, `@nestjs/microservices`, or `@nestjs/typeorm`.

## Package Types

```
linkeunid/ (GitHub org)
├── ligo/                    # Core framework (this repo)
├── ligo-database-pgx/        # External: PostgreSQL integration
├── ligo-microservices/       # External: Message brokers
├── ligo-auth0/              # External: Auth0 integration
└── ligo-cache-redis/       # External: Redis caching
```

---

## Core vs External Packages

| Aspect | Core Package (ligo) | External Package |
|--------|-------------------|-----------------|
| **Location** | `github.com/linkeunid/ligo` | `github.com/linkeunid/ligo-*` |
| **Imports** | Uses internal packages | Imports `github.com/linkeunid/ligo` |
| **Access** | Can use `internal/` | Can only use public API |
| **Examples** | Echo adapter, Guards, Pipes | Database, Cache, Auth |

**Key principle:** External packages use only Ligo's public API.

---

## Step-by-Step: Creating an External Package

### Example: `ligo-cache-redis`

A Redis caching package similar to `@nestjs/cache-manager`.

#### 1. Create the Repository

```bash
mkdir ligo-cache-redis
cd ligo-cache-redis
go mod init github.com/linkeunid/ligo-cache-redis
go get github.com/linkeunid/ligo@latest
```

#### 2. Create the Module Factory

```go
// module.go
package cacheredis

import (
    "context"
    "time"

    "github.com/linkeunid/ligo"
    "github.com/redis/go-redis/v9"
)

type Config struct {
    Addr     string
    Password string
    DB       int
}

func Module(config Config) ligo.Module {
    return ligo.NewModule("cache-redis",
        ligo.Providers(
            ligo.Factory[*redis.Client](func() *redis.Client {
                return redis.NewClient(&redis.Options{
                    Addr:     config.Addr,
                    Password: config.Password,
                    DB:       config.DB,
                })
            }),
            ligo.Factory[*CacheService](func(client *redis.Client) *CacheService {
                return NewCacheService(client)
            }),
        ),
    )
}
```

#### 3. Create the Service

```go
// cache.go
package cacheredis

import (
    "context"
    "encoding/json"
    "time"

    "github.com/redis/go-redis/v9"
)

type CacheService struct {
    client *redis.Client
}

func NewCacheService(client *redis.Client) *CacheService {
    return &CacheService{client: client}
}

func (s *CacheService) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return s.client.Set(ctx, key, data, ttl).Err()
}

func (s *CacheService) Get(ctx context.Context, key string, dest any) error {
    return s.client.Get(ctx, key).Scan(dest)
}
```

#### 4. Usage in User's App

```go
// User's app
package main

import (
    "github.com/linkeunid/ligo"
    cacheredis "github.com/linkeunid/ligo-cache-redis"
)

func main() {
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
    )

    app.Register(
        cacheredis.Module(cacheredis.Config{
            Addr: "localhost:6379",
        }),
        user.Module(),
    )

    app.Run()
}
```

---

## Package Patterns

### Pattern 1: Integration Package (Like `@nestjs/typeorm`)

Database/ORM integration

```go
// Package: github.com/linkeunid/ligo-database-pgx
package databasepgx

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/linkeunid/ligo"
)

type Config struct {
    DSN string
}

func Module(config Config) ligo.Module {
    return ligo.NewModule("database",
        ligo.Providers(
            ligo.Factory[*pgxpool.Pool](func() *pgxpool.Pool {
                pool, err := pgxpool.New(context.Background(), config.DSN)
                if err != nil {
                    panic(err)
                }
                return pool
            }),
        ),
        ligo.OnModuleInit(func() error {
            // Run migrations
            return nil
        }),
    )
}
```

### Pattern 2: Feature Package (Like `@nestjs/passport`)

Complete feature with multiple components

```go
// Package: github.com/linkeunid/ligo-passport
package passport

import (
    "github.com/linkeunid/ligo"
)

type Strategy string

const (
    LocalStrategy   Strategy = "local"
    JWTStrategy     Strategy = "jwt"
    OAuthStrategy   Strategy = "oauth"
)

type Config struct {
    DefaultStrategy Strategy
}

func Module(config Config) ligo.Module {
    return ligo.NewModule("passport",
        ligo.Providers(
            ligo.Value(config),
            ligo.Factory[*AuthService](NewAuthService),
        ),
        ligo.Guards(
            AuthGuard(),
        ),
        ligo.Middlewares(
            AuthMiddleware(),
        ),
    )
}
```

### Pattern 3: Microservices Transport (Like `@nestjs/microservices`)

Message broker transport

```go
// Package: github.com/linkeunid/ligo-microservices-redis
package microservicesredis

import (
    "github.com/linkeunid/ligo/microservices/server"
    "github.com/linkeunid/ligo/microservices/broker"
    "github.com/redis/go-redis/v9"
)

type Config struct {
    Addr string
}

func NewServer(config Config) *server.Server {
    client := redis.NewClient(&redis.Options{Addr: config.Addr})
    broker := redis.NewBroker(client)
    return server.NewServer(broker)
}

// Usage in user's app
import msredis "github.com/linkeunid/ligo-microservices-redis"

func Module(config msredis.Config) ligo.Module {
    return ligo.NewModule("microservice",
        ligo.Providers(
            ligo.Factory[*server.Server](func() *server.Server {
                return msredis.NewServer(config)
            }),
        ),
    )
}
```

---

## Dynamic Modules (Advanced)

### When to Use Dynamic Modules

Dynamic modules are useful when:
1. **Runtime Configuration** - Module needs configuration at runtime
2. **Multiple Instances** - Same module type with different configs
3. **Conditional Registration** - Module structure depends on config

### Dynamic Module Pattern

**Regular Module:**
```go
func Module() ligo.Module {
    return ligo.NewModule("static",
        ligo.Providers(
            ligo.Value("static-value"),
        ),
    )
}
```

**Dynamic Module:**
```go
func RegisterDynamicModule(config string) ligo.Module {
    return ligo.NewModule("dynamic-wrapper",
        ligo.Dynamic(
            func(opts ...any) ligo.Module {
                // Config is passed through opts
                return ligo.NewModule("dynamic-instance",
                    ligo.Providers(
                        ligo.Value(config),
                    ),
                )
            },
            config,
        ),
    )
}
```

---

## Naming Conventions

### Repository Naming

```
✅ Good:
- ligo-database-pgx     (PostgreSQL integration)
- ligo-cache-redis       (Redis caching)
- ligo-auth0             (Auth0 integration)
- ligo-microservices     (Message brokers)

❌ Avoid:
- ligo-database         (Too generic)
- ligo-cache            (Too generic)
- ligo-auth             (Which auth provider?)
```

### Import Aliases

```go
// Good: Clear aliases
import (
    cacheredis "github.com/linkeunid/ligo-cache-redis"
    dbpgx     "github.com/linkeunid/ligo-database-pgx"
    auth0      "github.com/linkeunid/ligo-auth0"
)
```

---

## Module Integration Patterns

### Pattern 1: Standalone Module

```go
// External package provides a ready-to-use module
package mypackage

func Module() ligo.Module {
    return ligo.NewModule("mypackage",
        ligo.Providers(...),
    )
}

// User's app
import "github.com/linkeunid/ligo-mypackage"

app.Register(mypackage.Module())
```

### Pattern 2: Configurable Module

```go
// External package accepts configuration
package mypackage

type Config struct {
    Host     string
    Port     int
    Username string
}

func Module(config Config) ligo.Module {
    return ligo.NewModule("mypackage",
        ligo.Providers(
            ligo.Factory[*Client](func() *Client {
                return NewClient(config.Host, config.Port, config.Username)
            }),
        ),
    )
}

// User's app
import "github.com/linkeunid/ligo-mypackage"

app.Register(
    mypackage.Module(mypackage.Config{
        Host:     "localhost",
        Port:     5432,
        Username: "user",
    }),
)
```

---

## Common Integration Points

### 1. Providers

External packages register providers that users inject:

```go
// External package
func Module() ligo.Module {
    return ligo.NewModule("database",
        ligo.Providers(
            ligo.Factory[*Database](NewDatabase),
        ),
    )
}

// User's controller
func Controller(db *Database) ligo.Controller {
    return &Controller{db: db}
}
```

### 2. Controllers

External packages can provide controllers:

```go
// External package
func Controllers() ligo.ModuleOption {
    return ligo.Controllers(
        func(cache *CacheService) ligo.Controller {
            return &CacheController{cache: cache}
        },
    )
}

// Combined with providers
func FullModule() ligo.Module {
    return ligo.NewModule("cache",
        ligo.Providers(
            ligo.Factory[*CacheService](NewCacheService),
        ),
        Controllers(),
    )
}
```

### 3. Guards

```go
// External package
func AuthGuard() ligo.Guard {
    return func(ctx ligo.Context) (bool, error) {
        token := ctx.Request().Header.Get("Authorization")
        // Validate token
        return true, nil
    }
}
```

### 4. Middleware

```go
// External package
func LoggingMiddleware(logger Logger) ligo.Middleware {
    return func(next ligo.HandlerFunc) ligo.HandlerFunc {
        return func(ctx ligo.Context) error {
            start := time.Now()
            err := next(ctx)
            logger.Log(ctx.Request().URL.Path, time.Since(start))
            return err
        }
    }
}
```

### 5. Pipes

```go
// External package
func ValidationPipe[T any](v *T) ligo.Pipe {
    return ligo.Pipe{
        name: "ValidationPipe",
        fn: func(ctx ligo.Context) error {
            // Validate v
            return nil
        },
    }
}
```

---

## Dependency Management

### Version Constraints

```go
// go.mod of external package
module github.com/linkeunid/ligo-cache-redis

go 1.21

require (
    github.com/linkeunid/ligo v0.9.6  // Core framework
    github.com/redis/go-redis/v9 v9.0.0
)
```

### Ligo Version Compatibility

| Ligo Version | External Package Compatible? | Notes |
|---------------|----------------------------|-------|
| 0.5.x         | Yes (stable API) | Feature-complete release |
| 0.6.x         | Yes (no public API changes) | Internal restructuring only |

---

## Best Practices

### 1. Minimal Core

Keep core framework small:
- **Core**: Module system, DI, HTTP routing
- **External**: Databases, caching, auth, messaging

### 2. Public API Only

External packages can only use public Ligo API:
- ✅: `ligo.NewModule`, `ligo.Providers`, `ligo.Factory`
- ❌: `internal/di`, `internal/app/*` (any `internal/` package)

### 3. Module Factory

Always provide a module factory:
```go
func Module() ligo.Module {
    return ligo.NewModule("package-name", ...)
}

// Or with config
func Module(config Config) ligo.Module {
    return ligo.NewModule("package-name", ...)
}
```

### 4. Error Types

Define package-specific errors:
```go
var (
    ErrConnection = errors.New("connection failed")
    ErrTimeout   = errors.New("operation timed out")
)
```

### 5. Documentation

Include examples in your package README:
```markdown
# ligo-cache-redis

Redis caching for Ligo applications.

## Installation

```bash
go get github.com/linkeunid/ligo-cache-redis
```

## Usage

```go
import (
    "github.com/linkeunid/ligo"
    cacheredis "github.com/linkeunid/ligo-cache-redis"
)

app.Register(
    cacheredis.Module(cacheredis.Config{
        Addr: "localhost:6379",
    }),
)
```
```

---

## Quick Reference

### Decision Guide

```
What do you want to build?
├── Database integration?
│  └── Create: ligo-database-{driver}
│     └── Pattern: Module + Provider
│
├── Caching?
│  └── Create: ligo-cache-{backend}
│     └── Pattern: Module + Service
│
├── Message broker transport?
│  └── Create: ligo-microservices-{broker}
│     └── Pattern: Server + Broker interface
│
├── Authentication?
│  └── Create: ligo-auth-{provider}
│     └── Pattern: Module + Guards + Middleware
│
└── Utility functions?
   └── Create: ligo-{feature}
      └── Pattern: Exported functions/Pipes/Guards
```

### Package Template

```bash
# 1. Create package
mkdir ligo-myfeature
cd ligo-myfeature
go mod init github.com/linkeunid/ligo-myfeature

# 2. Add Ligo dependency
go get github.com/linkeunid/ligo@latest

# 3. Create module.go
# 4. Create README.md
# 5. Add tests
# 6: Tag release
```

---

## Summary

| Package Type | Purpose | Example |
|--------------|---------|---------|
| **Core** | Framework foundation | `ligo` |
| **Adapter** | HTTP framework | `adapters/echo` (in core) |
| **Integration** | Third-party service | `ligo-database-pgx` |
| **Feature** | Complete feature | `ligo-passport` |
| **Transport** | Message broker | `ligo-microservices-redis` |

**Key principle:** External packages should be self-contained, use only Ligo's public API, and provide a module factory for easy integration.
