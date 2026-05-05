# Ligo

A modular Go framework with lightweight dependency injection, inspired by NestJS.

[![Go Version](https://img.shields.io/badge/go-1.21+-blue)](https://go.dev/dl)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-217%20passing-brightgreen)](https://github.com/linkeunid/ligo)
[![Coverage](https://img.shields.io/badge/coverage-48.9%25-yellow)](https://github.com/linkeunid/ligo)

> **Note:** Ligo **v1.0** is ready. All requirements completed including comprehensive documentation, integration tests, performance benchmarks, and stability guarantees.

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
- [Migration Guide](docs/migration.md) - Migrating from 0.x to 1.0
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

## Roadmap

- [x] **Current Version:** 1.0 (All requirements completed)
- **Test Coverage:** 208 tests passing, 50.4% coverage
- **Documentation:** Complete (API guides, migration, best practices, performance, deployment, stability, microservices)
- **Next:** `github.com/linkeunid/ligo/microservices` - Message brokers, event-driven architecture

See [Roadmaps](docs/roadmaps/) for:
  - [x] [1.0 Release Plan](docs/roadmaps/1.0-release.md) - Complete
  - [Package Ecosystem](docs/roadmaps/ecosystem.md) - Separate packages for DB, microservices, etc.
  - [Microservices Guide](docs/microservices.md) - How to build distributed systems with Ligo
  - [Future Features](docs/roadmaps/future-features.md) - WebSocket, GraphQL, Scheduling
  - [Adapter Proposals](docs/roadmaps/adapter-proposals.md) - Fiber, Gin, Chi adapters

## Examples

See the [ligo-boilerplate](https://github.com/linkeunid/ligo-boilerplate) repository for complete examples:
- **REST API** - Full CRUD operations with response helpers
- **Authentication** - JWT-style auth with guards and role-based access
- **Authorization** - Custom guards, roles guard, admin-only endpoints
- **File Upload** - Multipart file upload with streaming downloads

See [Examples Guide](docs/examples.md) for detailed documentation and API usage.

**Note:** Database integration and microservices will be provided as separate packages (like `@nestjs/typeorm` and `@nestjs/microservices`). See [Microservices Guide](docs/microservices.md) for implementation details.

## License

MIT License - see [LICENSE](LICENSE) for details.
