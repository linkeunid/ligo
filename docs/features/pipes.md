# Pipes

Pipes transform input data before it reaches the handler. They're used for validation, parsing, and transformation.

## Pipe Signature

```go
type Pipe func(any) (any, error)
```

## Creating a Pipe

```go
func ValidationPipe() Pipe {
    return func(input any) (any, error) {
        // Validate input
        if err := validate(input); err != nil {
            return nil, err
        }
        return input, nil
    }
}
```

## Using Pipes

```go
func (c *UserController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r.Group("/users"))

    // Apply validation pipe
    cr.POST("/", c.Create).
        Guard(AuthGuard()).
        Pipe(ValidationPipe())

    // Multiple pipes - executed in order
    cr.PUT("/:id", c.Update).
        Guard(AuthGuard()).
        Pipe(ParseIntPipe("id"), ValidationPipe())
}
```

## Common Pipe Patterns

### Validation Pipe

```go
func ValidationPipe[T any]() Pipe {
    return func(input any) (any, error) {
        data, ok := input.(T)
        if !ok {
            return nil, errors.New("invalid type")
        }

        if err := validator.Validate(data); err != nil {
            return nil, err
        }

        return data, nil
    }
}
```

### Transform Pipe

```go
func TrimSpacePipe() Pipe {
    return func(input any) (any, error) {
        if str, ok := input.(string); ok {
            return strings.TrimSpace(str), nil
        }
        return input, nil
    }
}
```

### Parsing Pipe

```go
func ParseIntPipe() Pipe {
    return func(input any) (any, error) {
        if str, ok := input.(string); ok {
            num, err := strconv.Atoi(str)
            if err != nil {
                return nil, err
            }
            return num, nil
        }
        return input, nil
    }
}
```

## Pipe Execution

Pipes execute in the order specified, transforming the data sequentially:

```go
// Execution flow: original -> Trim -> ParseInt -> Validate -> handler
cr.POST("/data", c.Create).
    Pipe(TrimSpacePipe(), ParseIntPipe(), ValidationPipe())
```

## Error Handling

If a pipe returns an error, the chain stops and the error is returned:

```go
func ValidationPipe() Pipe {
    return func(input any) (any, error) {
        if invalid {
            return nil, errors.New("validation failed")
        }
        return input, nil
    }
}

// The handler won't be called if validation fails
cr.POST("/", c.Create).Pipe(ValidationPipe())
```
