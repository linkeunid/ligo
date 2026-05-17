# Sneak Peek — What's Coming Next

What's being built next for the Ligo ecosystem. These are upcoming packages, not part of the core framework.

---

## 🔥 Priority 1 — In Active Focus

### Microservices + RabbitMQ

**Package:** `github.com/linkeunid/ligo-microservices`
**Transport (first):** RabbitMQ via [`rabbitmq/amqp091-go`](https://github.com/rabbitmq/amqp091-go)

Mirrors the experience of `@nestjs/microservices` — but Go-idiomatic. Both messaging patterns supported from day one:

**Request/Response (RPC):**
```go
// Consumer side — handles incoming RPC requests
func (s *OrderService) Register(r *ligo.HookRegistry) {
    r.OnBootstrap(s.StartListening)
}

func (s *OrderService) StartListening() error {
    return s.broker.Handle("orders.create", s.HandleCreate)
}

func (s *OrderService) HandleCreate(ctx context.Context, msg *microservices.Message) (any, error) {
    var input CreateOrderInput
    msg.Decode(&input)
    order, err := s.usecase.Create(ctx, input)
    return order, err
}

// Producer side — sends RPC request, waits for reply
order, err := client.Send(ctx, "orders.create", CreateOrderInput{...})
```

**Event / Pub-Sub (fire and forget):**
```go
// Publisher
s.broker.Emit("order.created", OrderCreatedEvent{OrderID: order.ID})

// Subscriber
s.broker.On("order.created", s.HandleOrderCreated)
```

**Module setup:**
```go
func Module() ligo.Module {
    return ligo.NewModule("orders",
        ligo.Imports(
            microservices.RabbitMQModule(microservices.RabbitMQConfig{
                URL:      "amqp://guest:guest@localhost:5672/",
                Exchange: "ligo",
            }),
        ),
        ligo.Providers(
            // HookedSingleton because nothing else depends on *OrderService —
            // it exists only to attach broker handlers during OnBootstrap.
            ligo.HookedSingleton[*OrderService](NewOrderService),
        ),
    )
}
```

**Planned transports (after RabbitMQ):** Redis, NATS, Kafka, TCP

---

## 📋 Priority 2 — Planned

### Database Module (pgx — raw SQL, full control)
**Package:** `github.com/linkeunid/ligo-db`
**Driver:** [`jackc/pgx/v5`](https://github.com/jackc/pgx) — no ORM, full SQL control

Same layered experience as `@nestjs/typeorm` + TypeORM, but you write raw SQL. No magic. No generated queries. Just pgx with proper DI wiring.

**Three layers (all optional, compose as needed):**

**Layer 1 — Module-level setup (like TypeORM's `TypeOrmModule.forRoot`):**
```go
// Registers *pgxpool.Pool as a global injectable
app.Register(
    database.PostgresModule(database.Config{
        DSN:          "postgres://user:pass@localhost/mydb",
        MaxConns:     25,
        MinConns:     5,
        MaxLifetime:  5 * time.Minute,
    }),
)
```

**Layer 2 — Inject the pool directly into your services (raw pgx):**
```go
// Full pgx power — write any SQL you want
type UserService struct {
    db *pgxpool.Pool
}

func NewUserService(db *pgxpool.Pool) *UserService {
    return &UserService{db: db}
}

func (s *UserService) FindByID(ctx context.Context, id int) (*User, error) {
    row := s.db.QueryRow(ctx,
        "SELECT id, name, email FROM users WHERE id = $1", id)
    var u User
    return &u, row.Scan(&u.ID, &u.Name, &u.Email)
}
```

**Layer 3 — Typed repository pattern (like TypeORM's EntityRepository):**
```go
// ligo wires up the repo via DI — you still write raw SQL
type UserRepository struct {
    db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) FindAll(ctx context.Context) ([]*User, error) {
    rows, _ := r.db.Query(ctx, "SELECT id, name, email FROM users")
    return pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[User])
}

// Register in module — DI injects pool automatically
func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Providers(
            ligo.Factory[*UserRepository](NewUserRepository),
            ligo.Factory[*UserService](NewUserService),
        ),
    )
}
```

**Planned features:**
- `*pgxpool.Pool` auto-registered as provider
- Transaction helper (begin/commit/rollback scoped to request)
- Multi-database support (register multiple pools with named keys)
- `database.ForFeature()` — register repos per module (mirrors `TypeOrmModule.forFeature()`)
- Health check integration (`database.HealthIndicator`)

---

### Task Scheduling
**Package:** `github.com/linkeunid/ligo-schedule`

Cron jobs and interval tasks as lifecycle-managed providers.

```go
// HookedSingleton — no other provider depends on the scheduler, so eager
// resolution is needed to make sure Start fires on OnBootstrap.
ligo.HookedSingleton[*ReportScheduler](NewReportScheduler)

func (s *ReportScheduler) Register(r *ligo.HookRegistry) {
    r.OnBootstrap(s.Start)
    r.OnShutdown(s.Stop)
}

func (s *ReportScheduler) Start() error {
    s.cron.AddFunc("0 9 * * *", s.SendDailyReport)
    s.cron.Start()
    return nil
}
```

---

### WebSocket Support
**Package:** `github.com/linkeunid/ligo-ws`

Hub pattern with room-based broadcasting.

```go
func (c *ChatController) Routes(r ligo.Router) {
    cr := ligo.NewChainRouter(r)
    cr.GET("/ws", c.HandleConnection)
}

func (c *ChatController) HandleConnection(ctx ligo.Context) error {
    conn, _ := c.hub.Upgrade(ctx)
    conn.On("message", c.HandleMessage)
    return nil
}
```

---

## 🔮 Priority 3 — On the Radar

| Feature | Package | Notes |
|---|---|---|
| GraphQL | `ligo-graphql` | Code-first schema, resolver DI, DataLoader, subscriptions |
| OpenAPI / Swagger | `ligo-swagger` | Auto-generate spec from route builder chains |
| Redis cache module | `ligo-cache` | `cache.Module(config)` + injectable `CacheService` |
| Distributed rate limiting | Part of `ligo-cache` | Redis-backed `ThrottleGuard` replacement |
| gRPC | `ligo-grpc` | Server/client with DI-injected stubs |
| MongoDB | `ligo-mongodb` | Raw mongo-driver, no ODM — see below |

### MongoDB Support (Low Priority)
**Package:** `github.com/linkeunid/ligo-mongodb`
**Driver:** [`mongodb/mongo-go-driver`](https://github.com/mongodb/mongo-go-driver) — no ODM, full query control

Same layered experience as `@nestjs/mongoose` + Mongoose, but you write raw mongo-driver queries. No schema magic, no generated methods.

**Three layers (all optional, compose as needed):**

**Layer 1 — Module-level setup (like `MongooseModule.forRoot`):**
```go
// Registers *mongo.Client and *mongo.Database as global injectables
app.Register(
    mongodb.Module(mongodb.Config{
        URI:      "mongodb://localhost:27017",
        Database: "myapp",
    }),
)
```

**Layer 2 — Inject collection per module (like `MongooseModule.forFeature`):**
```go
// Registers *mongo.Collection("users") as injectable in this module
func Module() ligo.Module {
    return ligo.NewModule("user",
        ligo.Imports(
            mongodb.ForFeature("users"),  // injects *mongo.Collection
        ),
        ligo.Providers(
            ligo.Factory[*UserRepository](NewUserRepository),
        ),
    )
}
```

**Layer 3 — Typed repository (like Mongoose Model, but raw queries):**
```go
type UserRepository struct {
    col *mongo.Collection
}

func NewUserRepository(col *mongo.Collection) *UserRepository {
    return &UserRepository{col: col}
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*User, error) {
    var u User
    err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
    return &u, err
}

func (r *UserRepository) Create(ctx context.Context, u *User) error {
    _, err := r.col.InsertOne(ctx, u)
    return err
}
```

**Planned features:**
- `*mongo.Client` and `*mongo.Database` auto-registered as providers
- `mongodb.ForFeature(collection)` — per-module collection injection
- Transaction helper (session-scoped begin/commit/abort)
- Multi-database support (register multiple databases with named keys)
- Health check integration (`mongodb.HealthIndicator`)

---

## Not Planned

| Feature | Reason |
|---|---|
| ORM / Query builder | Intentionally out of scope — use pgx directly |
| Hot Module Replacement | Not applicable in Go |
| REPL | Compiled language — N/A |
| OpenTelemetry | Application-level concern; not framework responsibility |
