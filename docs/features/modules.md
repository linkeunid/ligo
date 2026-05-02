# Modules

A `Module` is a self-contained unit of functionality that bundles providers, controllers, and middleware.

## Creating a Module

```go
func UserModule() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(...),
        ligo.Middlewares(...),
        ligo.Controllers(...),
        ligo.Imports(...),
    )
}
```

## Module Options

| Option | Description |
|--------|-------------|
| `Providers(providers...)` | Add providers to the module |
| `Controllers(constructors...)` | Add controller constructors |
| `Middlewares(constructors...)` | Add middleware constructors |
| `Imports(modules...)` | Import child modules |

## Complete Example

```go
package user

import "github.com/linkeunid/ligo"

func Module() ligo.Module {
    return ligo.NewModule("user",
        // Providers - dependency injection
        ligo.Providers(
            ligo.Factory[*UserRepo](NewUserRepo),
            ligo.Factory[*UserService](NewUserService),
        ),
        // Middleware - module-level with DI
        ligo.Middlewares(func(auth *AuthService) ligo.Middleware {
            return AuthMiddleware(auth)
        }),
        // Controllers - auto-injected dependencies
        ligo.Controllers(func(svc *UserService) ligo.Controller {
            return NewUserController(svc)
        }),
    )
}

type UserRepo struct{}
type UserService struct { repo *UserRepo }

func NewUserRepo() *UserRepo { return &UserRepo{} }
func NewUserService(repo *UserRepo) *UserService {
    return &UserService{repo: repo}
}
```

## Module Imports

Modules can import other modules as children. When a module is registered, **all controllers and middleware from its imported children are bound recursively** — you do not need to register child modules separately.

```go
func ApiModule() ligo.Module {
    return ligo.NewModule("api",
        ligo.Imports(
            user.Module(),
            auth.Module(),
            posts.Module(),
        ),
    )
}

// Registering only ApiModule is enough — routes from user, auth, and posts are all bound.
app.Register(ApiModule())
```

### Middleware Scoping

Module middleware applies only to that module's own controllers — it does not cascade to imported children. This follows NestJS convention. To protect all routes (including imported ones), use app-level middleware:

```go
app := ligo.New(
    ligo.WithRouter(echo.NewAdapter()),
    ligo.WithMiddleware(AuthMiddleware), // applies to all routes
)
```

### Diamond Imports

If two modules both import the same child, the child is processed exactly once (deduplicated by module name). There is no double-registration:

```go
// auth is imported by both user and file — it is bound once.
userModule := ligo.NewModule("user", ligo.Imports(auth.Module()), ...)
fileModule := ligo.NewModule("file", ligo.Imports(auth.Module()), ...)
mainModule := ligo.NewModule("main", ligo.Imports(userModule, fileModule))
```

> **Important:** Module names must be unique across the tree. Two different modules with the same name will have only the first one processed; the second is silently skipped.

### Dynamic Modules

Dynamic modules are fully expanded (factory called, all fields merged) before their imports are traversed, so child modules added by the factory are also bound correctly:

```go
ligo.Dynamic(func(opts ...any) ligo.Module {
    return ligo.NewModule("config",
        ligo.Imports(database.Module()), // this child is also bound
    )
})
```

## Provider Visibility

By default, providers are only visible within their module. Use `Export()` to make them available to sibling modules:

```go
func AuthModule() ligo.Module {
    return ligo.NewModule("auth",
        ligo.Providers(
            ligo.Export(ligo.Factory[*AuthService](NewAuthService)),
        ),
    )
}
```

Now `AuthService` can be injected in other modules:

```go
func UserModule() ligo.Module {
    return ligo.NewModule("user",
        ligo.Middlewares(func(auth *AuthService) ligo.Middleware {
            // AuthService is available because it was exported
            return AuthMiddleware(auth)
        }),
        ligo.Controllers(func(svc *UserService) ligo.Controller {
            return NewUserController(svc)
        }),
    )
}
```
