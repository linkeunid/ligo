# Migration Guide: 0.x → 1.0

This guide helps you migrate your Ligo applications from 0.x versions to 1.0.

## Overview

Ligo 1.0 is a major milestone that establishes the stable public API. While most code will work without changes, there are some important updates to be aware of.

## Breaking Changes

### None Expected

As of 0.9, no breaking changes are planned for 1.0. The 1.0 release primarily:

- Finalizes the public API
- Completes documentation
- Achieves target test coverage
- Establishes stability guarantees

## New Features in 1.0

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

No import changes needed for 1.0.

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

Ligo 1.0 requires Go 1.21 or later.

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
go get github.com/linkeunid/ligo@v0.9.0
go mod tidy
```

## Get Help

If you encounter issues during migration:

1. Check the [GitHub Issues](https://github.com/linkeunid/ligo/issues)
2. Start a [GitHub Discussion](https://github.com/linkeunid/ligo/discussions)
3. Review the [Examples](examples.md) for updated patterns

## Summary

Most applications will upgrade to 1.0 without code changes. The main opportunities are:

1. Add validation pipes for better input validation
2. Add guards for authorization
3. Add interceptors for cross-cutting concerns
4. Use typed pipes for route parameters

These changes are optional but recommended for better code quality and maintainability.
