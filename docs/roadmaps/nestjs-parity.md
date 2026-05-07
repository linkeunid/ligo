# NestJS Feature Parity

Ligo is inspired by NestJS. This document tracks which NestJS concepts have been adopted and which are planned or out of scope.

## Adopted ✅

### Core Architecture

| NestJS Feature | Ligo Implementation |
|---|---|
| Modules (imports, exports, providers, controllers) | `ligo.NewModule()` with full composition |
| Dependency Injection (singleton) | `ligo.Factory[T](fn)` — default, cached |
| Dependency Injection (transient) | `ligo.Transient[T](fn)` — new instance per resolve |
| Dependency Injection (value) | `ligo.Value[T](instance)` — pre-built singleton |
| Interface-based provider resolution | Container scans for concrete implementors |
| Provider exports | `ligo.Export(provider)` |
| Dynamic modules | `ligo.Dynamic(factory, opts)` |
| Controllers | `ligo.Controller` interface with `Routes(r Router)` |
| HTTP adapter pattern (Express/Fastify equivalent) | Echo v5 adapter; pluggable architecture |

### Request Pipeline

| NestJS Feature | Ligo Implementation |
|---|---|
| Middleware (global, module-level, route-level) | `ligo.Middleware`, `ligo.WithMiddleware()`, `ligo.Middlewares()` |
| Guards | `ligo.Guard` — built-in: `RolesGuard`, `AdminGuard`, `ThrottleGuard` |
| Pipes (validation & transformation) | `ligo.Pipe` — built-in: `ValidationPipe`, `ParseIntPipe`, `ParseBoolPipe`, `UUIDPipe`, `TrimPipe` |
| Interceptors | `ligo.Interceptor` — built-in: `TimeoutInterceptor`, `LoggingInterceptor` |
| Exception filters | `ligo.ExceptionFilter` — route-level error handling |
| Route builder / chain pattern | `ligo.NewChainRouter()` with fluent `.Guard().Pipe().Intercept().Filter().Handle()` |
| Request-scoped context storage | `ctx.Set()` / `ctx.Get()` / `ligo.Get[T](ctx, key)` |

### Lifecycle Hooks

| NestJS Hook | Ligo Implementation |
|---|---|
| `OnModuleInit` | Interface method or `HookedFactory` / `HookedController` |
| `OnApplicationBootstrap` | Interface method or explicit hook registration |
| `BeforeApplicationShutdown` | Interface method or explicit hook registration |
| `OnApplicationShutdown` | Interface method or explicit hook registration |
| `OnModuleDestroy` | Interface method or explicit hook registration |
| Compile-time safe hook registration | `HookedFactory[T]()`, `HookedController()` with `Register(*HookRegistry)` |
| Parallel hook execution | Goroutines + WaitGroup (~50% faster startup/shutdown) |

### HTTP

| NestJS Feature | Ligo Implementation |
|---|---|
| Route groups / prefixes | `router.Group(prefix)` |
| All HTTP methods (GET, POST, PUT, DELETE, PATCH, OPTIONS) | Full support |
| Response helpers | `ctx.OK()`, `ctx.Created()`, `ctx.BadRequest()`, `ctx.NotFound()`, etc. (24 methods) |
| File download streaming | `ctx.Stream(reader)` |
| Graceful shutdown | `WithGracefulShutdown(timeout)` |
| Logger injection | `ligo.Logger` auto-registered as provider |
| Struct validation via tags | `ValidationPipe` with `go-playground/validator` |

### By Design Differences (Not Decorators)

NestJS uses TypeScript decorators. Ligo uses Go-idiomatic equivalents:

| NestJS | Ligo Equivalent |
|---|---|
| `@Module()` decorator | `ligo.NewModule()` function |
| `@Controller()` decorator | `Routes(r Router)` method |
| `@Injectable()` decorator | `ligo.Factory[T](fn)` |
| `@UseGuards()` decorator | `.Guard(...)` on route builder |
| `@UsePipes()` decorator | `.Pipe(...)` on route builder |
| `@UseInterceptors()` decorator | `.Intercept(...)` on route builder |
| `@UseFilters()` decorator | `.Filter(...)` on route builder |
| `@OnModuleInit()` / etc. | Interface methods or `Register(*HookRegistry)` |

---

## Not Yet Adopted ⏳

### Planned as Separate Packages

| NestJS Package | Planned Ligo Package | Priority |
|---|---|---|
| `@nestjs/microservices` | `github.com/linkeunid/ligo/microservices` | High — in progress |
| `@nestjs/schedule` | `github.com/linkeunid/ligo/schedule` | High |
| `@nestjs/websockets` | `github.com/linkeunid/ligo/ws` | Medium |
| `@nestjs/graphql` | `github.com/linkeunid/ligo/graphql` | Medium |
| `@nestjs/swagger` | `github.com/linkeunid/ligo/swagger` | Medium |
| `@nestjs/typeorm` equivalent | `github.com/linkeunid/ligo/database` | Medium (gorm, sqlx, ent) |
| `@nestjs/passport` equivalent | External `ligo-auth-*` packages | Low |
| `@nestjs/cache-manager` | `github.com/linkeunid/ligo/cache` | Low |
| Health checks (Terminus) | External package or contrib | Low |

### Planned for Core

| NestJS Feature | Status | Notes |
|---|---|---|
| Request-scoped providers | Planned | Internal child container infrastructure exists; not yet public API |
| Distributed rate limiting | Planned | Current `ThrottleGuard` is in-memory only; Redis-backed planned |

### Not Applicable (Go Language Differences)

| NestJS Feature | Reason |
|---|---|
| Decorators / Metadata reflection | Go has no decorator syntax; builder pattern used instead |
| Hot Module Replacement (HMR) | Go requires recompilation; no runtime module swapping |
| REPL | Go is compiled; no interactive runtime |
| Schematics (code generation) | Handled by separate `ligo-cli` tool |
| Lazy loading modules | Not idiomatic in Go; out of scope by design |
| Runtime class metadata | TypeScript-specific; replaced by explicit `Register()` methods |

### Not Planned

| NestJS Feature | Reason |
|---|---|
| Server-Sent Events | Can be implemented via standard Go `http.Flusher`; no framework support needed |
| gRPC | Possible future package but not on active roadmap |
| OpenTelemetry / Distributed Tracing | Application-level concern; not a framework responsibility |

---

## Ecosystem Map

```
github.com/linkeunid/ligo          ← Core (v0.6.0) ✅
github.com/linkeunid/ligo-memory   ← In-memory repository helpers ✅

Planned packages:
github.com/linkeunid/ligo/microservices  ← TCP, Redis, RabbitMQ, NATS, Kafka
github.com/linkeunid/ligo/schedule       ← Cron jobs, interval tasks
github.com/linkeunid/ligo/ws             ← WebSocket hub, rooms, broadcasting
github.com/linkeunid/ligo/graphql        ← Code-first schema, resolver DI, DataLoader
github.com/linkeunid/ligo/swagger        ← OpenAPI spec generation
github.com/linkeunid/ligo/database       ← DB integration (gorm, sqlx, ent)
github.com/linkeunid/ligo/cache          ← Redis/memory caching module

External (third-party) packages:
github.com/linkeunid/ligo-auth-jwt       ← JWT auth strategies
github.com/linkeunid/ligo-auth0          ← Auth0 integration
github.com/linkeunid/ligo-cache-redis    ← Redis cache adapter
github.com/linkeunid/ligo-database-pgx  ← PostgreSQL via pgx
```
