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

Modules can import other modules as children:

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
