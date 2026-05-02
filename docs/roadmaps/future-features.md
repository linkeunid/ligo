# Future Features Roadmap

Features that could be added as separate packages/modules post-1.0.

---

## 🚧 In Development

### Microservices
**Status:** Design Complete, Implementation Pending
**Package:** `github.com/linkeunid/ligo/microservices`
**Documentation:** [Microservices Guide](../microservices.md)

**Features:**
- Transport layer abstractions (TCP, Redis, RabbitMQ, Kafka, NATS)
- Message broker interface
- Request/Response patterns
- Pub/Sub patterns
- Event-driven architecture
- Client and Server abstractions

**Key Concepts:**

The microservices package follows the same patterns as NestJS `@nestjs/microservices`:

1. **Broker Interface** - Abstraction over message brokers (Redis, RabbitMQ, Kafka)
2. **Transport Layer** - Different transport implementations
3. **Server/Client** - Microservice server and client abstractions
4. **Message Patterns** - Request/Response, Pub/Sub, Event-driven

**Example: Dynamic Microservice Module**

Similar to NestJS, you can create dynamic microservice modules:

```go
func RegisterMicroservice(config MicroserviceConfig) ligo.Module {
    return ligo.NewModule("microservice",
        ligo.Dynamic(
            func(opts ...any) ligo.Module {
                // Create module based on config
                return CreateMicroserviceInstance(config)
            },
            config,
        ),
    )
}
```

**Implementation Guide:**
See [Microservices Documentation](../microservices.md) for:
- Architecture overview
- Transport layer implementations
- Usage examples
- Comparison with NestJS
- Best practices
- Implementation roadmap

**Reference:** [rabbitmq/amqp091-go](https://github.com/rabbitmq/amqp091-go)

---

## High Priority

### Task Scheduling
**Status:** Planned
**Proposal:** `github.com/linkeunid/ligo/schedule`

### WebSocket Support
**Status:** Planned
**Proposal:** `github.com/linkeunid/ligo/ws`

---

## Medium Priority

### GraphQL Support
**Status:** Planned
**Proposal:** `github.com/linkeunid/ligo/graphql`

### gRPC Support
**Status:** Planned
**Proposal:** `github.com/linkeunid/ligo/grpc`

### OpenAPI/Swagger Integration
**Status:** Planned
**Proposal:** `github.com/linkeunid/ligo/swagger`

---

## Low Priority

### Message Queues (Additional Transports)
**Status:** Not implemented
**Proposals:**
- Kafka: `github.com/linkeunid/ligo/kafka`
- Redis Streams: `github.com/linkeunid/ligo/redis-streams`

### Caching Layer
**Status:** Not implemented
**Proposal:** `github.com/linkeunid/ligo/cache`

### Rate Limiting (Distributed)
**Status:** Basic in-memory ThrottleGuard exists
**Proposal:** Redis-based distributed rate limiting

### Tracing/Observability
**Status:** Not implemented
**Proposal:** OpenTelemetry integration

---

## Contributing

See [Package Ecosystem](ecosystem.md#contributing) for guidelines on implementing ecosystem packages.
