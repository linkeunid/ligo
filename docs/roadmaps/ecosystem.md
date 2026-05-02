# Ligo Package Ecosystem

Ligo follows a modular approach where advanced features are provided as separate packages, similar to NestJS ecosystem packages like `@nestjs/microservices` and `@nestjs/typeorm`.

## Table of Contents

- [Priority Roadmap](#priority-roadmap)
- [Core Framework](#core-framework)
- [Separate Packages (Planned)](#separate-packages-planned)
  - [Database Integration](#database-integration)
  - [Microservices](#microservices)
  - [WebSocket](#websocket)
  - [GraphQL](#graphql)
  - [Scheduling](#scheduling)
- [Comparison with NestJS](#comparison-with-nestjs)
- [Creating a Package](#creating-a-package)
- [Contributing](#contributing)
- [Philosophy](#philosophy)

---

## Priority Roadmap

### 🚧 In Development: **Microservices (RabbitMQ)**

**Package:** `github.com/linkeunid/ligo/microservices`

- RabbitMQ transport layer
- Message producer/consumer patterns
- RPC over RabbitMQ
- Service discovery (Consul, etcd)
- Event-driven architecture
- Circuit breaker pattern

**Why first?** Enterprise need for distributed systems, message queue integration, microservices communication.

### 📋 Planned (Priority Order)

1. **Scheduling** - Cron jobs, interval tasks (simple, high value)
2. **WebSocket** - Real-time features (chat, notifications)
3. **Database** - Repository pattern helpers (works with gorm/sqlx/ent)
4. **GraphQL** - Code-first schema, resolver registration
5. **Message Queues** - Kafka, Redis Streams (additional transports)

---

## Core Framework

**Package:** `github.com/linkeunid/ligo`

The core framework provides:
- HTTP routing with adapter pattern
- Dependency injection container
- Module system
- Guards, Pipes, Interceptors, Exception Filters
- Lifecycle hooks
- Built-in utilities (validation, auth helpers, logging)

**This is all you need for:**
- REST APIs
- CRUD operations
- Authentication/Authorization
- File uploads
- Basic web applications

---

## Separate Packages (Planned)

### Database Integration

**Proposed:** `github.com/linkeunid/ligo/database` or `github.com/linkeunid/ligo/sql`

Similar to `@nestjs/typeorm`, this package provides:
- Repository pattern abstraction
- Transaction management
- Migration support
- Connection pooling
- Integration with popular Go ORMs:
  - `gorm.io/gorm`
  - `ent/ent`
  - `sqlx`
  - `upper/db`

**Example usage:**
```go
import "github.com/linkeunid/ligo/sql"

func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(
            sql.WithRepository[*UserRepository](
                sql.NewGormRepository(db),
            ),
        ),
    )
}
```

**Why separate package?**
- Database choice is opinionated
- Not all apps need a database
- Keeps core framework lightweight
- Allows community to create adapters for their preferred ORM

---

### Microservices

**Proposed:** `github.com/linkeunid/ligo/microservices`

Similar to `@nestjs/microservices`, this package provides:
- TCP transport layer
- Redis transport layer
- Kafka transport layer
- Service discovery (Consul, etcd)
- Event-driven patterns
- RPC patterns
- Circuit breakers

**Example usage:**
```go
import "github.com/linkeunid/ligo/microservices"

func Module() ligo.Module {
    return ligo.NewModule("order",
        microservices.WithClient(
            microservices.NewTCPClient("inventory-service:3000"),
        ),
        microservices.WithServer(
            microservices.NewTCPServer(":3000"),
        ),
    )
}
```

**Why separate package?**
- Microservices add complexity
- Not all apps need distributed systems
- Transport layer choice is opinionated
- Keeps core focused on HTTP

---

### WebSocket

**Proposed:** `github.com/linkeunid/ligo/ws`

Similar to `@nestjs/websockets`, this package provides:
- WebSocket upgrade handler
- Connection management (Hub pattern)
- Room/channel support
- Message broadcasting
- Connection lifecycle hooks

---

### GraphQL

**Proposed:** `github.com/linkeunid/ligo/graphql`

Similar to `@nestjs/graphql`, this package provides:
- Schema definition (code-first or SDL)
- Resolver registration via DI
- DataLoader integration
- Subscription support

---

### Scheduling

**Proposed:** `github.com/linkeunid/ligo/schedule`

Similar to `@nestjs/schedule`, this package provides:
- Cron expression parser
- Interval-based scheduling
- Job registration
- Concurrent execution with worker pool

---

## Comparison with NestJS

| NestJS | Ligo Equivalent | Status |
|--------|-----------------|--------|
| `@nestjs/common` | `github.com/linkeunid/ligo` (core) | ✅ Complete |
| `@nestjs/typeorm` | `github.com/linkeunid/ligo/database` | 📋 Planned |
| `@nestjs/microservices` | `github.com/linkeunid/ligo/microservices` | 📋 Planned |
| `@nestjs/websockets` | `github.com/linkeunid/ligo/ws` | 📋 Planned |
| `@nestjs/graphql` | `github.com/linkeunid/ligo/graphql` | 📋 Planned |
| `@nestjs/schedule` | `github.com/linkeunid/ligo/schedule` | 📋 Planned |
| `@nestjs/swagger` | `github.com/linkeunid/ligo/swagger` | 📋 Planned |

---

## Creating a Package

If you want to create a Ligo ecosystem package:

1. **Name it:** `github.com/linkeunid/ligo/<name>`
2. **Import core:** `import "github.com/linkeunid/ligo"`
3. **Follow patterns:**
   - Use DI for dependency injection
   - Integrate with Module system
   - Provide Builder/Chain pattern where applicable
4. **Document:** Add examples and usage guide

---

## Contributing

Want to build an ecosystem package?

1. Open an issue proposing the package
2. Discuss API design
3. Implement following Ligo patterns
4. Add comprehensive tests and docs
5. Submit for review

---

## Philosophy

**Small core, pluggable extras:**

- Core framework handles 80% of use cases
- Separate packages handle specialized needs
- Users only pay (in complexity) for what they use
- Community can create competing implementations
