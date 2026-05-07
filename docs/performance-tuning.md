# Ligo Performance Tuning Guide

This guide covers performance optimization techniques for Ligo applications.

## Table of Contents

- [Benchmarking](#benchmarking)
- [Database Performance](#database-performance)
- [HTTP Performance](#http-performance)
- [Memory Management](#memory-management)
- [Concurrency](#concurrency)
- [Logging](#logging)
- [DI Container Performance](#di-container-performance)
- [Profiling](#profiling)

## Benchmarking

### Running Benchmarks

Ligo includes benchmarks for measuring framework performance:

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkLigoAppCreation -benchmem .

# Run with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./...

# Run with memory profiling
go test -bench=. -memprofile=mem.prof ./...
```

### Interpreting Results

Benchmark output format:
```
BenchmarkName-20     1000000    1234 ns/op    512 B/op    8 allocs/op
```

- `1000000`: Number of iterations
- `1234 ns/op`: Nanoseconds per operation
- `512 B/op`: Bytes allocated per operation
- `8 allocs/op`: Number of allocations per operation

### Baseline Performance

Based on benchmark results (13th Gen Intel i9-13900HK):

| Operation | ns/op | Notes |
|-----------|-------|-------|
| App Creation | ~400 ns | One-time cost at startup |
| Module Creation | ~280 ns | Per module |
| Provider Creation | ~0.2-5 ns | Negligible |
| Module Registration | ~270 ns | Per module |
| Controller Registration | ~170 ns | Per controller |

## Database Performance

### Connection Pooling

Configure appropriate pool sizes based on your workload:

```go
func NewDatabase(config *Config) (*Database, error) {
    db, err := sql.Open("postgres", config.DSN())
    if err != nil {
        return nil, err
    }

    // Connection pool settings
    db.SetMaxOpenConns(25)           // Maximum open connections
    db.SetMaxIdleConns(5)            // Maximum idle connections
    db.SetConnMaxLifetime(5 * time.Minute)  // Connection lifetime
    db.SetConnMaxIdleTime(1 * time.Minute)  // Idle timeout

    return &Database{db: db}, nil
}
```

**Guidelines:**

- `MaxOpenConns`: Typically 2-4x CPU cores for database-bound workloads
- `MaxIdleConns`: 10-20% of MaxOpenConns
- `ConnMaxLifetime`: Below database server timeout (typically 5-10 minutes)
- `ConnMaxIdleTime`: 1-2 minutes to release unused connections

### Use Prepared Statements

```go
// Good: Prepared statement (cached)
func (r *Repository) FindByID(id string) (*User, error) {
    var user User
    err := r.db.QueryRowContext(ctx,
        "SELECT id, name, email FROM users WHERE id = $1", id).
        Scan(&user.ID, &user.Name, &user.Email)
    return &user, err
}

// Bad: Dynamic query (not cached)
func (r *Repository) FindByID(id string) (*User, error) {
    query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", id)
    // ...
}
```

### Batch Operations

```go
// Good: Batch insert
func (r *Repository) CreateBatch(users []*User) error {
    query := `INSERT INTO users (name, email) VALUES ($1, $2)`

    tx, err := r.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.Prepare(query)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, user := range users {
        if _, err := stmt.Exec(user.Name, user.Email); err != nil {
            return err
        }
    }

    return tx.Commit()
}

// Bad: Individual inserts in loop
func (r *Repository) CreateBatch(users []*User) error {
    for _, user := range users {
        if _, err := r.db.Exec(
            "INSERT INTO users (name, email) VALUES ($1, $2)",
            user.Name, user.Email); err != nil {
            return err
        }
    }
    return nil
}
```

### Use Transactions Wisely

```go
// Keep transactions short
func (s *Service) Transfer(from, to string, amount int) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Do work quickly
    if err := s.debit(tx, from, amount); err != nil {
        return err
    }
    if err := s.credit(tx, to, amount); err != nil {
        return err
    }

    return tx.Commit()
}
```

## HTTP Performance

### Use Response Compression

Enable gzip compression for JSON responses:

```go
import "github.com/labstack/echo/v5/middleware"

func main() {
    e := echo.New()
    e.Use(middleware.Gzip())

    router := ligo.NewAdapter(e)
    app := ligo.New(
        ligo.WithRouter(router),
        // ...
    )
}
```

### Optimize JSON Encoding

```go
// Use efficient JSON tags
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email,omitempty"`  // Omit empty
}

// Avoid omitempty for frequently empty fields if it adds overhead
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`  // Include always
    Bio   string `json:"bio"`    // Empty string is fine
}
```

### Streaming for Large Responses

```go
func (c *Controller) ExportData(ctx ligo.Context) error {
    // Good: Stream large response
    reader, writer := io.Pipe()

    go func() {
        defer writer.Close()
        // Write data as it becomes available
        csvWriter := csv.NewWriter(writer)
        defer csvWriter.Flush()
        // ... write CSV data
    }()

    return ctx.Stream(reader)
}
```

### Use HTTP/2

```go
import "golang.org/x/net/http2"
import "golang.org/x/net/http2/h2c"

func main() {
    e := echo.New()

    // Enable HTTP/2
    server := &http.Server{
        Addr:    ":8080",
        Handler: e,
    }
    http2.ConfigureServer(server, &http2.Server{})

    // Use h2c for HTTP/2 without TLS
    // server.Handler = h2c.NewHandler(e, &http2.Server{})

    app := ligo.New(
        ligo.WithRouter(ligo.NewAdapter(e)),
    )
}
```

## Memory Management

### Reuse Buffers

```go
// Good: Sync.Pool for buffer reuse
var bufferPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

func processLargeData(data []byte) ([]byte, error) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    // Use buffer
    buf.Write(data)

    return buf.Bytes(), nil
}

// Bad: Allocate new buffer each time
func processLargeData(data []byte) ([]byte, error) {
    buf := new(bytes.Buffer)
    buf.Write(data)
    return buf.Bytes(), nil
}
```

### Limit Request Body Size

```go
import "github.com/labstack/echo/v5/middleware"

func main() {
    e := echo.New()

    // Limit request body to 1MB
    e.Use(middleware.BodyLimit("1M"))

    router := ligo.NewAdapter(e)
    // ...
}
```

### Avoid Memory Leaks in Closures

```go
// Good: Only capture what's needed
func (s *Service) Process(id string) {
    go func(id string) {
        // Only capture id, not entire service
        user, _ := s.repo.FindByID(id)
        // ...
    }(id)
}

// Bad: Capture entire service
func (s *Service) Process(id string) {
    go func() {
        // Captures 's' which may keep other references alive
        user, _ := s.repo.FindByID(id)
        // ...
    }()
}
```

## Concurrency

### Use Worker Pools

```go
type WorkerPool struct {
    tasks chan Task
    wg    sync.WaitGroup
}

func NewWorkerPool(size int) *WorkerPool {
    p := &WorkerPool{
        tasks: make(chan Task, size*2),
    }

    for i := 0; i < size; i++ {
        p.wg.Add(1)
        go p.worker()
    }

    return p
}

func (p *WorkerPool) worker() {
    defer p.wg.Done()
    for task := range p.tasks {
        task.Run()
    }
}

func (p *WorkerPool) Submit(task Task) {
    p.tasks <- task
}

func (p *WorkerPool) Close() {
    close(p.tasks)
    p.wg.Wait()
}
```

### Limit Concurrent Requests

```go
import "golang.org/x/time/rate"

func RateLimitMiddleware(requestsPerSecond int) ligo.Middleware {
    limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), 10)

    return func(next ligo.HandlerFunc) ligo.HandlerFunc {
        return func(ctx ligo.Context) error {
            if !limiter.Allow() {
                return ctx.TooManyRequests("rate limit exceeded")
            }
            return next(ctx)
        }
    }
}
```

### Use Context for Cancellation

```go
func (s *Service) ProcessWithTimeout(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    resultCh := make(chan Result, 1)
    errCh := make(chan error, 1)

    go func() {
        result, err := s.longRunningOperation()
        if err != nil {
            errCh <- err
            return
        }
        resultCh <- result
    }()

    select {
    case <-ctx.Done():
        return ctx.Err()
    case err := <-errCh:
        return err
    case result := <-resultCh:
        return s.handleResult(result)
    }
}
```

## Logging

### Use Structured Logging

```go
// Good: Structured logging
logger.Info("user created",
    ligo.LoggerField{Key: "user_id", Value: user.ID},
    ligo.LoggerField{Key: "email", Value: user.Email},
)

// Bad: String concatenation
logger.Info(fmt.Sprintf("user created: %s (%s)", user.ID, user.Email))
}
```

### Adjust Log Level in Production

```go
isProduction := os.Getenv("ENV") == "production"

if isProduction {
    logger = ligo.NewLogger(
        ligo.WithLoggerJSON(),
        ligo.WithLoggerDebug(false),  // Disable debug logs
    )
} else {
    logger = ligo.NewLogger(
        ligo.WithLoggerText(),
        ligo.WithLoggerDebug(true),  // Enable debug logs
    )
}
```

### Sampling for High-Frequency Logs

```go
// Sample only 10% of debug logs
func shouldSampleDebug() bool {
    return rand.Float64() < 0.1
}

func logDebugIfNeeded(logger ligo.Logger, msg string, fields ...ligo.LoggerField) {
    if shouldSampleDebug() {
        logger.Debug(msg, fields...)
    }
}
```

## DI Container Performance

### Built-in Optimizations

Ligo includes several built-in performance optimizations:

**Interface Resolution Caching:**
- First resolution scans all providers to find interface implementations
- Subsequent resolutions use cached mapping (O(1) lookup)
- ~90% faster after first resolve
- Automatic - no configuration needed

**Parallel Hook Execution:**
- Lifecycle hooks (OnInit, OnBootstrap, OnShutdown, OnDestroy) execute in parallel
- ~50% faster startup/shutdown for applications with multiple providers
- Automatic - no configuration needed

**Per-Type Locking:**
- Each provider type has its own lock during resolution
- Reduces contention compared to global locking
- Automatic - no configuration needed

### Prefer Singleton over Transient

```go
// Good: Singleton (created once)
ligo.Factory[*Database](NewDatabase)

// Use only when you need new instances
ligo.Transient[*RequestContext](func() *RequestContext {
    return NewRequestContext()
})
```

### Avoid Circular Dependencies

Circular dependencies trigger detection logic which adds overhead:

```go
// Bad: Circular dependency
func AModule() ligo.Module {
    return ligo.NewModule("a",
        ligo.Imports(BModule()),  // A depends on B
        ligo.Providers(
            ligo.Factory[*A](func(b *B) *A {
                return &A{b: b}
            }),
        ),
    )
}

func BModule() ligo.Module {
    return ligo.NewModule("b",
        ligo.Imports(AModule()),  // B depends on A
        ligo.Providers(
            ligo.Factory[*B](func(a *A) *B {
                return &B{a: a}
            }),
        ),
    )
}

// Good: Refactor to shared module
func SharedModule() ligo.Module {
    return ligo.NewModule("shared",
        ligo.Providers(
            ligo.Factory[*A](NewA),
            ligo.Factory[*B](NewB),
        ),
    )
}
```

## Profiling

### CPU Profiling

```go
import (
    "os"
    "runtime/pprof"
)

func main() {
    // Start CPU profiling
    if os.Getenv("CPU_PROFILE") == "true" {
        f, _ := os.Create("cpu.prof")
        pprof.StartCPUProfile(f)
        defer pprof.StopCPUProfile()
    }

    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
    )
    app.Run()
}
```

### Memory Profiling

```go
func main() {
    // Enable memory profiling
    if os.Getenv("MEM_PROFILE") == "true" {
        f, _ := os.Create("mem.prof")
        defer f.Close()
        runtime.GC()  // Get clean snapshot
        pprof.WriteHeapProfile(f)
    }

    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
    )
    app.Run()
}
```

### HTTP Profiling Endpoint

```go
import _ "net/http/pprof"

func main() {
    // Start pprof server on separate port
    go func() {
        http.ListenAndServe(":6060", nil)
    }()

    // Your app
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
    )
    app.Run()
}
```

### Analyzing Profiles

```bash
# CPU profile
go tool pprof cpu.prof
# Commands: top, list <function>, web

# Memory profile
go tool pprof mem.prof
# Commands: top, list <function>, web

# HTTP profiling
# Visit http://localhost:6060/debug/pprof/
```

## Performance Checklist

### Startup Performance
- [ ] Minimize the number of providers
- [ ] Avoid heavy computation in `OnModuleInit`
- [ ] Use lazy initialization where possible
- [ ] Profile startup time

### Runtime Performance
- [ ] Enable connection pooling
- [ ] Use prepared statements
- [ ] Implement caching for expensive operations
- [ ] Use response compression
- [ ] Optimize JSON encoding

### Memory Performance
- [ ] Reuse buffers with `sync.Pool`
- [ ] Limit request body size
- [ ] Avoid memory leaks in goroutines
- [ ] Profile memory usage

### Concurrency
- [ ] Use worker pools for CPU-bound work
- [ ] Implement rate limiting
- [ ] Use context for cancellation
- [ ] Avoid excessive goroutine creation

### Monitoring
- [ ] Add metrics collection
- [ ] Set up alerting
- [ ] Regular performance profiling
- [ ] Load testing before releases
