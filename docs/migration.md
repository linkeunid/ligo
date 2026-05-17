# Migration Guide

This guide helps you migrate your Ligo applications between versions.

## [Migration Guide: 0.x → 0.5](#0x-→-05)

## [Migration Guide: 0.5 → 0.6](#05-→-06)

## [Migration Guide: 0.8 → 0.9](#08-→-09)

## [Migration Guide: 0.9.0 → 0.9.6](#090-→-096)

## [Migration Guide: 0.9.x → 0.10.0](#09x-→-0100)

---

## 0.9.x → 0.10.0

Two behavior changes to plan for, plus two additive APIs. All adjustments
are mechanical.

### Echo adapter: `SetContainer` is idempotent

Calling `Adapter.SetContainer(c)` more than once used to prepend the
request-scope middleware on every call. Each repeated call created an
extra layer of child container per request. The middleware now installs at
most once per adapter; subsequent calls just rebind the container pointer.

If you relied on the old behavior (unlikely — it was a bug), wrap your own
container instead.

### Echo adapter: `groupAdapter.Serve` returns `ErrServeOnGroup`

Calling `Serve(addr)` on a route group returned `nil` silently — it
looked like a successful server start but no server actually started. It
now returns `echo.ErrServeOnGroup`. Only the root `Adapter` can `Serve`.

```go
g := router.Group("/api")
if err := g.Serve(":8080"); errors.Is(err, echo.ErrServeOnGroup) {
    // programmer error — call Serve on the root router instead
}
```

### Echo adapter: per-request context pool removed

The `sync.Pool` of `contextAdapter` allocated through every request because
nothing ever called `Put`. The pool was pure overhead. Removed; contexts
are allocated fresh per request. Reintroduce a pool only with paired `Put`
and measured benchmark wins.

### `MockContext`: `ImATeapot`, `SetBody`, `WithBindError`

`MockContext` was missing `ImATeapot` — using it as a `ligo.Context`
would fail to compile. Added the method and a compile-time assertion
(`var _ http.Context = (*MockContext)(nil)`) so future interface drift
fails the build.

`Bind` and `BindQuery` were silent no-ops, useless for any handler that
read request data. New helpers:

```go
m := testing.NewMockContext()
m.SetBody(CreateUserInput{Name: "alice"})
// ...
m.WithBindError(errors.New("malformed JSON"))
```

Defaults remain no-ops for backwards compatibility with existing tests.

### Provider hooks now run sequentially by default

`OnInit` and `OnBootstrap` previously fired in parallel goroutines, which
made registration order non-deterministic. In 0.10.0 the default is
sequential execution in the order providers were registered. Apps that
relied on parallel execution opt in explicitly:

```go
// Before: parallel was implicit.
app := ligo.New()

// After: parallel is opt-in.
app := ligo.New(ligo.WithParallelHooks())
```

Sequential is almost always what you want — it makes log output
readable, makes failures easier to attribute to a specific provider, and
guarantees ordering for providers that have informal startup
dependencies (e.g., logger must be ready before the database hook
prints). Opt back into parallel only when you have many independent
providers whose startup work is I/O-bound and unordered.

### `Resolve[T]` returns `(T, error)` instead of panicking

The container's `Resolve[T]` now returns the resolution error rather
than panicking on missing dependencies, ambiguous interfaces, or
circular dependencies. A new `MustResolve[T]` preserves the old
panic-on-failure behavior for cases where a failure really is fatal.

```go
// Before
svc := di.Resolve[*UserService](container) // panic on failure

// After — handle the error
svc, err := di.Resolve[*UserService](container)
if err != nil {
    return fmt.Errorf("resolve user service: %w", err)
}

// After — keep the panic
svc := di.MustResolve[*UserService](container)
```

The same split exists at the application level: `ligo.Resolve[T](app)`
returns `(T, error)`, `ligo.MustResolve[T](app)` panics.

`internal/di` is not a public package, so direct callers of `di.Resolve`
are limited to the framework itself and ligo-* satellite repos. Use the
new `ligo.Resolve` / `ligo.MustResolve` re-exports from application
code.

