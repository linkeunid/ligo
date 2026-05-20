# Timeline — General Webapp Stack

Target use case: build a typical production webapp on Ligo with **Postgres + Auth (incl. OAuth) + Email** out of the box, plus the supporting plumbing a real product needs (config, cache, jobs, observability, storage).

This is the path from today's ecosystem (HTTP core, pgx db, RabbitMQ microservices) to "you can ship a SaaS on Ligo without writing every cross-cutting concern yourself."

> Status legend: ✅ shipped · 🚧 in progress · 📋 planned · 🔮 exploring

---

## Where we are today (2026-05)

| Package | Version | Scope |
|---|---|---|
| `linkeunid/ligo` | `v0.11.0` | HTTP core, DI, modules, guards, pipes, interceptors, lifecycle |
| `linkeunid/ligo-config` | `v0.3.0` | ✅ Layered config — env, `.env`, typed struct binding, validator, eager load |
| `linkeunid/ligo-auth` | `v0.1.0` | ✅ JWT verify-only — HMAC/RSA/ECDSA signers, Guard integration, custom claims |
| `linkeunid/ligo-database` | `v0.1.10` | pgx pool, BaseRepository, RunInTx, health indicator (⚠️ SQL migrations 📋 not yet implemented) |
| `linkeunid/ligo-cache` | `v0.1.0` | ✅ Pluggable store backends, in-memory store, per-entry TTL, HTTP interceptor |
| `linkeunid/ligo-mail` | `v0.1.0` | ✅ SMTP with connection pooling, templates, attachments, TLS, retry with backoff |
| `linkeunid/ligo-microservices` | `v0.2.3` | RabbitMQ RPC + pub/sub |
| `linkeunid/ligo-validator` | `v0.1.9` | Validator integration |
| `linkeunid/ligo-memory` | `v0.1.9` | In-memory cache primitive |
| `linkeunid/ligo-cli` | — | ✅ Project + module scaffolding (`ligo new`, `ligo new-module`) |

Remaining gap to "general webapp": **oauth, sessions/api-key auth, redis cache store, background jobs, storage, observability, health endpoints, migrations runner**.

---

## Minimum viable webapp stack

What a typical Ligo webapp needs to be considered production-ready:

```
┌─────────────────────────────────────────┐
│  ligo (HTTP, DI, guards, pipes)         │  ✅
├─────────────────────────────────────────┤
│  ligo-config       (typed env/file)     │  ✅  v0.3.0
│  ligo-database     (pgx)                │  ✅  v0.1.10
│  ligo-auth         (JWT verify)         │  ✅  v0.1.0  ← sessions/API-key 📋
│  ligo-oauth        (Google/GH/etc.)     │  📋  ← P0
│  ligo-mail         (SMTP)               │  ✅  v0.1.0  ← SES/SendGrid 📋
│  ligo-cache        (in-memory)          │  ✅  v0.1.0  ← Redis store 📋
│  ligo-jobs         (background workers) │  📋
│  ligo-storage      (S3/GCS)             │  🔮
│  ligo-observability (otel + prom)       │  📋
│  ligo-health       (/healthz /readyz)   │  📋
│  ligo-cli          (scaffolding)        │  ✅
└─────────────────────────────────────────┘

SQL migrations ship inside `ligo-database` (not a separate package). **Status: 📋 not yet implemented in `ligo-database` — planned for Phase 2.**
```

---

## Phased timeline

Dates are targets, not commits. Phases gate on the prior phase, not the calendar.

### Phase 0 — Core hardening (now → v1.0)

**Goal:** lock the API surface so satellites can depend on it without churn.

- [ ] Freeze public API of `linkeunid/ligo` (no breaking changes after `v1.0`)
- [ ] Stability doc per package (signal which APIs are stable vs experimental)
- [ ] Reference deployment (k8s manifests, graceful shutdown, pprof, health endpoints)
- [ ] `govulncheck` + race detector in CI on all repos
- [ ] Versioned changelog convention across satellites

**Exit criteria:** `ligo` v1.0 tagged. Satellites can pin `>=1.0,<2.0`.

---

### Phase 1 — Webapp essentials (P0)

The smallest set that unlocks "I can ship a SaaS on Ligo."

#### 1a. `ligo-config` ✅ shipped (`v0.3.0`, 33 tests)

Layered config — env, `.env`, typed struct binding, `go-playground/validator`, eager loading.

```go
cfg := ligo_config.MustLoad(
    ligo_config.WithEnvFiles(".env.local", ".env"),
    ligo_config.WithExpand(true),
)
addr := ":" + cfg.GetOr("PORT", "8080")

app := ligo.New(ligo.WithAddr(addr))
app.Register(ligo_config.ModuleWith(cfg)) // publish into DI
```

**What shipped:** env + `.env` loading, variable expansion, typed `Bind[T]()` into structs, validator integration, `ModuleWith` for pre-DI eager load, namespaced access, `GetOr`/`GetIntOr` helpers.

**Not yet:** JSON/YAML/TOML file sources, Vault/SSM secret backends.

