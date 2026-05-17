# CLAUDE.md

## Behavioral Foundation

1. Don't assume. Don't hide confusion. Surface tradeoffs.
2. Minimum code that solves the problem. Nothing speculative.
3. Touch only what you must. Clean up only your own mess.
4. Define success criteria. Loop until verified.

## Project

**Ligo** — modular Go framework with lightweight DI, inspired by NestJS.
Module: `github.com/linkeunid/ligo` · Go 1.25.9 · MIT license

## Commands

```bash
go build ./...                          # Build
go test ./...                           # Run tests
go test -v ./...                        # Verbose tests
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out  # Coverage
go test -bench=. -benchmem ./...        # Benchmarks
go test -run TestName ./...             # Single test
go test -race ./...                     # Race detector
golangci-lint run                       # Lint (config: .golangci.yml)
gofumpt -w .                            # Format (stricter than gofmt)
govulncheck ./...                       # Vulnerability scan (stdlib + deps)
staticcheck ./...                       # Standalone staticcheck (also in golangci)
go mod tidy                             # Tidy deps
```

### Static analysis stack

Every ligo* repo ships `.golangci.yml` (schema v2) enabling
`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gofumpt`,
`misspell`, `unconvert`, `unparam`, `revive`, `bodyclose`, `errorlint`,
`nolintlint`, `whitespace`, `tagalign`, `gci`. `govet.enable` adds
`shadow` and `nilness`.

`infertypeargs` flags `Pkg.Generic[T](...)` calls where `T` is
inferable from the arguments. Required reading:
https://pkg.go.dev/golang.org/x/tools/gopls/internal/analysis/infertypeargs.
Source lives inside the `gopls/internal/` package tree, which means it
cannot be wired into `golangci-lint v2` (no public import). It runs in
your editor via `gopls` and we leave it as an IDE-only check. Strip the
explicit type args when the warning lights up — they add visual noise
and drift out of sync with the underlying signature.