### New: `ligo.Resolve` / `ligo.MustResolve` (app-level)

```go
import "github.com/linkeunid/ligo"

user, err := ligo.Resolve[*UserService](app)
if err != nil { /* handle */ }

// or, when failure should crash:
user := ligo.MustResolve[*UserService](app)
```

Both must be called after `app.Run()` has built the container.

### New: `ligo.WithParallelHooks()` Option

Restores the pre-0.10.0 parallel execution of provider `OnInit` and
`OnBootstrap` hooks. See "sequential by default" above for guidance on
when to enable it.

### Lifecycle: append-after-Start now panics

`AppLifecycle.AddServer`, `AppendStartHook`, and `AppendStopHook` panic
with `"ligo: lifecycle already started"` if called after `Start()` (same
panic message as the existing double-start guard). All three methods now
hold `mu` for the duration of the append, so concurrent registration
during the wiring phase is race-free. Previously, late appends silently
succeeded but the hooks never fired and the writes raced with `Start`.

### Lifecycle: `Stop()` is idempotent

A second `Stop()` returns `nil` immediately instead of double-shutting
down the HTTP server and re-running stop hooks. Concurrent `Stop()` calls
now run hooks exactly once.

### Lifecycle: `Start()` rolls back on hook failure

If start hook `i` returns an error, stop hooks at indices `i-1..0` run in
reverse before `Start` returns. The returned error is
`errors.Join(originalErr, rollbackErrs...)` so callers can inspect both
the original failure and any rollback failures with `errors.Is` /
`errors.As`. No opt-out flag — rollback is always on.

### Shutdown errors are returned, not swallowed

`(*App).shutdown` previously logged each `BeforeApplicationShutdown` /
`OnApplicationShutdown` / `OnModuleDestroy` failure but returned `nil`,
so operators got exit code 0 even on partial-shutdown failures. It now
returns `errors.Join` of every wrapped failure (each prefixed with the
hook kind, e.g. `"OnApplicationShutdown: ..."`). The HTTP serve path
propagates the joined error to callers.

### Parallel hook errors carry full messages

`executeHooksParallel` used to return the string
`"hook execution failed: %d errors occurred"`, discarding the individual
error messages. It now returns `errors.Join` of every hook failure so
calling code can `errors.As` to inspect specific causes.

### Context interface adds `RequestContext()`

Every `Context` implementation must now expose
`RequestContext() context.Context`. The echo adapter and `MockContext`
both implement it; custom Context implementations need to add it.
Handlers and interceptors should call `ctx.RequestContext()` (not
`ctx.Request().Context()`) when issuing cancellable downstream calls so
they observe timeouts and graceful-shutdown cancellation.

### `TimeoutInterceptor` is correctness-clean

`TimeoutInterceptor`:

- Derives the timeout from the per-request context, so client
  disconnect, parent timeouts, and graceful shutdown propagate. Previously
  it derived from `context.Background()` and ran independent of any other
  cancellation signal.
- Hands the handler a `Context` whose `RequestContext()` returns the
  timeout-bound context, so cancellable downstream calls exit promptly
  when the timeout fires.
- Documents that the handler goroutine is best-effort and not forcibly
  stopped — handlers that ignore cancellation keep running. The prior
  implementation also had this leak but wrote it onto the same
  ResponseWriter as the interceptor, which is undefined behavior. The
  new wrapper does not write a response when the timeout fires.

The `internal/http/interceptors` leaf package is removed (all logic moved
into `internal/http/interceptors.go`). External code never imported it
(internal/ scoped).

### Rate limiting: `NewThrottler` for app-scoped state

Use `NewThrottler(maxRequests, window)` for an isolated rate-limiter you
can `Close()` on shutdown. Each instance owns its own store and cleanup
goroutine, so two apps in the same process no longer share state and
tests can scope fresh state per case:

```go
t := ligo.NewThrottler(10, time.Minute)
defer t.Close()
cr.POST("", c.Create).Guard(t.Guard("ip"))
```

