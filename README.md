# Ligo

A modular Go framework with lightweight dependency injection, inspired by NestJS.

[![Go Version](https://img.shields.io/badge/go-1.25.9-blue)](https://go.dev/dl)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Features

- **Modular Architecture** - Self-contained modules with providers, controllers, and middleware
- **Dependency Injection** - Automatic dependency resolution with zero boilerplate
- **HTTP Routing** - Adapter-agnostic router interface with Echo v5 adapter
- **Type-Safe** - Full type safety with generics

## Installation

```bash
go get github.com/linkeunid/ligo
```

## Quick Start

```go
package main

import (
    "github.com/linkeunid/ligo"
    "github.com/linkeunid/ligo/adapters/echo"
)

func main() {
    router := echo.NewAdapter()
    app := ligo.New(
        ligo.WithRouter(router),
        ligo.WithAddr(":8080"),
    )

    app.Register(
        ligo.NewModule("hello",
            ligo.Providers(ligo.Value("Hello, World!")),
            ligo.Controllers(func(msg string) ligo.Controller {
                return &helloController{msg: msg}
            }),
        ),
    )

    app.Run()
}

type helloController struct { msg string }

func (c *helloController) Routes(r ligo.Router) {
    r.Handle("GET", "/", func(ctx ligo.Context) error {
        return ctx.String(200, c.msg)
    })
}
```

## Documentation

- [App & Lifecycle](docs/features/app.md) - Application configuration and lifecycle hooks
- [Modules](docs/features/modules.md) - Creating and organizing modules
- [Providers](docs/features/providers.md) - Dependency injection with providers
- [Controllers](docs/features/controllers.md) - Handling HTTP requests
- [Middleware](docs/features/middleware.md) - Global, module-level, and route-level middleware
- [DI Container](docs/features/di-container.md) - How the DI container works

## Examples

```bash
# Basic example
cd examples/basic-api && go run .

# Full modular example
cd examples/api && go run .
```

## License

MIT License - see [LICENSE](LICENSE) for details.
