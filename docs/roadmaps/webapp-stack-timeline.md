# Timeline — General Webapp Stack

Target use case: build a typical production webapp on Ligo with **Postgres + Auth (incl. OAuth) + Email** out of the box, plus the supporting plumbing a real product needs (config, cache, jobs, observability, storage).

This is the path from today's ecosystem (HTTP core, pgx db, RabbitMQ microservices) to "you can ship a SaaS on Ligo without writing every cross-cutting concern yourself."

> Status legend: ✅ shipped · 🚧 in progress · 📋 planned · 🔮 exploring

---

## Where we are today (2026-05)

| Package | Version | Scope |
|---|---|---|
| `linkeunid/ligo` | `v0.10.0` | HTTP core, DI, modules, guards, pipes, interceptors, lifecycle |
| `linkeunid/ligo-database` | `v0.1.9` | pgx pool, BaseRepository, RunInTx (⚠️ SQL migrations 📋 not yet implemented) |
| `linkeunid/ligo-microservices` | `v0.2.2` | RabbitMQ RPC + pub/sub |
| `linkeunid/ligo-validator` | (in repo) | Validator integration |
| `linkeunid/ligo-memory` | (in repo) | In-memory cache primitive |

Gap to "general webapp": **auth/oauth, email, config, cache (redis), background jobs, storage, observability, health, migrations runner**.

---

## Minimum viable webapp stack

What a typical Ligo webapp needs to be considered production-ready:

```
┌─────────────────────────────────────────┐
│  ligo (HTTP, DI, guards, pipes)         │  ✅
├─────────────────────────────────────────┤
│  ligo-config       (typed env/file)     │  📋
│  ligo-database     (pgx)                │  ✅
│  ligo-auth         (sessions/JWT)       │  📋  ← P0
│  ligo-oauth        (Google/GH/etc.)     │  📋  ← P0
│  ligo-mail         (SMTP/SES/SendGrid)  │  📋  ← P0
│  ligo-cache        (redis)              │  📋
│  ligo-jobs         (background workers) │  📋
│  ligo-storage      (S3/GCS)             │  🔮
│  ligo-observability (otel + prom)       │  📋
│  ligo-health       (/healthz /readyz)   │  📋
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

#### 1a. `ligo-config` 📋

Typed config loader (env + file + secret backends).

```go
type AppConfig struct {
    HTTP struct {
        Port int    `env:"PORT" default:"8080"`
        Host string `env:"HOST" default:"0.0.0.0"`
    }
    Database struct {
        DSN string `env:"DATABASE_URL" required:"true"`
    }
}

app.Register(config.Module[AppConfig]())
```

- Sources: env, `.env`, JSON/YAML/TOML, Vault/SSM adapter
- Validation via `go-playground/validator`
- Injectable via `*AppConfig`

#### 1b. `ligo-auth` 📋 — **highest priority after config**

Session + JWT + API-key auth, RBAC primitives.

```go
app.Register(
    auth.Module(auth.Config{
        Strategy: auth.JWT,
        Secret:   cfg.JWT.Secret,
        TTL:      24 * time.Hour,
    }),
)

// Guards
cr := ligo.NewChainRouter(r).Use(auth.RequireUser())
cr.GET("/me", c.Me)

// In handler
user := auth.CurrentUser(ctx)
```

Planned:
- Strategies: JWT, session (cookie + redis store), API key, basic
- `AuthGuard` + `RoleGuard` + `ScopeGuard`
- `*UserService` interface — user supplies the persistence (sane defaults via `ligo-database`)
- Password hashing (argon2id default, bcrypt opt-in)
- Refresh token rotation
- CSRF middleware for cookie sessions

#### 1c. `ligo-oauth` 📋

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

#### 1d. `ligo-mail` 📋

```go
mail.Module(mail.Config{
    Transport: mail.SMTP{Host: "...", Port: 587, ...},
    From:      "noreply@example.com",
})

// Inject *mail.Sender
err := s.mail.Send(ctx, mail.Message{
    To:       []string{user.Email},
    Subject:  "Welcome",
    Template: "welcome",
    Data:     map[string]any{"Name": user.Name},
})
```

- Transports: SMTP, SES, SendGrid, Postmark, Mailgun, Resend
- `html/template` + `text/template` rendering, MJML opt-in
- Attachment support
- Dev transport (writes to disk / captures for tests)
- Queue-aware: integrates with `ligo-jobs` for async send (Phase 2)

**Phase 1 exit criteria:** a sample app `webapp-full` in `sample/` showing signup/login (password + Google), email verification, password reset, authenticated CRUD on Postgres.

---

### Phase 2 — Operational essentials

Now the app *works*, make it *operable*.

#### 2a. `ligo-cache` 📋
Redis (default) + in-memory fallback. Backs sessions, rate limiting, generic kv.

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
> **Status:** not yet implemented in `ligo-database` today (`v0.1.9` has no migration runner — samples use raw `migrations/*.sql` files run manually). Planned as a Phase 2 addition to the existing package.

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
ligo-config ──────────► (everyone)
ligo-database ───────► ligo-auth ──► ligo-oauth
ligo-cache ──────────► ligo-auth (session store)
                  └──► ligo-ratelimit
ligo-jobs ───────────► ligo-mail (async send)
ligo-observability ──► (everyone, opt-in)
ligo-health ─────────► ligo-database, ligo-cache, ligo-mail
(migrations live inside ligo-database)
```

Build order falls out naturally: `config` → `auth` → `oauth` + `mail` (parallel) → `cache` → `jobs` → rest.

---

## Sample apps to ship alongside

Living in `linkeunid/dev/fw/sample/`:

| Sample | Purpose |
|---|---|
| `the-basic` ✅ | Smallest HTTP service |
| `db-pgx` ✅ | Postgres CRUD |
| `micro-rmq` ✅ | RabbitMQ RPC + pub/sub |
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
2. **Auth user store:** does `ligo-auth` ship a default `users` table schema (via `ligo-database`), or stay storage-agnostic with an interface? Leaning interface + optional `ligo-auth-pgx` adapter.
3. **Mail templates:** ship a default template set (welcome, verify, reset) or leave entirely to the user? Leaning ship-with-overrides.
4. **OAuth session model:** stateless JWT-only, or require redis-backed session for OAuth state? Stateless-with-signed-state-cookie is workable but more fragile.