`ligo.ThrottleGuard("ip", 10, time.Minute)` still works using a
process-wide singleton. The fixes for `evictOldestEntries` (renamed
`evictArbitraryEntries`, godoc clarified) and the coarse mutex (per-entry
lock dropped) apply to both paths.

### Validation: contradictory format errors suppressed

The two-pass exhaustive validator stops surfacing "must be valid email"
or "must be one of …" for fields that already failed `required`. The
substitution-with-`"x"` shim used to satisfy `required` but fail format
tags itself, so users saw `required` AND `email` for the same empty
input. Non-format tags (`min`, `max`, `gte`, `lte`, etc.) still surface
in pass 2.

`validation.ValidateExhaustive` (exported) is now the single
implementation; `internal/http/pipes` imports it.

### `FormatChain` caps recursion at 32 levels

Cyclic error chains (legal under `errors.Unwrap`) and pathologically
deep ones no longer stack-overflow the formatter — chain depth is capped
and the output ends with `<truncated>` when the limit is hit.

### Module middleware no longer prefixes routes

Previously, attaching middleware to a module silently re-namespaced every
route in that module under `/<module-name>/...` — adding a logger
middleware turned `/users` into `/auth/users`. Routes now keep their
declared paths regardless of middleware attachment. The middleware is
applied through an isolated sub-group with an empty prefix, so other
modules are unaffected. If you actually want a module-level path prefix,
declare it explicitly in your route paths (`/auth/login` instead of
`/login` inside the auth module).

### `RouteBuilder.Handle()` panics on missing handler

`.GET("/x").Guard(...)` without `.Handle(fn)` used to silently skip
registration, producing a 404 at runtime instead of a registration-time
error. Builds now panic with `ligo: GET /x has no handler — call
.Handle(fn) on the route builder`. Build-time misuse is fatal; runtime
404s are reserved for unmatched paths.

### Guards return `ErrGuardDenied`

When a `Guard` returns `(false, nil)`, the wrapper now returns the
sentinel `ligo.ErrGuardDenied` instead of `fmt.Errorf("guard denied
access")`. ExceptionFilters detect it with `errors.Is(err,
ligo.ErrGuardDenied)` and typically map to HTTP 403:

```go
func AuthFilter(err error, ctx ligo.Context) error {
    if errors.Is(err, ligo.ErrGuardDenied) {
        return ctx.Forbidden("not allowed")
    }
    return err
}
```

### `Context.Stream` now takes `io.Reader`

`Stream(reader any) error` is now `Stream(reader io.Reader) error`.
Passing a non-reader value used to fail at runtime with a 400 (echo
adapter) or silently no-op (mock). Type-mismatched calls now fail at
compile time. If your reader needs to be closed, pass an `io.ReadCloser`
— the echo adapter detects `io.Closer` and closes after streaming.

### Logger options now compose correctly

`logger.WithJSON()`, `logger.WithText()`, and `logger.WithDebug()`
previously rebuilt the slog handler from scratch each time, so the last
applied option won — `New(WithJSON(), WithDebug(true))` returned a text
handler at debug level. Options now mutate a config struct and the
handler is built once at the end of `New`, so any ordering produces the
expected combination (JSON at debug, text at debug, etc.).

### Logger: `SetDebug` is concurrency-safe

`SetDebug` no longer rebuilds the handler. It updates a `slog.LevelVar`
shared with the handler, which is atomic by contract. Concurrent
`Debug`/`Info`/`SetDebug` calls now race-free.

### Internal: `app.Provider` interface adds `Fn() any`

The internal `internal/app.Provider` interface now requires `Fn() any`,
removing the prior `reflect.ValueOf(p).MethodByName("Fn").Call(...)`
shortcut. The root-package `ligo.Provider` struct already exposes
`Fn() any`, so user code is unaffected. Only matters for downstreams
that embed the internal interface directly.

---

## 0.9.0 → 0.9.6

