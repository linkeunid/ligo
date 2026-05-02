# Pipes

Pipes run before the handler to validate, parse, or transform request data.

## Pipe Signature

```go
type Pipe func(ctx Context) (any, error)
```

If a pipe returns an error, the chain stops and the handler is not called.

## Built-in Pipes

### ValidationPipe

Binds the request body to `T`, validates it using `validate` struct tags, and stores the result in context. Retrieve it in the handler with `ValidatedBody[T]`:

```go
type CreateUserInput struct {
    Name  string `json:"name"  validate:"required,min=3"`
    Email string `json:"email" validate:"required,email"`
}

func (c *UserController) Create(ctx ligo.Context) error {
    input := ligo.ValidatedBody[CreateUserInput](ctx) // *CreateUserInput
    user, err := c.service.Create(*input)
    if err != nil {
        return err
    }
    return ctx.Created(user)
}

func (c *UserController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r.Group("/users"))
    cr.POST("", c.Create).
        Pipe(ligo.ValidationPipe(&CreateUserInput{})).
        Handle()
}
```

`ValidatedBody[T]` panics with a clear message if `ValidationPipe` was not added to the route — catching misconfigurations at startup rather than silently returning nil.

### ParseIntPipe

Parses a path parameter as `int` and stores it in context under the parameter name:

```go
func (c *UserController) GetByID(ctx ligo.Context) error {
    id := ctx.Get("id").(int)
    // ...
}

cr.GET("/:id", c.GetByID).Pipe(ligo.ParseIntPipe("id")).Handle()
```

### ParseBoolPipe

Parses a path parameter as `bool`:

```go
cr.GET("/:active", c.List).Pipe(ligo.ParseBoolPipe("active")).Handle()
```

### UUIDPipe

Validates that a path parameter is a valid UUID:

```go
cr.GET("/:id", c.Get).Pipe(ligo.UUIDPipe("id")).Handle()
```

### TrimPipe

Trims whitespace from a query or body field:

```go
cr.POST("", c.Create).
    Pipe(ligo.TrimPipe("name"), ligo.TrimPipe("email")).
    Pipe(ligo.ValidationPipe(&CreateUserInput{})).
    Handle()
```

## Chaining Pipes

Pipes execute in the order specified:

```go
cr.PUT("/:id", c.Update).
    Pipe(ligo.ParseIntPipe("id")).
    Pipe(ligo.ValidationPipe(&UpdateUserInput{})).
    Handle()
```

Multiple pipes can also be passed to a single `.Pipe()` call:

```go
cr.PUT("/:id", c.Update).
    Pipe(ligo.ParseIntPipe("id"), ligo.ValidationPipe(&UpdateUserInput{})).
    Handle()
```

## Validating Multiple DTOs

**Do not call `ValidationPipe` twice on the same route.** Each call reads the HTTP request body via `ctx.Bind` — the body stream can only be read once, so the second call gets an empty body. Both calls also store under the same context key, so the second would overwrite the first.

**Wrong:**

```go
cr.POST("", c.Create).
    Pipe(ligo.ValidationPipe(&dto.CreateUserInput{})).
    Pipe(ligo.ValidationPipe(&dto.ExtraInput{})).   // body already consumed
    Handle()
```

**Right — use a single composite DTO:**

```go
type CreateUserInput struct {
    dto.BaseUserInput                               // embed shared fields
    Role string `json:"role" validate:"required"`
}

cr.POST("", c.Create).
    Pipe(ligo.ValidationPipe(&CreateUserInput{})).
    Handle()
```

**Right — combine path param extraction with body validation:**

```go
cr.PUT("/:id", c.Update).
    Pipe(ligo.ParseIntPipe("id")).                  // path param → ctx.Get("id").(int)
    Pipe(ligo.ValidationPipe(&dto.UpdateUserInput{})).
    Handle()

func (c *UserController) Update(ctx ligo.Context) error {
    id := ctx.Get("id").(int)
    input := ligo.ValidatedBody[dto.UpdateUserInput](ctx)
    // ...
}
```

## Custom Pipes

```go
func PositiveIntPipe(param string) ligo.Pipe {
    return func(ctx ligo.Context) (any, error) {
        str := ctx.Param(param)
        n, err := strconv.Atoi(str)
        if err != nil || n <= 0 {
            return nil, fmt.Errorf("param %q must be a positive integer", param)
        }
        ctx.Set(param, n)
        return n, nil
    }
}
```
