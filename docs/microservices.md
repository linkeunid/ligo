# Ligo Microservices Package

This guide explains the `github.com/linkeunid/ligo-microservices` package for building distributed systems with Ligo.

## Overview

The microservices package provides transport layer implementations for message brokers, following patterns similar to NestJS `@nestjs/microservices`.

## Package Structure

```
github.com/linkeunid/ligo/microservices/
├── transport/          # Transport layer implementations
│   ├── tcp/           # TCP transport for direct messaging
│   ├── redis/         # Redis transport (pub/sub, RPC)
│   ├── rabbitmq/      # RabbitMQ transport
│   ├── kafka/         # Kafka transport
│   └── nats/          # NATS transport
├── server/            # Microservice server
├── client/            # Microservice client
├── broker/            # Message broker abstraction
├── pattern/           # Messaging patterns
└── options/           # Configuration options
```

See [External Packages Guide](external-packages.md) for how to create packages like this.

## Core Abstractions

### Broker Interface

```go
// broker/broker.go
package broker

import (
    "context"
)

// Message represents a message sent through the broker.
type Message struct {
    Pattern string            // Routing pattern or topic
    Data    []byte            // Message payload
    Options map[string]any    // Additional options
}

// Handler handles incoming messages.
type Handler func(ctx context.Context, msg Message) error

// Broker is the abstraction for message brokers.
type Broker interface {
    // Publish sends a message to the broker.
    Publish(ctx context.Context, pattern string, data []byte) error
    
    // Subscribe registers a handler for a pattern.
    Subscribe(ctx context.Context, pattern string, handler Handler) error
    
    // Close closes the broker connection.
    Close() error
}
```

### Server Interface

```go
// server/server.go
package server

type Server struct {
    broker Broker
}

func NewServer(broker Broker) *Server {
    return &Server{broker: broker}
}

// Listen starts listening for messages on a pattern.
func (s *Server) Listen(ctx context.Context, pattern string, handler Handler) error {
    return s.broker.Subscribe(ctx, pattern, handler)
}

// Publish sends a message to a pattern.
func (s *Server) Publish(ctx context.Context, pattern string, data []byte) error {
    return s.broker.Publish(ctx, pattern, data)
}
```

## Transport Implementations

### Redis Transport

```go
// transport/redis/broker.go
package redis

import (
    "context"
    "github.com/redis/go-redis/v9"
    "github.com/linkeunid/ligo/microservices/broker"
)

type Broker struct {
    client *redis.Client
}

func NewBroker(client *redis.Client) *Broker {
    return &Broker{client: client}
}

func (b *Broker) Publish(ctx context.Context, pattern string, data []byte) error {
    return b.client.Publish(ctx, pattern, data).Err()
}

func (b *Broker) Subscribe(ctx context.Context, pattern string, handler broker.Handler) error {
    pubsub := b.client.Subscribe(ctx, pattern)
    if pubsub.Err() != nil {
        return pubsub.Err()
    }

    ch := pubsub.Channel()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case msg := <-ch:
            handler(ctx, broker.Message{
                Pattern: msg.Channel,
                Data:    msg.Payload,
            })
        }
    }
}

func (b *Broker) Close() error {
    return b.client.Close()
}
```

## Usage Examples

### Example 1: Redis Pub/Sub Module

```go
// microservice-redis/module.go
package microserviceredis

import (
    "github.com/linkeunid/ligo"
    "github.com/linkeunid/ligo/microservices/server"
    "github.com/linkeunid/ligo/microservices/transport/redis"
    "github.com/redis/go-redis/v9"
)

type Config struct {
    Addr string
}

func Module(config Config) ligo.Module {
    return ligo.NewModule("microservice-redis",
        ligo.Providers(
            ligo.Factory[*redis.Client](func() *redis.Client {
                return redis.NewClient(&redis.Options{
                    Addr: config.Addr,
                })
            }),
            ligo.Factory[*server.Server](func(client *redis.Client) *server.Server {
                broker := redis.NewBroker(client)
                return server.NewServer(broker)
            }),
        ),
    )
}
```

### Example 2: Using in Your App

```go
import (
    "github.com/linkeunid/ligo"
    msredis "github.com/linkeunid/ligo-microservices-redis"
)

func main() {
    app := ligo.New(
        ligo.WithRouter(echo.NewAdapter()),
        ligo.WithAddr(":8080"),
    )
    
    app.Register(
        msredis.Module(msredis.Config{
            Addr: "localhost:6379",
        }),
        user.Module(),
    )
    
    app.Run()
}
```

### Example 3: Publishing Events

```go
type UserService struct {
    server *microservices.Server
}

func (s *UserService) CreateUser(ctx ligo.Context) error {
    // Create user...
    
    // Publish event
    event := UserCreatedEvent{
        ID:   user.ID,
        Name: user.Name,
    }
    data, _ := json.Marshal(event)
    
    s.server.Publish(ctx.Context(), "user.created", data)
    
    return ctx.Created(user)
}
```

## Comparison with NestJS

| NestJS | Ligo |
|--------|------|
| `@nestjs/microservices` | `github.com/linkeunid/ligo/microservices` |
| `ClientProxy` | `client.Client` |
| `Server` | `server.Server` |
| `Transport` | `transport/` |
| `ClientsModule` | Module with providers |
| `TcpClientOptions` | `tcp.Options` |
| `RedisClientOptions` | `redis.Options` |

## Implementation Status

- [ ] Core abstractions (Broker, Server, Client)
- [ ] TCP transport
- [ ] Redis transport
- [ ] RabbitMQ transport
- [ ] Documentation

## See Also

- [External Packages Guide](external-packages.md) - How to create external Ligo packages
- [Package Development Guide](package-development.md) - General package development patterns