No source-level breaking changes — drop-in upgrade. Several internal
improvements landed across the patch series that consumers will want
to know about.

### Race-safe `App.Container()` / `Adapter.Shutdown()`

The container handle and the underlying `http.Server` are now stored as
`atomic.Pointer[T]`. Tests that spawned `App.Run()` in a goroutine and
read `app.Container()` from the parent goroutine could race on Go 1.25
under `-race`; the upgrade removes the race without any API change.

### Go 1.25.10 baseline

`go.mod` requires Go 1.25.10. The bump is driven by `govulncheck` — Go
1.25.9 had three reachable stdlib CVEs (`GO-2026-4986`, `GO-2026-4977`,
`GO-2026-4971`) fixed in 1.25.10. Consumers should bump their own
`go.mod` directive to 1.25.10 or newer.

### `.golangci.yml` migrated to golangci-lint v2

The shared linter config is now v2-schema. v1.x is no longer compatible
with Go 1.25 (the v1 binary refuses to analyse newer Go versions).
`gopls`'s `infertypeargs` analyser is no longer available to
`golangci-lint v2` (lives in an internal package), so it now runs only
in the editor via gopls — see `CLAUDE.md` for the IDE-only note.

### CI: Node 24 actions

`actions/checkout@v6`, `actions/setup-go@v6`, and
`golangci/golangci-lint-action@v9` (Node-24 runtime) silence the Node
20 deprecation warning ahead of GitHub's 2026-09-16 removal.

---

## 0.8 → 0.9

### New: `HookedSingleton[T]`

`HookedFactory[T]` is lazy — the factory only runs when something else in
the DI graph resolves the type. For providers whose only purpose is to
attach lifecycle hooks (RPC handler registrations, schedulers, background
workers) there is no consumer, so the factory never runs and `Register`
never fires. Previously the only workarounds were injecting an unused
dependency into a controller or putting the registration inside a
module-level `OnInit`.

`HookedSingleton[T]` solves this directly: same semantics as
`HookedFactory` but resolved eagerly at startup, before any `OnInit` /
`OnBootstrap` hook executes.

```go
// Before — *OrderMessaging never instantiated, Register never called,
// queue bindings never set up.
ligo.HookedFactory[*OrderMessaging](NewOrderMessaging)

// After — eagerly resolved at startup, Register fires, OnBootstrap binds
// the broker handlers.
ligo.HookedSingleton[*OrderMessaging](NewOrderMessaging)
```

No breaking changes — existing `HookedFactory` usage continues to work
unchanged. Switch to `HookedSingleton` only for providers that are
"register-only" with no other DI consumer.

---

## 0.x → 0.5

This guide helps you migrate your Ligo applications from 0.x versions to 0.5.

## Overview

Ligo 0.5 is a feature-complete release that establishes the stable public API. While most code will work without changes, there are some important updates to be aware of.

## Breaking Changes

### None Expected

As of 0.4, no breaking changes are planned for 0.5. The 1.0 release primarily:

- Finalizes the public API
- Completes documentation
- Achieves target test coverage
- Establishes stability guarantees

## New Features in 0.5

### Built-in Pipes

New validation and transformation pipes are now available:

```go
// ValidationPipe for struct validation
type CreateUserInput struct {
    Name  string `validate:"required,min=3"`
    Email string `validate:"required,email"`
}

cr.POST("", c.Create).Pipe(ligo.ValidationPipe(&CreateUserInput{}))

// Retrieve the validated body in the handler:
func (c *Controller) Create(ctx ligo.Context) error {
    input := ligo.ValidatedBody[CreateUserInput](ctx) // *CreateUserInput
    // ...
}

// ParseIntPipe for integer parameters
cr.GET("/:id", c.Get).Pipe(ligo.ParseIntPipe("id"))

// UUIDPipe for UUID validation
cr.GET("/:id", c.Get).Pipe(ligo.UUIDPipe("id"))
```

### Built-in Guards

New authorization guards:

