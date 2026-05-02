# Ligo

A modular Go framework with lightweight dependency injection, inspired by NestJS.

[![Go Version](https://img.shields.io/badge/go-1.25.9-blue)](https://go.dev/dl)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Status](https://img.shields.io/badge/status-0.9%20beta-yellow)](https://github.com/linkeunid/ligo)

> **Note:** Ligo is at **v0.9** beta. Production-ready with Guards, Pipes, Interceptors, Exception Filters, and HTTP response helpers.

## Features

- **Modular Architecture** - Self-contained modules with providers, controllers, and middleware
- **Dependency Injection** - Automatic dependency resolution with zero boilerplate
- **HTTP Routing** - Adapter-agnostic router interface with Echo v5 adapter
- **Guards** - Authorization with composable guard functions
- **Pipes** - Validation and transformation with composable pipes
- **Interceptors** - Logging, caching, and response transformation
- **Exception Filters** - Error handling and HTTP response conversion
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
- [Guards](docs/features/guards.md) - Authorization with guards
- [Pipes](docs/features/pipes.md) - Validation and transformation with pipes
- [Interceptors](docs/features/interceptors.md) - Logging, caching, and transformation with interceptors
- [Exception Filters](docs/features/exception-filters.md) - Error handling and HTTP response conversion
- [DI Container](docs/features/di-container.md) - How the DI container works

## Roadmap

- **Current Version:** 0.9 - Production ready with all core features
- **In Development:** `github.com/linkeunid/ligo/microservices` - RabbitMQ-based microservices (message queue, RPC, event-driven architecture)

See [Roadmaps](docs/roadmaps/) for:
  - [Package Ecosystem](docs/roadmaps/ecosystem.md) - Separate packages for DB, microservices, etc.
  - [1.0 Release Plan](docs/roadmaps/1.0-release.md) - Timeline to production release
  - [Future Features](docs/roadmaps/future-features.md) - WebSocket, GraphQL, Scheduling
  - [Adapter Proposals](docs/roadmaps/adapter-proposals.md) - Fiber, Gin, Chi adapters

## Examples

See the [ligo-boilerplate](https://github.com/linkeunid/ligo-boilerplate) repository for complete examples:
- **REST API** - Full CRUD operations with response helpers
- **Authentication** - JWT-style auth with guards and role-based access
- **Authorization** - Custom guards, roles guard, admin-only endpoints
- **File Upload** - Multipart file upload with streaming downloads

See [Examples Guide](docs/examples.md) for detailed documentation and API usage.

**Note:** Database integration and microservices will be provided as separate packages (like `@nestjs/typeorm` and `@nestjs/microservices`). See [Package Ecosystem](docs/roadmaps/ecosystem.md).

## License

MIT License - see [LICENSE](LICENSE) for details.
