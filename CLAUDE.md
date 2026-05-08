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

## Watch Out

- `ctx.Set(key, val)` / `ctx.Get(key)` uses `any` — use constants for keys, type-assert on retrieval
- Echo adapter's `wrapHandlerWithMiddleware` is shared between Adapter and groupAdapter — changes affect both
- Error types use tree-format messages (`ErrControllerBinding`) — check `Cause()`/`Unwrap()` chain for root cause