```go
// RolesGuard for role-based access
cr.GET("/admin", c.Admin).Guard(ligo.RolesGuard("user", "admin"))

// AdminGuard for admin-only access
cr.GET("/admin", c.Admin).Guard(ligo.AdminGuard("user"))

// ThrottleGuard for rate limiting
cr.GET("/api", c.API).Guard(ligo.ThrottleGuard(100, time.Minute))
```

### Built-in Interceptors

New request interceptors:

```go
// TimeoutInterceptor for request timeout
cr.GET("/slow", c.Slow).Intercept(ligo.TimeoutInterceptor(5 * time.Second))

// LoggingInterceptor for request logging
cr.GET("/", c.Index).Intercept(ligo.LoggingInterceptor(logger))
```

## Recommended Updates

### Add Validation to Your Controllers

If you're using manual validation, switch to `ValidationPipe`:

**Before:**
```go
func (c *Controller) Create(ctx ligo.Context) error {
    var input CreateUserInput
    if err := ctx.Bind(&input); err != nil {
        return ctx.BadRequest(err.Error())
    }

    // Manual validation
    if input.Name == "" {
        return ctx.BadRequest("name is required")
    }
    // ...
}
```

**After:**
```go
func (c *Controller) Create(ctx ligo.Context) error {
    var input CreateUserInput
    if err := ctx.Bind(&input); err != nil {
        return err
    }
    // Validation happens in the pipe
    return c.service.Create(input)
}

// In Routes()
cr.POST("", c.Create).Pipe(ligo.ValidationPipe(&CreateUserInput{}))
```

### Add Type Safety to Parameters

Use typed pipes for route parameters:

**Before:**
```go
func (c *Controller) Get(ctx ligo.Context) error {
    idStr := ctx.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        return ctx.BadRequest("invalid id")
    }
    // ...
}
```

**After:**
```go
func (c *Controller) Get(ctx ligo.Context) error {
    id := ctx.Get("id").(int) // Set by ParseIntPipe
    // ...
}

// In Routes()
cr.GET("/:id", c.Get).Pipe(ligo.ParseIntPipe("id"))
```

### Add Request Timeout

Protect slow endpoints with timeout interceptors:

```go
cr.GET("/slow", c.Slow).
    Intercept(ligo.TimeoutInterceptor(5 * time.Second)).
    Handle()
```

### Add Rate Limiting

Protect public APIs with throttling:

```go
cr.GET("/api", c.API).
    Guard(ligo.ThrottleGuard(100, time.Minute)).
    Handle()
```

## Deprecated Patterns

### Direct Context Access

While still supported, prefer using the builder pattern for route configuration:

**Before:**
```go
func (c *Controller) Routes(r ligo.Router) {
    r.Handle("GET", "/users", c.GetAllUsers)
}
```

**After:**
```go
func (c *Controller) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r)
    cr.GET("/users", c.GetAllUsers).
        Filter(GlobalExceptionFilter).
        Handle()
}
```

## Step-by-Step Migration

### 1. Update Dependencies

```bash
go get -u github.com/linkeunid/ligo@latest
go mod tidy
```

### 2. Update Imports

No import changes needed for 0.5.

### 3. Review Your Code

Check for any usage of internal packages (should not be used):

```go
// ❌ Don't use internal packages
import "github.com/linkeunid/ligo/internal/core/container"

// ✅ Use public API
import "github.com/linkeunid/ligo"
```

### 4. Add Validation Tags

Add validation tags to your input structs:

```go
type CreateUserInput struct {
    Name  string `validate:"required,min=3,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"gte=0,lte=150"`
}
```

### 5. Add Validation Pipes

Update your controllers to use validation pipes:

```go
func (c *Controller) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r.Group("/users"))

    cr.POST("", c.Create).
        Filter(GlobalExceptionFilter).
        Pipe(ligo.ValidationPipe(&CreateUserInput{})).
        Handle()

    cr.PUT("/:id", c.Update).
        Filter(GlobalExceptionFilter).
        Pipe(ligo.ParseIntPipe("id")).
        Pipe(ligo.ValidationPipe(&UpdateUserInput{})).
        Handle()
}
```

### 6. Add Guards for Authorization

Add role-based guards where needed:

```go
cr.DELETE("/:id", c.Delete).
    Filter(GlobalExceptionFilter).
    Guard(authGuard, ligo.RolesGuard("admin")).
    Handle()
