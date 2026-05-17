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
golangci-lint run                       # Lint
go fmt ./...                            # Format
go mod tidy                             # Tidy deps
```

## Conventions

- Root files are thin re-exports from `internal/` packages — never add logic to root-level files
- HTTP abstractions in `internal/http/` are adapter-agnostic — never import Echo directly in `internal/http/`
- Module middleware resolves via DI and applies per module group
- Middleware chaining applies in reverse order (last wraps first)
- Logger auto-registers as provider; injectable without explicit registration
- No hardcoded string keys or `fmt.Printf` in core code — use structured logging
- Interface type support: `Factory[MyInterface](fn)` registers under interface type; container scans for concrete implementor

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