#### 1b. `ligo-auth` ✅ shipped (`v0.1.0`, 38 tests) — JWT verify-only

JWT verify-only — extracts tokens, verifies via HMAC/RSA/ECDSA signers, attaches claims as a `ligo.Guard`.

```go
app.Register(
    ligo_auth.Module(ligo_auth.Config{
        Signer: ligo_auth.NewHMACSigner(ligo_auth.HMACSecret("secret")),
    }),
)

// Protect routes
cr.GET("/profile", c.Profile).
    Guard(authProvider.Guard()).
    Handle()

// Retrieve claims in handler
claims := ligo_auth.ClaimsFromContext[MyClaims](ctx)
```

**What shipped:** JWT verification (HS256/384/512, RS256/384/512, ES256/384/512), pluggable token extractors (Bearer default), custom claims via `ClaimsFactory`, Guard integration, context retrieval with generics.

**Not yet (still 📋):** session (cookie + redis store), API key, basic auth, `RoleGuard` / `ScopeGuard`, password hashing (argon2id/bcrypt), refresh token rotation, CSRF middleware, RBAC primitives, `*UserService` interface.

#### 1c. `ligo-oauth` 📋 — **highest priority unshipped P0 item**

Built on top of `ligo-auth`. Wraps `golang.org/x/oauth2`.

```go
oauth.Module(
    oauth.WithProvider(oauth.Google(cfg.Google.ClientID, cfg.Google.Secret)),
    oauth.WithProvider(oauth.GitHub(cfg.GitHub.ClientID, cfg.GitHub.Secret)),
    oauth.WithCallback("/auth/:provider/callback"),
)
```

- Providers: Google, GitHub, GitLab, Microsoft, Apple, Facebook, Discord
- PKCE on by default
- State + nonce stored in session/redis
- Account-linking hook (`OnLink(user, providerProfile) error`)
- OIDC discovery for custom providers

#### 1d. `ligo-mail` ✅ shipped (`v0.1.0`, 28 tests) — SMTP only

```go
app.Register(
    ligo_mail.Module(ligo_mail.Config{
        Host:        "smtp.example.com",
        Port:        587,
        Username:    "user@example.com",
        Password:    "secret",
        From:        mail.Address{Name: "MyApp", Address: "noreply@example.com"},
        TemplateDir: "./templates",
    }),
)

// Inject *ligo_mail.Service
err := svc.Send(ctx, ligo_mail.Message{
    To:           []mail.Address{{Address: user.Email}},
    Subject:      "Welcome!",
    Template:     "welcome",
    TemplateData: user,
})
```

**What shipped:** SMTP transport with connection pooling, `html/template` + `text/template` rendering, MIME attachments (mixed/alternative/related), inline images, STARTTLS + implicit TLS, retry with backoff, `OnInit`/`OnShutdown` lifecycle hooks, batch sending.

**Not yet (still 📋):** SES, SendGrid, Postmark, Mailgun, Resend transports; MJML opt-in; dev transport (disk capture); queue-aware async send via `ligo-jobs` (Phase 2).

**Phase 1 exit criteria:** a sample app `webapp-full` in `sample/` showing signup/login (password + Google), email verification, password reset, authenticated CRUD on Postgres.

---

### Phase 2 — Operational essentials

Now the app *works*, make it *operable*.

#### 2a. `ligo-cache` ✅ shipped (`v0.1.0`, 28 tests) — in-memory store only

```go
app.Register(ligo_cache.Module(ligo_cache.Config{
    DefaultTTL: 5 * time.Minute,
}))

// Inject *ligo_cache.Cache
val, err := cache.Get(ctx, "user:123")
cache.Set(ctx, "user:123", user, 30*time.Second)
user, err := cache.Wrap(ctx, "user:123", func() (any, error) {
    return db.FindUser(123)
})
```

**What shipped:** pluggable `Store` interface, in-memory store with per-entry TTL + background janitor, `Get`/`Set`/`Del`/`Reset`/`Wrap` (get-or-compute), HTTP response caching interceptor.

**Not yet (still 📋):** Redis store backend (will back sessions, rate limiting); `ligo-ratelimit` integration.