```

### 7. Test Your Application

```bash
go test ./...
go run ./cmd/your-app
```

## Compatibility Notes

### Go Version

Ligo 0.5 requires Go 1.21 or later.

### Echo Adapter

The Echo adapter requires Echo v5:

```bash
go get github.com/labstack/echo/v5@latest
```

### Validation

Validation pipes require the validator package:

```bash
go get github.com/go-playground/validator/v10
```

## Rollback Plan

If you encounter issues after upgrading:

```bash
# Revert to previous version
go get github.com/linkeunid/ligo@v0.5.0
go mod tidy
```

## Get Help

If you encounter issues during migration:

1. Check the [GitHub Issues](https://github.com/linkeunid/ligo/issues)
2. Start a [GitHub Discussion](https://github.com/linkeunid/ligo/discussions)
3. Review the [Examples](examples.md) for updated patterns

## Summary

Most applications will upgrade to 0.5 without code changes. The main opportunities are:

1. Add validation pipes for better input validation
2. Add guards for authorization
3. Add interceptors for cross-cutting concerns
4. Use typed pipes for route parameters

These changes are optional but recommended for better code quality and maintainability.

---

## 0.5 → 0.6

This guide helps you migrate from 0.5 to 0.6.

## Overview

Ligo 0.6 is an internal restructuring that improves code organization and maintainability. **The public API remains unchanged** - most applications will upgrade without any code changes.

## Breaking Changes

### For Framework Users: None ✅

**No action required!** The public API is 100% backward compatible.

All your existing code will continue to work:
- All imports remain the same
- All function signatures unchanged
- All behavior preserved

### For Framework Contributors: Internal Changes

If you work on the Ligo framework itself, you'll need to update internal imports:

**Before:**
```go
import "github.com/linkeunid/ligo/internal/core/container"
```

**After:**
```go
import "github.com/linkeunid/ligo/internal/di"
```

## What Changed in 0.6

### Internal Package Reorganization

1. **DI Container Package Renamed**
   - Old: `internal/core/container`
   - New: `internal/di`
   - Reason: Clearer naming

2. **HTTP Layer Split into Subdirectories**
   - Old: Flat `internal/http/` (10 files)
   - New: Organized subdirectories
     - `internal/http/guards/` - Guard implementations
     - `internal/http/pipes/` - Pipe implementations
     - `internal/http/interceptors/` - Interceptor implementations
   - Reason: Better code organization

## Benefits

- **Better Organization:** Related code grouped together
- **Clearer Naming:** Package names reflect their purpose
- **Easier Navigation:** Faster to find what you need
- **Improved Maintainability:** Easier to understand and modify

## Step-by-Step Migration

### 1. Update Dependencies

```bash
go get -u github.com/linkeunid/ligo@v2.0.0
go mod tidy
```

### 2. Verify Your Application

```bash
go test ./...
go run ./cmd/your-app
```

### 3. (Framework Contributors Only) Update Internal Imports

If you work on the Ligo framework codebase:

```bash
# Update internal imports
find . -name "*.go" -type f -exec sed -i 's|internal/core/container|internal/di|g' {} \;
```

## Compatibility Notes

### Go Version

Ligo 0.6 requires Go 1.21 or later (same as 0.5).

### Dependencies

No new dependencies added in 0.6.

### Public API

The public API is unchanged - all your code will work as-is.

## Rollback Plan

If you encounter issues:

```bash
# Revert to previous version
go get github.com/linkeunid/ligo@v0.5.0
go mod tidy
```

## Summary

**For Users:** No changes required - upgrade and go!

**For Contributors:** Update internal imports from `core/container` to `di`.

The 0.6 release is all about **internal improvements** with **zero breaking changes** for users.
