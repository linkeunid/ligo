# Ligo

A modular Go framework with lightweight dependency injection, inspired by NestJS.

[![Go Version](https://img.shields.io/badge/go-1.25+-blue)](https://go.dev/dl)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-377%20passing-brightgreen)](https://github.com/linkeunid/ligo)
[![Coverage](https://img.shields.io/badge/coverage-77.7%25-brightgreen)](https://github.com/linkeunid/ligo)

> **Note:** Ligo **v0.11.0** is available. `Context` is now a concrete struct that wraps the new `Adapter` interface — handlers receive `*ligo.Context` and adapters implement 14 methods instead of 45. `ctx.OK(...)`, `ctx.BadRequest(...)`, `ctx.QueryInt(...)` all keep working. See the [Migration Guide](docs/migration.md).

## Features

- **Modular Architecture** - Self-contained modules with providers, controllers, and middleware
- **Dependency Injection** - Automatic dependency resolution with zero boilerplate
- **Lifecycle Hooks** - OnModuleInit, OnApplicationBootstrap, BeforeApplicationShutdown, OnApplicationShutdown, OnModuleDestroy
- **HTTP Routing** - Adapter-agnostic router interface with Echo v5 adapter
- **Guards** - Authorization with composable guard functions
- **Pipes** - Validation and transformation with composable pipes
- **Interceptors** - Logging, caching, and response transformation
- **Exception Filters** - Error handling and HTTP response conversion
- **Type-Safe** - Full type safety with generics

## Installation

```bash
# Get the latest version (v0.11.0)
go get github.com/linkeunid/ligo@latest

# Or specify a version
go get github.com/linkeunid/ligo@v0.11.0
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
            ligo.Controllers(NewHelloController),
        ),
    )

    app.Run()
}

type helloController struct { msg string }

func NewHelloController(msg string) *helloController {
    return &helloController{msg: msg}
}

func (c *helloController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r)
    cr.GET("/", c.Hello).Handle()
}

func (c *helloController) Hello(ctx ligo.Context) error {
    return ctx.OK(map[string]string{"message": c.msg})
}
```

## Documentation

**Guides:**
- [Examples Guide](docs/examples.md) - Detailed documentation and API usage
- [Migration Guide](docs/migration.md) - Migrating from 0.x to 0.6.0
- [Best Practices](docs/best-practices.md) - Development patterns and conventions
- [Performance Tuning](docs/performance-tuning.md) - Optimization and profiling
- [Deployment Guide](docs/deployment.md) - Docker, Kubernetes, Cloud platforms
- [Stability Policy](docs/stability.md) - Versioning and backward compatibility
- [Package Development](docs/package-development.md) - How to create new Ligo packages
- [External Packages](docs/external-packages.md) - Creating third-party integration packages
- [Microservices](docs/microservices.md) - Distributed systems with Ligo

**Features:**
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

## CLI

Scaffold projects and generate Clean Architecture boilerplate with **[ligo-cli](https://github.com/linkeunid/ligo-cli)**:

```bash
go install github.com/linkeunid/ligo-cli/cmd/ligo@latest
```

```bash
ligo new my-app            # scaffold a new project
ligo new my-app --full     # full boilerplate (users, file upload, auth)
ligo new my-app --runner   # background worker/runner

ligo g res product         # generate all layers (entity, usecase, repo, controller, module)
ligo g co product --full   # generate controller only
ligo g run email           # generate background worker

ligo serve                 # go run ./cmd/api/
ligo serve --watch         # auto-reload on file changes
ligo work email            # run a background worker
ligo build                 # go build -o bin/app ./cmd/api/
```

See [ligo-cli](https://github.com/linkeunid/ligo-cli) for the full command reference.

## Ecosystem

| Package | Description |
|---------|-------------|
| [ligo](https://github.com/linkeunid/ligo) | Core framework |
| [ligo-cli](https://github.com/linkeunid/ligo-cli) | CLI — scaffolding and code generation |
| [ligo-boilerplate](https://github.com/linkeunid/ligo-boilerplate) | Starter project with Clean Architecture |
| [ligo-memory](https://github.com/linkeunid/ligo-memory) | In-memory store for dev/testing |
| [ligo-validator](https://github.com/linkeunid/ligo-validator) | Validator provider |

**Coming next:** `ligo-microservices` (RabbitMQ), `ligo-db` (pgx, no ORM), `ligo-schedule`, `ligo-ws` — see [Sneak Peek](docs/roadmaps/sneak-peek.md).

## Examples

See the [ligo-boilerplate](https://github.com/linkeunid/ligo-boilerplate) repository for complete examples:
- **REST API** - Full CRUD operations with response helpers
- **Authentication** - JWT-style auth with guards and role-based access
- **Authorization** - Custom guards, roles guard, admin-only endpoints
- **File Upload** - Multipart file upload with streaming downloads

See [Examples Guide](docs/examples.md) for detailed documentation and API usage.

## License

MIT License - see [LICENSE](LICENSE) for details.