Install the toolchain once:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install mvdan.cc/gofumpt@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install golang.org/x/tools/gopls@latest
go install github.com/daixiang0/gci@latest
go install github.com/4meepo/tagalign/cmd/tagalign@latest
```

### Struct tag alignment (tagalign)

`tagalign` (run as part of `golangci-lint`) requires struct tags to line
up across fields. When a struct has more than one field with multiple
tags, the tag columns must align with two-space separators:

```go
// Good — columns line up
type CreateUserInput struct {
    Name  string `json:"name"  validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
}

// Bad — single space, columns drift
type CreateUserInput struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
}
```

Auto-fix:

```bash
tagalign -fix -sort $(find . -name "*.go" -not -path "./vendor/*")
```

`-sort` also enforces a deterministic tag ordering (`json` before
`validate`, etc.). Single-field structs are exempt.

### Import order (gci)

`gci` is enabled in `.golangci.yml` to keep import blocks uniform across
the codebase. Three groups, in this order, separated by a blank line:

1. **Standard library** (`context`, `fmt`, `net/http`, …)
2. **Third-party** (`github.com/labstack/echo/v5`, `github.com/jackc/...`)
3. **Local** — anything under `github.com/linkeunid/`

```go
import (
    "context"
    "fmt"

    "github.com/labstack/echo/v5"

    "github.com/linkeunid/ligo"
    "github.com/linkeunid/ligo-microservices"
)
```

Auto-fix a whole tree:

```bash
gci write --skip-generated \
    -s standard -s default -s "prefix(github.com/linkeunid/)" \
    --custom-order .
```

Outside the linkeunid org, replace the `prefix(...)` value with the
consuming module's path so local imports group correctly.

Pre-release checklist for any change touching exported API:

```bash
gci write --skip-generated -s standard -s default \
    -s "prefix(github.com/linkeunid/)" --custom-order .  # imports order
gofumpt -w .            # auto-fix formatting
go test -race ./...     # tests + race detector
golangci-lint run       # static checks (incl. gci, gofumpt, tagalign)
govulncheck ./...       # CVE scan — must come back clean for stdlib
```

If `govulncheck` flags stdlib CVEs, bump Go (e.g. `mise install go@latest`
then update `go.mod`). Dependency CVEs that don't reach a call path are
informational; ones in actual call paths block release.

### CI

`.github/workflows/ci.yml` runs three jobs on every push to `main` and on
every PR: `golangci-lint` v2 (with the shared `.golangci.yml`), `go test
-race`, and `govulncheck`. Import-order drift, unformatted code,
unwrapped errors, stdlib CVEs, etc. fail the build before merge. Mirror
the same workflow into every ligo* repo so the contract stays uniform.
Action versions are pinned to the Node-24 runtime (checkout v6,
setup-go v6, golangci-lint-action v9) ahead of GitHub's 2026-09-16
Node-20 removal.

## Conventions

- Root files (`app.go`, `provider.go`, `lifecycle.go`, `router.go`, `options.go`, `module.go`) form the **public API surface**. `app.go` keeps the `App` struct and orchestration entry points (`New`, `Run`, `Resolve[T]`); the rest are mostly re-exports / convenience constructors. New algorithmic logic belongs in `internal/` packages — only the public-API surface lives at the root
- HTTP abstractions in `internal/http/` are adapter-agnostic — never import Echo directly in `internal/http/`
- Module middleware resolves via DI and applies per module group
- Middleware chaining applies in reverse order (last wraps first)
- Logger auto-registers as provider; injectable without explicit registration
- No hardcoded string keys or `fmt.Printf` in core code — use structured logging
- Interface type support: `Factory[MyInterface](fn)` registers under interface type; container scans for concrete implementor
- Provider `OnInit` and `OnBootstrap` hooks run **sequentially** in registration order by default (since v0.10.0). Opt into the legacy parallel execution with `ligo.WithParallelHooks()` — only do this for many independent I/O-bound providers where ordering does not matter
- `Resolve[T]` (both `di.Resolve` and the new `ligo.Resolve`) returns `(T, error)`. Use `MustResolve[T]` when a failure should crash the process; never panic from library code that has a non-fatal recovery path

## Working Docs (not committed)

`docs/superpowers/`, `docs/pull-requests/`, `docs/todos/` are gitignored.
They hold session-scoped artifacts (brainstorming specs, PR drafts, audit
reports, scratch plans) that should not land in the public repo. Never
`git add` files under these paths — even when asked to commit "everything."
Write freely, but stage and commit only files outside these directories.

## Dev Cycle

Ligo is consumed as a Go module by sibling repos (`ligo-microservices`,
`ligo-boilerplate`, `ligo-cli`, sample apps). Verify changes against a real
consumer before tagging — `go test` alone won't catch contract breaks.

1. **Make changes** in `ligo/`. Add/update unit tests (`*_test.go` next to
   the file) and an integration test in `integration_test.go` if the change
   adds public API or alters lifecycle behavior.
2. **Build + test locally**:
   ```bash
   go build ./...
   go test -count=1 ./...
   ```
3. **Try it in a real consumer before releasing.** In the consumer's repo:
   ```bash
   go mod edit -replace github.com/linkeunid/ligo=../../ligo
   go build ./... && go run ./cmd/...   # exercise the changed code path
   ```
   The `replace` directive points at the local checkout so you can iterate
   without tagging. Do this for at least one downstream that exercises the
   affected surface (e.g. `sample/micro-rmq` for messaging/hook changes,
   `ligo-boilerplate` for HTTP/DI changes).
4. **Update docs**: every public API change must land in the same patch:
   - `docs/migration.md` — add a section if behavior or API changed
   - `docs/features/<topic>.md`, `docs/best-practices.md`,
     `docs/examples.md` — wherever the topic is covered
   - `README.md` + `docs/roadmaps/1.0-release.md` — version badge and test
     count (`go test -v ./... 2>&1 | grep -cE "^\s*--- PASS"`) and overall
     coverage (`go test -coverpkg=./... -coverprofile=/tmp/cov.out ./...
     && go tool cover -func=/tmp/cov.out | tail -1`)
5. **Commit** with a focused message describing what changed and why.
6. **Release** via `./scripts/release.sh [patch|minor|major]`:
   - `patch` — internal fixes, doc-only changes
   - `minor` — new exported API, additive interface widening, new hooks
   - `major` — only after 1.0; breaking changes
   The script tags, pushes, and creates the GitHub release. Releases are
   semver-tagged on `main` only.
7. **Update consumers**: remove the local `replace` and bump the version:
   ```bash
   go mod edit -dropreplace github.com/linkeunid/ligo
   go get github.com/linkeunid/ligo@vX.Y.Z
   go mod tidy
   ```

If a downstream needs adjustments to match a new API, do those in the
downstream repo and release it after ligo. Never release ligo without a
working consumer build proving the change.

## Watch Out

- `ctx.Set(key, val)` / `ctx.Get(key)` uses `any` — use constants for keys, type-assert on retrieval
- Echo adapter's `wrapHandlerWithMiddleware` is shared between Adapter and groupAdapter — changes affect both
- Error types use tree-format messages (`ErrControllerBinding`) — check `Cause()`/`Unwrap()` chain for root cause