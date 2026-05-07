# Migration Guide

This guide helps you migrate your Ligo applications between versions.

## [Migration Guide: 0.x → 0.5](#0x-→-05)

## [Migration Guide: 0.5 → 0.6](#05-→-06)

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
