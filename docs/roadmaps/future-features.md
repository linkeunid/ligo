# Future Features Roadmap

Features that could be added as separate packages/modules post-1.0.

---

## 🚧 In Development

### Microservices (RabbitMQ)
**Status:** In Development
**Package:** `github.com/linkeunid/ligo/microservices`

**Features:**
- RabbitMQ transport layer
- Message producer/consumer patterns
- RPC patterns over message queue
- Event-driven architecture
- Service discovery (Consul, etcd)
- Circuit breaker for resilience

```go
// Proposed API
func Module() ligo.Module {
    return ligo.NewModule("order",
        microservices.WithRabbitMQConsumer(
            "order-events",
            microservices.WithHandler(func(msg *amqp.Delivery) error {
                // Handle order event
                return processOrder(msg)
            }),
        ),
        microservices.WithRabbitMQProducer(
            "user-events",
        ),
    )
}

// RPC over RabbitMQ
func (s *OrderService) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
    return s.userService.RPC("user-service.CreateUser", ctx, req)
}
```

**Implementation:**
- RabbitMQ client wrapper with connection pooling
- Message serialization/deserialization
- Retry policies and dead letter queues
- Health checks for consumer lag
- Graceful shutdown for consumers

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
