# HTTP Router Adapter Proposals

Ligo is designed to work with any HTTP router through its adapter pattern. Below are proposals for additional router adapters.

## Table of Contents

- [Available Adapters](#available-adapters)
  - [Echo v5](#echo-v5-)
- [Proposed Adapters](#proposed-adapters)
  - [Fiber](#fiber)
  - [Gin](#gin)
  - [Chi](#chi)
  - [Stdlib](#stdlib)
- [Implementation Guide](#implementation-guide)
- [Performance Comparison](#performance-comparison)
- [Contributing](#contributing)

---

## Available Adapters

### Echo v5 ✅
**Package:** `github.com/linkeunid/ligo/adapters/echo`
**Status:** Complete

```go
import "github.com/linkeunid/ligo/adapters/echo"

app := ligo.New(
    ligo.WithRouter(echo.NewAdapter()),
    ligo.WithAddr(":8080"),
)
```

---

## Proposed Adapters

### Fiber
**Proposal:** `github.com/linkeunid/ligo/adapters/fiber`

```go
import "github.com/linkeunid/ligo/adapters/fiber"

app := ligo.New(
    ligo.WithRouter(fiber.NewAdapter()),
    ligo.WithAddr(":3000"),
)
```

**Implementation Notes:**
- Fiber uses fasthttp instead of net/http
- Need to adapt Context interface
- Performance optimization opportunities

**Reference:** [gofiber/fiber](https://github.com/gofiber/fiber)

---

### Gin
**Proposal:** `github.com/linkeunid/ligo/adapters/gin`

```go
import "github.com/linkeunid/ligo/adapters/gin"

app := ligo.New(
    ligo.WithRouter(gin.NewAdapter()),
    ligo.WithAddr(":8080"),
)
```

**Implementation Notes:**
- Gin is very popular in Go community
- Similar to Echo in design
- Should be straightforward to implement

**Reference:** [gin-gonic/gin](https://github.com/gin-gonic/gin)

---

### Chi
**Proposal:** `github.com/linkeunid/ligo/adapters/chi`

```go
import "github.com/linkeunid/ligo/adapters/chi"

app := ligo.New(
    ligo.WithRouter(chi.NewAdapter()),
    ligo.WithAddr(":8080"),
)
```

**Implementation Notes:**
- Chi is idiomatic net/http router
- Minimal API surface
- Good for standard library enthusiasts

**Reference:** [go-chi/chi](https://github.com/go-chi/chi)

---

### Stdlib
**Proposal:** `github.com/linkeunid/ligo/adapters/stdlib`

```go
import "github.com/linkeunid/ligo/adapters/stdlib"

app := ligo.New(
    ligo.WithRouter(stdlib.NewAdapter()),
    ligo.WithAddr(":8080"),
)
```

**Implementation Notes:**
- Pure net/http ServeMux
- No external dependencies
- For those who want zero dependencies

---

## Adapter Interface

All adapters must implement:

```go
type Router interface {
    // Group creates a new route group with a path prefix
    Group(prefix string) Router

    // Use adds global middleware
    Use(middleware ...Middleware)

    // HTTP methods
    Get(path string, handlers ...HandlerFunc) Route
    Post(path string, handlers ...HandlerFunc) Route
    Put(path string, handlers ...HandlerFunc) Route
    Delete(path string, handlers ...HandlerFunc) Route
    Patch(path string, handlers ...HandlerFunc) Route
    Options(path string, handlers ...HandlerFunc) Route

    // Serve starts the server
    Serve(addr string) error
}

type SetLoggerRouter interface {
    SetLogger(logger Logger)
}

type SetContainerRouter interface {
    SetContainer(container *container.Container)
}

type GracefulServer interface {
    Shutdown(ctx context.Context) error
}
```

---

## Implementing a New Adapter

1. Create package under `adapters/<name>/`
2. Implement `Router` interface
3. Optionally implement `SetLoggerRouter`, `SetContainerRouter`, `GracefulServer`
4. Export `NewAdapter()` function
5. Add tests
6. Add documentation

**Template:**

```go
package myrouter

import (
    "github.com/linkeunid/ligo/internal/http"
)

type Adapter struct {
    router *myrouter.Router
    // ... other fields
}

func NewAdapter() *Adapter {
    return &Adapter{
        router: myrouter.New(),
    }
}

// Implement Router interface...
```

---

## Performance Comparison

Target benchmarks (requests per second):

| Router | Single GET | JSON Response | Complex Route |
|--------|-----------|---------------|---------------|
| Echo   | ~100k     | ~80k          | ~60k          |
| Fiber  | ~150k     | ~120k         | ~90k          |
| Gin    | ~95k      | ~75k          | ~55k          |
| Chi    | ~85k      | ~65k          | ~50k          |
| Stdlib | ~70k      | ~55k          | ~40k          |

*Approximate values; actual results depend on hardware and workload.*

---

## Contributing

To add a new adapter:

1. Open an issue proposing the adapter
2. Discuss implementation approach
3. Create PR following adapter template
4. Include benchmarks
5. Add example in documentation