#### 2b. `ligo-jobs` 📋
Background workers. Likely wrap [`hibiken/asynq`](https://github.com/hibiken/asynq) or [`riverqueue/river`](https://github.com/riverqueue/river) (decision pending — river leans pg-native, aligns with our pgx-first stance).

```go
jobs.Module(jobs.Config{Backend: jobs.River{Pool: pool}})

// Producer
s.jobs.Enqueue(ctx, &SendWelcomeEmail{UserID: u.ID})

// Consumer — auto-registered via DI
type SendWelcomeEmailWorker struct { mail *mail.Sender }
func (w *SendWelcomeEmailWorker) Work(ctx context.Context, job *SendWelcomeEmail) error { ... }
```

- Cron via `ligo-schedule` integration
- Retries with exponential backoff
- Dead-letter queue
- Web UI (optional)

#### 2c. `ligo-observability` 📋
OpenTelemetry traces + metrics, Prometheus exporter, structured logs w/ trace ID, request logger middleware.

> Note: the existing roadmap marks OTel as "not planned" at the framework level. This package would be **opt-in**, sitting in the ecosystem, not in core — consistent with that stance.

#### 2d. `ligo-health` 📋
`/healthz` (liveness) and `/readyz` (readiness) with pluggable indicators. `ligo-database`, `ligo-cache`, `ligo-mail` ship indicators.

#### 2e. SQL migrations (inside `ligo-database`) 📋
> **Status:** not yet implemented in `ligo-database` today (`v0.1.10` has no migration runner — samples use raw `migrations/*.sql` files run manually). Planned as a Phase 2 addition to the existing package.

First-class migration runner, shipped **as part of `ligo-database`** — not a separate package. Migrations are tightly coupled to the db pool (same DSN, same lifecycle), so splitting them out adds friction without value.

Likely thin wrapper over [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate) with DI-friendly API:

```go
// embed sql files
//go:embed migrations/*.sql
var migrationsFS embed.FS

app.Register(
    database.PostgresModule(database.Config{DSN: cfg.Database.DSN}),
    database.WithMigrations(database.MigrationConfig{
        FS:       migrationsFS,
        Dir:      "migrations",
        OnBoot:   database.AutoMigrate, // or database.RequireUpToDate
    }),
)
```

- Embedded `fs.FS` source (no runtime file paths)
- `AutoMigrate` on boot (dev) vs `RequireUpToDate` (prod — fail boot if pending)
- CLI: `go run ./cmd/migrate up|down|status|create <name>`
- Advisory-lock guard so multi-replica boots don't race
- Health indicator exposes current schema version

**Phase 2 exit criteria:** webapp sample passes a soak test, exposes Prom metrics, runs background email send, has `/readyz` blocking on db+redis.

---

### Phase 3 — Storage & polish

#### 3a. `ligo-storage` 🔮
S3/GCS/MinIO/local. Signed URL helpers. Streamed up/download.

#### 3b. `ligo-i18n` 🔮
Translation catalogs, locale negotiation middleware.

#### 3c. `ligo-ratelimit` 🔮
Token bucket / sliding window. Redis-backed via `ligo-cache`. Per-IP, per-user, per-route.

#### 3d. `ligo-swagger` 🔮
OpenAPI generation from route builder chains (already on existing roadmap).

---

## Dependency graph

```
ligo-config ✅ ───────► (everyone)
ligo-database ✅ ────► ligo-auth ✅ ──► ligo-oauth 📋
ligo-cache ✅ ───────► ligo-auth (session store 📋)
                  └──► ligo-ratelimit 📋
ligo-mail ✅ ────────► (standalone, async via ligo-jobs 📋)
ligo-jobs 📋 ────────► ligo-mail (async send)
ligo-observability 📋► (everyone, opt-in)
ligo-health 📋 ──────► ligo-database, ligo-cache, ligo-mail
(migrations live inside ligo-database)
```

Build order falls out naturally: ~~`config`~~ → ~~`auth`~~ → `oauth` + ~~`mail`~~ (parallel) → ~~`cache`~~ (Redis store) → `jobs` → rest.

---

## Sample apps to ship alongside

Living in `linkeunid/dev/fw/sample/`:

| Sample | Purpose |
|---|---|
| `the-basic` ✅ | Smallest HTTP service |
| `db-pgx` ✅ | Postgres CRUD |
| `micro-rmq` ✅ | RabbitMQ RPC + pub/sub |
| `hello-ligo` ✅ | CLI-scaffolded starter app (`ligo new`) |
| `webapp-auth` 📋 | Phase 1 milestone — signup/login/oauth/email |
| `webapp-full` 📋 | Phase 2 milestone — adds jobs, cache, observability |

---

## What this doc is not

- Not a commitment to dates. Phases gate on completion, not calendar.
- Not exhaustive. Lower-priority packages (gRPC, GraphQL, WS, Mongo) tracked in [`sneak-peek.md`](./sneak-peek.md) and [`ecosystem.md`](./ecosystem.md).
- Not a replacement for [`1.0-release.md`](./1.0-release.md) — that gates Phase 0.

---

## Open questions

1. **Jobs backend:** `asynq` (redis) vs `river` (postgres). River aligns with our pgx-first stance; asynq is more mature. Decide before Phase 2 kickoff.
2. ~~**Auth user store:** does `ligo-auth` ship a default `users` table schema (via `ligo-database`), or stay storage-agnostic with an interface?~~ **Resolved (v0.1.0):** storage-agnostic — JWT verify-only, no user persistence in `ligo-auth` itself. User store question deferred to session/API-key auth work.
3. **Mail templates:** ship a default template set (welcome, verify, reset) or leave entirely to the user? Leaning ship-with-overrides.
4. **OAuth session model:** stateless JWT-only, or require redis-backed session for OAuth state? `ligo-auth` v0.1.0 is verify-only (no token issuance) — this question applies when we add session support.
5. **Cache Redis store:** `go-redis/redis` vs `rueidis`. Decide when building the Redis `Store` implementation for `ligo-cache`.
