# Ligo Core Audit — 2026-05

**Audit window:** 2026-05-17
**Scope:** every `.go` file under `internal/` and `adapters/echo/`, plus root-level re-export verification (30 files across 16 directories)
**Baseline tooling:** `go vet`, `staticcheck`, `golangci-lint v2` — all clean (see Appendix)
**Method:** line-by-line reading via the `Read` tool, no agent delegation. Findings filed inline as they were encountered, then renumbered for this report.

---

## Summary

- **Total findings:** 41 (BLOCKER: 6, MAJOR: 31, MINOR: 4)
- **Simplification candidates:** 9
- **Test gaps:** 7
- **Recommended pre-v1.0.0 fix set:** all 6 BLOCKERs plus the 27 MAJORs in categories Correctness / Concurrency / Resource safety / Error handling with effort S or M (listed under Phase B below).

The headline finding is **BLOCKER-001 (`duplicate-container-packages`)**: `internal/core/container` is a near-byte-identical copy of `internal/di`, the tests live in the dead copy, production imports only `internal/di`, and that's the package shown at 0.0% coverage. Fixing this single defect both removes ~400 lines of drift-prone duplication and explains the surprise coverage number that motivated the audit.

The other production-blocker is **BLOCKER-005 (`binder-module-mw-prefixes-routes`)** — attaching a middleware to a module silently re-namespaces every route in that module under `/<module-name>/...`. Anyone shipping with module middleware today has URLs that depend on whether a logger middleware was attached at boot.

The TimeoutInterceptor defect (BLOCKER-004) and the never-Put context pool (BLOCKER-003) are the two non-obvious adapter bugs.

---

## Findings

### [BLOCKER-001] duplicate-container-packages
- **File:** `internal/core/container/container.go` (entire file) vs `internal/di/container.go` + `internal/di/errors.go`
- **Category:** Correctness / Simplification
- **Description:** `internal/core/container` is a near-byte-identical copy of `internal/di` (verified via diff — same struct layout, same Resolve generics, same error types, only ordering of helpers differs and a few `di.`/`container.` comment typos). Production code (`app.go`, `adapters/echo/router.go`, `internal/http/*`, `internal/testing/*`) imports only `internal/di`. Nothing imports `internal/core/container`. Meanwhile the 18-test `internal/core/container/container_test.go` covers 68.3% of the dead duplicate — the test suite is exercising code that is not on the production import graph. Bug fixes applied to `internal/di` will be invisible to those tests; fixes applied to `internal/core/container` will not affect production.
- **Repro/evidence:** `grep -rn "internal/core/container" --include='*.go'` returns only the test file in that package. `grep -rn "internal/di" --include='*.go'` returns 13 production files.
- **Suggested fix:** Pick the canonical package (`internal/di` is the prod target). Move the 18 tests over, renaming `container.` → `di.`. Delete `internal/core/container/`. Re-run coverage — `internal/di` should land near 68.3%.
- **Effort:** M

### [BLOCKER-002] mockcontext-missing-imateapot
- **File:** `internal/testing/mocks.go:127-153` (vs `internal/http/context.go:75`)
- **Category:** Correctness (test ergonomics)
- **Description:** `Context` declares `ImATeapot(msg ...string) error`. `MockContext` implements 27 of the 28 response helpers but is missing `ImATeapot`. The moment any test tries to use `MockContext` where a `ligo.Context` is required, the build fails with a method-set error. The package compiles today only because no consumer exercises the interface conformance.
- **Suggested fix:** Add `func (m *MockContext) ImATeapot(msg ...string) error { return nil }`. Add a static interface-assertion in the package (`var _ httpifc.Context = (*MockContext)(nil)`) so future drift fails at build time.
- **Effort:** S

### [BLOCKER-003] echo-contextpool-get-never-put
- **File:** `adapters/echo/router.go:173-190`
- **Category:** Correctness / Performance
- **Description:** `contextPool` is declared and `contextPool.Get()` runs on every request, but **no code ever calls `contextPool.Put`**. Each Get falls through to the pool's `New` factory and allocates a fresh `*contextAdapter`. The pool has zero hit rate; the "reset values map to avoid leaking data between requests" loop at lines 185-187 is dead code (the map is always freshly allocated by `New`).
- **Repro/evidence:** Add a counter inside the `New` factory — every request bumps it.
- **Suggested fix:** Either (a) call `contextPool.Put(ca)` after the wrapped handler returns in `wrapHandlerWithMiddleware` and properly clear `ca.c`, `ca.reqCont`, and the values map; or (b) delete the pool, document that contexts are per-request fresh, and let `sync.Pool` come back when there is a measured win.
- **Effort:** M

### [BLOCKER-004] timeoutinterceptor-ctx-and-leak
- **File:** `internal/http/interceptors/interceptors.go:22-40`
- **Category:** Concurrency / Correctness
- **Description:** Three intertwined defects in `TimeoutInterceptor`:
  1. **Wrong context parent.** `context.WithTimeout(context.Background(), timeout)` derives from `Background`, not the request context. Client disconnect, parent middleware timeouts, and graceful shutdown signals are ignored — the timer fires regardless.
  2. **Goroutine leak past timeout.** When `timeoutCtx.Done()` wins the select, the goroutine running `next(ctx)` continues. The buffered `done` channel keeps the goroutine from blocking on send, but the goroutine outlives the request, holding all captured state (db handles, allocated buffers) until `next` returns naturally — which may be never.
  3. **Response writer race.** After the interceptor returns the timeout error, the framework writes the timeout response onto `ctx.Response()`. The leaked goroutine may also call `next(ctx)` which writes to the same `ResponseWriter`. `http.ResponseWriter` is not safe for concurrent use — result is undefined: superfluous WriteHeader logs, garbled bodies, or a runtime-guard crash.
- **Repro/evidence:** Wire `TimeoutInterceptor(50*time.Millisecond)` to a handler that sleeps 1s and writes. Observe goroutine count via `runtime.NumGoroutine` before/after, and inspect the response under `go test -race`.
- **Suggested fix:** Derive `timeoutCtx` from a request-context accessor on `Context` (add `RequestContext() context.Context` to the interceptors.Context interface). Pass `timeoutCtx` into a wrapped context handed to `next`, and document that the goroutine is best-effort. Pair the timeout with a cancellation signal the handler can read so it can exit promptly. Until the handler responds to cancellation, the leak is bounded but real — say so in the godoc.
- **Effort:** M

### [BLOCKER-005] binder-module-mw-prefixes-routes
- **File:** `internal/http/binder.go:66-73`
- **Category:** Correctness / API surprise
- **Description:** When a module has middleware (`len(modMw) > 0`), `bindModuleControllers` creates a Group under `"/" + mod.Name` and registers all controllers under that prefix. When the module has no middleware, controllers register at root. Same module gets different URL paths depending on whether middleware is attached — adding a single middleware silently shifts every route under `/<module-name>/...`. No way to attach module middleware without re-namespacing routes.
- **Repro/evidence:** `module.New("auth", Middlewares(LogMw), Controllers(UserCtrl))` — routes that worked at `/users` now serve at `/auth/users`.
- **Suggested fix:** Apply module middleware to the existing router/group without changing the path prefix. If a prefix is wanted, take an explicit `Prefix("/auth")` option rather than deriving it from the module name.
- **Effort:** S

### [BLOCKER-006] duplicate-validateexhaustive
- **File:** `internal/validation/validator.go:48-104` vs `internal/http/pipes/pipes.go:141-220`
- **Category:** Correctness (drift risk) / Simplification
- **Description:** `validateExhaustive` is implemented twice with identical logic and identical doc-strings. Both carry the "replace empty with `x`" UX flaw (MAJOR-020). A fix applied to one will not flow to the other; new validation tags added to one path will silently behave differently in the other. Same pattern as BLOCKER-001.
- **Suggested fix:** Have `internal/http/pipes` import `internal/validation` and call its `validateExhaustive`. Pipe package keeps only the binding/context glue.
- **Effort:** S

---

### [MAJOR-001] lifecycle-append-not-thread-safe
- **File:** `internal/core/lifecycle/lifecycle.go:27-39`
- **Category:** Concurrency
- **Description:** `AddServer`, `AppendStartHook`, `AppendStopHook` mutate `AppLifecycle` fields without holding `l.mu`. Only `IsStarted`/`Start` use the mutex. If any provider's `OnInit` adds a stop hook (common pattern — register cleanup at startup), that write races with whatever else is appending concurrently.
- **Suggested fix:** Take `l.mu.Lock()` in all three methods. Once started, reject further appends (return error or panic with a clear message) instead of silently swallowing them.
- **Effort:** S

### [MAJOR-002] lifecycle-stop-not-thread-safe
- **File:** `internal/core/lifecycle/lifecycle.go:60-75`
- **Category:** Concurrency
- **Description:** `Stop` reads `l.server` and `l.onStop` without holding `l.mu`, and does not guard against concurrent invocation. A second Stop while the first is running can double-call `srv.Shutdown` (returns `http.ErrServerClosed`, masking the real first-call error) and re-iterate hooks.
- **Suggested fix:** Add a `stopped` flag protected by `l.mu`. Take the lock, snapshot `onStop`/`server`, set `stopped=true`, unlock, then iterate. Subsequent Stop calls return early.
- **Effort:** S

### [MAJOR-003] lifecycle-start-no-rollback
- **File:** `internal/core/lifecycle/lifecycle.go:51-56`
- **Category:** Resource safety
- **Description:** If a startup hook fails mid-way, hooks that already succeeded keep their side effects (open db pools, RMQ connections, started goroutines). Caller sees an error but the process is now in a half-initialized state. The typical reaction (`log.Fatal`) leaks goroutines and connections to the OS until process exit; in test contexts this leaks across tests.
- **Suggested fix:** On hook failure, run stop hooks `0..i-1` in reverse before returning. Optionally honor a `WithoutStartRollback` option.
- **Effort:** M

### [MAJOR-004] logger-options-overwrite-each-other
- **File:** `internal/core/logger/logger.go:56-79`
- **Category:** Correctness (API trap)
- **Description:** Each `WithJSON`/`WithText`/`WithDebug` option replaces the handler from scratch, losing settings from previously applied options. `New(WithJSON(), WithDebug(true))` produces **text** at debug level (WithDebug builds a fresh text handler). `New(WithDebug(true), WithJSON())` produces JSON at info level (WithJSON builds a fresh JSON handler with no level). Order silently determines behavior.
- **Suggested fix:** Accumulate options into a single config struct (`{format, level}`), then build the handler once at the end of `New`. Each option mutates the config, not the handler.
- **Effort:** S

### [MAJOR-005] logger-setdebug-race
- **File:** `internal/core/logger/logger.go:127-140`
- **Category:** Concurrency
- **Description:** `SetDebug` reassigns `l.handler` and `l.logger` without synchronization. Concurrent `Debug`/`Info` calls read `l.logger` while SetDebug writes it. Data race per the Go memory model.
- **Suggested fix:** `atomic.Pointer[slog.Logger]` or `RWMutex`; alternative is `slog`'s `LevelVar` so level updates without recreating the handler.
- **Effort:** S

### [MAJOR-006] module-two-hook-paths
- **File:** `internal/core/module/module.go:12-22`
- **Category:** API ergonomics
- **Description:** `Module` exposes both `OnInit []func() error` / `OnDestroy []func() error` AND `Hooks *lifecycle.ModuleHookRegistry`. Two parallel hook registration channels; downstream coordinators must check both, and users discover the duplication only when one path silently no-ops.
- **Suggested fix:** Collapse onto `ModuleHookRegistry`. Have `OnModuleInit`/`OnModuleDestroy` ModuleOption functions append to `m.Hooks` (allocating it on first use). Mark the raw slices deprecated pre-v1.0.
- **Effort:** M

### [MAJOR-007] app-build-provider-reflective-fn-lookup
- **File:** `internal/app/app.go:71`
- **Category:** Correctness
- **Description:** `reflect.ValueOf(p).MethodByName("Fn").Call(...)` reaches into the provider by string method name. The `Provider` interface (lines 37-44) defines `Type/Eager/IsTransient/IsExported/IsEagerResolve/Hooks` but **not** `Fn`. Any implementation lacking `Fn` panics at registration time, with no static guarantee.
- **Suggested fix:** Add `Fn() any` (or a typed factory shape) to the `Provider` interface so the compiler enforces it.
- **Effort:** S

### [MAJOR-008] app-buildmodule-discarded-type-assertion
- **File:** `internal/app/app.go:112`
- **Category:** Correctness
- **Description:** `provider, _ := p.(Provider)` discards the `ok` bool. When `p` is not a `Provider`, `provider` is nil and `BuildProviderEntry(provider)` panics on `p.Hooks()`.
- **Suggested fix:** `provider, ok := p.(Provider); if !ok { panic with a message naming the offending type }`.
- **Effort:** S

### [MAJOR-009] app-buildmodule-dead-modcontainer-branch
- **File:** `internal/app/app.go:100-119`
- **Category:** Correctness (dead-code / hidden bug)
- **Description:** `modContainer := parent` and then both branches register on the same container — the exported/non-exported distinction is meaningless. Either `modContainer := parent.NewChild()` was lost in a refactor (bug) or the branch is dead.
- **Suggested fix:** Decide intent. If module-private providers were the goal, restore `parent.NewChild()` and wire the child into controller/middleware resolution. Otherwise collapse to one branch and drop `IsExported` from this code path.
- **Effort:** M

### [MAJOR-010] server-app-shutdown-errors-swallowed
- **File:** `internal/app/server.go:120-134`
- **Category:** Error handling
- **Description:** `opts.AppShutdown()` and each `OnStop` hook error is logged then dropped. `serveWithGracefulShutdownAt` returns nil even when application shutdown failed. Callers cannot detect partial shutdown.
- **Suggested fix:** Accumulate shutdown errors (`errors.Join`) and return them. Keep logging; surface the joined error so the binary can `os.Exit(1)` on failure.
- **Effort:** S

### [MAJOR-011] server-erraddrinuse-contract
- **File:** `internal/app/server.go:21, 38-40, 80-87`
- **Category:** Correctness / API contract
- **Description:** `IsAddrInUse` matches only the sentinel `ErrAddrInUse`. OS-level binding errors are `syscall.EADDRINUSE` wrapped in `*net.OpError`. If a future adapter (or the current Echo adapter under some Go versions) passes the raw error through, `AutoPort` silently no-ops on real port collisions.
- **Suggested fix:** Either document the Router contract (`Serve must return ErrAddrInUse for collisions`) and validate it in tests, or change `IsAddrInUse` to also recognize `errors.Is(err, syscall.EADDRINUSE)`.
- **Effort:** S

### [MAJOR-012] http-builder-guard-denied-generic-error
- **File:** `internal/http/builder.go:118-120`
- **Category:** Error handling / API ergonomics
- **Description:** When a guard returns `(false, nil)`, the wrapper returns `fmt.Errorf("guard denied access")`. No HTTP status hint, no guard identification. Mapping this to 403 in an ExceptionFilter requires string-matching the message.
- **Suggested fix:** Introduce `ErrGuardDenied` sentinel (or typed error with guard name + recommended status). ExceptionFilter maps it to 403.
- **Effort:** S

### [MAJOR-013] http-builder-silent-nil-handler
- **File:** `internal/http/builder.go:82-84`
- **Category:** Correctness
- **Description:** `routeBuilder.Handle()` silently returns when handler is nil — no route registered, no log, no error. `.GET("/x").Guard(...)` without `.Handle(fn)` produces a 404 at runtime instead of a registration-time error.
- **Suggested fix:** Log a warning and/or panic at build time. Registration is startup-only; failing loud is correct.
- **Effort:** S

### [MAJOR-014] http-context-interface-bloat
- **File:** `internal/http/context.go:21-83`
- **Category:** API ergonomics / Simplification
- **Description:** `Context` defines ~40 methods, including 23 HTTP status helpers (`OK/Created/.../HTTPVersionNotSupported`). Every adapter must implement all 23 plus `Stream(reader any)`. Blocks the "second adapter validates the abstraction" recommendation and obscures the minimal surface.
- **Suggested fix:** Keep a small adapter contract on `Context` (Request/Response/Param/Query/Bind/Set/Get/JSON/String/NoContent/Stream). Move the 22 status helpers to package-level functions taking `Context` (`ligo.BadRequest(ctx, msg)`), or a thin helper struct.
- **Effort:** L
- **Status:** Done in v0.11.0 — `Context` is now a concrete struct embedding the new 14-method `Adapter` interface; 28 response helpers + 3 query helpers live as methods on `*Context`. Handlers receive `*ligo.Context`; `ctx.OK(...)` etc. unchanged.

### [MAJOR-015] http-context-stream-any
- **File:** `internal/http/context.go:82`
- **Category:** API ergonomics / Correctness
- **Description:** `Stream(reader any) error` accepts `any` instead of `io.Reader`. Each adapter must type-assert; wrong types panic at runtime instead of compile time.
- **Suggested fix:** `Stream(reader io.Reader) error`. If close is needed, accept `io.ReadCloser`.
- **Effort:** S

### [MAJOR-016] binder-resolveconstructor-validator-misnamed
- **File:** `internal/http/binder.go:107-142`
- **Category:** API ergonomics
- **Description:** `resolveConstructor` is shared between Middleware and Controller resolution but the validator parameter is statically typed `func(reflect.Value) (Middleware, error)`. The Controller path passes a validator returning `(nil, nil)` — the Middleware return type is meaningless for controllers. Suggests it was extracted mid-refactor.
- **Suggested fix:** Generic `resolveConstructor[T any]` returning `(T, error)`, or untyped (`any`) with the caller responsible for asserting.
- **Effort:** S

### [MAJOR-017] throttleguard-global-state
- **File:** `internal/http/guards/guards.go:95-99`
- **Category:** Correctness / Test ergonomics
- **Description:** `throttleStore`, `throttleMu`, `cleanupStarted` are package-level globals. Two apps in the same process share one rate-limit window. Tests cannot reset the store. The cleanup goroutine is started once and lives forever — no way to stop on shutdown.
- **Suggested fix:** Wrap state in a `Throttler` struct returned by `NewThrottler(maxRequests, window)`. Constructor returns both the guard and a `Close()`. Have ligo register `Close` on app shutdown.
- **Effort:** M

### [MAJOR-018] throttleguard-evict-not-oldest
- **File:** `internal/http/guards/guards.go:148-170`
- **Category:** Correctness / Doc accuracy
- **Description:** `evictOldestEntries` iterates `throttleStore` via `for key := range` — Go map iteration is randomized. The function name and doc claim "oldest"; in practice it evicts random entries. Under load, legitimate clients can be evicted while abusers persist.
- **Suggested fix:** Rename to `evictArbitraryEntries` and update the godoc, or track per-entry "last seen" and evict by that.
- **Effort:** S

### [MAJOR-019] throttleguard-coarse-mutex
- **File:** `internal/http/guards/guards.go:72-92`
- **Category:** Concurrency (performance)
- **Description:** `throttleMu` is held for the entire critical section, including `entry.mu.Lock()` and `filterOldCounts`. Holding the global lock through per-entry work serializes every throttle check across all clients; the per-entry mutex is redundant.
- **Suggested fix:** Take the global lock only long enough to look up/insert the entry; release before locking per-entry. Or drop the per-entry mutex entirely.
- **Effort:** S

### [MAJOR-020] validationpipe-exhaustive-passes-replace-with-x
- **File:** `internal/http/pipes/pipes.go:141-220` (and same logic in BLOCKER-006)
- **Category:** Correctness / UX
- **Description:** Two-pass `validateExhaustive` substitutes `"x"` for empty strings to bypass `required`, then re-runs validation. For fields with format validators (`email`, `oneof=a|b`, `uuid`, regex), `"x"` itself fails — second pass surfaces "must be a valid email" alongside the "required" error. Users see contradictory messages for the same empty input.
- **Suggested fix:** Skip second-pass errors for format-sensitive tags when the original error was `required`. Or pick a tag-aware sentinel.
- **Effort:** M

### [MAJOR-021] errors-formatchain-unbounded-recursion
- **File:** `internal/errors/chain.go:51-68`
- **Category:** Correctness
- **Description:** `FormatChain` recurses on `cause` indefinitely. A cyclic chain (legal under `errors` — `A.Unwrap() = B, B.Unwrap() = A`) produces stack overflow. Pathological deep chains DoS the formatter.
- **Suggested fix:** Track visited errors or cap depth at e.g. 32; emit `<truncated>` on hit.
- **Effort:** S

### [MAJOR-022] echo-setcontainer-twice-double-mw
- **File:** `adapters/echo/router.go:46-53`
- **Category:** Correctness
- **Description:** `SetContainer` prepends `requestScopeMiddleware` on every call. Two calls insert it twice, creating a child-of-child container per request.
- **Suggested fix:** Track whether request-scope middleware is installed; replace idempotently or panic on double-set.
- **Effort:** S

### [MAJOR-023] echo-groupadapter-serve-noop
- **File:** `adapters/echo/router.go:155-157`
- **Category:** Correctness
- **Description:** `groupAdapter.Serve(addr)` returns nil unconditionally. Calling `Serve` on a group silently succeeds without starting a server.
- **Suggested fix:** Return `errors.New("ligo: cannot Serve on a route group; call Serve on the root router")`. Or panic at registration time.
- **Effort:** S

### [MAJOR-024] echo-stream-bad-classification
- **File:** `adapters/echo/router.go:372-380`
- **Category:** Correctness
- **Description:** `Stream(reader any)` type-asserts to `io.ReadCloser`. A plain `io.Reader` (common case) produces `400 Bad Request: invalid reader` — server-side type confusion misclassified as client error.
- **Suggested fix:** After MAJOR-015 lands (`Stream(io.Reader)`), drop the assertion. If close is needed, accept `io.ReadCloser`.
- **Effort:** S

### [MAJOR-025] mockcontext-no-op-bind
- **File:** `internal/testing/mocks.go:91-93, 75-77`
- **Category:** Test ergonomics
- **Description:** `MockContext.Bind` and `BindQuery` return nil and leave targets zero-valued. Tests exercising handlers that depend on bind cannot verify either success or failure. Mock is unusable for any handler that uses Bind.
- **Suggested fix:** `MockContext.SetBody(any)` stashes a body; `WithBindError(error)` injects failure. `Bind` then copies / returns accordingly via reflect.
- **Effort:** M

### [MAJOR-026] app-parallel-hook-errors-aggregated-to-count
- **File:** `app.go:294-323` (`executeHooksParallel`)
- **Category:** Error handling
- **Description:** Parallel hook executor collects individual errors but returns only `"hook execution failed: %d errors occurred"`. Messages, stack info, provider identity all discarded.
- **Suggested fix:** `errors.Join(results...)`. Log each error with provider name as it arrives.
- **Effort:** S

### [MAJOR-027] app-shutdown-errors-swallowed
- **File:** `app.go:362-392` (`(*App).shutdown`)
- **Category:** Error handling
- **Description:** Shutdown logs every hook failure but the function always returns nil. Combined with MAJOR-010, the shutdown path is opaque to operators — exit code is always 0 even when destroy hooks fail.
- **Suggested fix:** `errors.Join` and return. Keep logging for liveness; surface the joined error so the binary can exit non-zero.
- **Effort:** S

### [MAJOR-028] di-build-returns-nil-nil
- **File:** `internal/di/container.go:302`
- **Category:** Correctness
- **Description:** `build` returns `(nil, nil)` when a registered entry has neither `eager` nor `factory`. Callers receive nil with no error; `cache.Store(typ, nil)` poisons cache; subsequent `Resolve[T]` panics for concrete T.
- **Suggested fix:** Return `&DIError{Type: typ.String(), Cause: errors.New("entry has neither eager nor factory")}`.
- **Effort:** S

### [MAJOR-029] di-dierror-missing-unwrap
- **File:** `internal/di/errors.go:49-57`
- **Category:** Error handling
- **Description:** `DIError` carries `Cause` but does not implement `Unwrap() error`. Same for `ErrCircularDependency`, `ErrAmbiguousDependency`, `ErrDuplicateProvider`. `errors.Is`/`errors.As` cannot traverse the chain.
- **Suggested fix:** Add `Unwrap() error` to every error type with a `Cause` field.
- **Effort:** S

### [MAJOR-030] di-errmissing-empty-requiredby
- **File:** `internal/di/errors.go:14-16`
- **Category:** Error handling
- **Description:** `ErrMissingDependency.Error()` always prints `"(required by %s)"` even when `RequiredBy` is empty, producing `"ligo: missing dependency T (required by )"`.
- **Suggested fix:** Conditional formatting — omit the clause when `RequiredBy == ""`.
- **Effort:** S

### [MAJOR-031] di-factory-error-loses-requiredby
- **File:** `internal/di/container.go:285-290`
- **Category:** Error handling
- **Description:** When a factory returns an error, the wrapping `DIError` is built with `RequiredBy: ""`, dropping the caller context the resolver already has.
- **Suggested fix:** Pass the caller type into `build` and set `RequiredBy` to the last element of `chain` when non-empty.
- **Effort:** S

---

### [MINOR-001] root-files-not-thin-re-exports
- **File:** `app.go` (420 lines, 23 decls), `provider.go` (220, 16 decls), `lifecycle.go` (264, 16 decls), `router.go` (190, 26 decls), `options.go` (182, 26 decls), `module.go` (199, 13 decls)
- **Category:** Documentation / convention drift
- **Description:** CLAUDE.md says "Root files are thin re-exports from `internal/` packages — never add logic to root-level files." Reality: `app.go` orchestrates startup/shutdown with 420 lines (sequential + parallel hook executors, eager-resolve, shutdown reverse loop). `provider.go` defines the public `Provider` struct and constructors. The convention does not match the codebase and is therefore unenforced.
- **Suggested fix:** Either update CLAUDE.md to reflect reality, or move `executeHooksParallel` / `shutdown` into `internal/app/` so app.go shrinks to a thin entry point.
- **Effort:** S (doc) or M (move logic)

### [MINOR-002] lifecycle-modulehookregistry-method-order
- **File:** `internal/core/lifecycle/registry.go:91-104`
- **Category:** Simplification (readability)
- **Description:** `GetInitHooks`/`GetDestroyHooks` are defined on `*ModuleHookRegistry` before the struct itself is declared. Compiles but obscures the type's surface area.
- **Suggested fix:** Move getters below the struct + constructor.
- **Effort:** S

### [MINOR-003] lifecycle-registry-inconsistent-semantics
- **File:** `internal/core/lifecycle/registry.go` (`HookRegistry` vs `ModuleHookRegistry`)
- **Category:** API ergonomics
- **Description:** `HookRegistry.OnInit` overwrites (single-set); `ModuleHookRegistry.OnInit` appends. Same method name, different semantics in the same package; doc does not call this out.
- **Suggested fix:** Rename one (`HookRegistry.SetInit` for the overwrite variant) or document the divergence.
- **Effort:** S

### [MINOR-004] lifecycle-hooks-refresh-value-receiver
- **File:** `internal/core/lifecycle/hooks.go:49-58`
- **Category:** API ergonomics
- **Description:** `Hooks.Refresh()` has a value receiver and returns the mutated copy. `h.Refresh()` (discarding return) silently no-ops.
- **Suggested fix:** Pointer receiver that mutates in place, or rename to `WithRefreshed`.
- **Effort:** S

---

## Simplification candidates

### [SIMP-001] logger-extractprovidername-deprecated
- **File:** `internal/core/logger/logger.go:150-154`
- **Category:** Dead-ish API
- **Description:** `ExtractProviderName` is marked deprecated but still used in `internal/di/container.go:83-86`. The replacement (`reflectutil.ExtractTypeName`) is right there.
- **Suggested fix:** Replace call sites with `reflectutil.ExtractTypeName`, delete the deprecated shim.
- **Effort:** S

### [SIMP-002] module-dynamicmodule-untyped-options
- **File:** `internal/core/module/module.go:5-9, 99-109`
- **Category:** API ergonomics
- **Description:** `DynamicModule.Factory func(...any) Module` accepts `...any` for configuration. Type safety zero; every consumer does unchecked casts.
- **Suggested fix:** Inventory callers and decide: keep as documented escape hatch, or replace with typed per-module constructors.
- **Effort:** M

### [SIMP-003] echo-addrinuse-string-match
- **File:** `adapters/echo/router.go:94-99`
- **Category:** Robustness
- **Description:** Adapter detects "address already in use" via `opErr.Op == "listen"` OR substring match. Locale/OS-dependent.
- **Suggested fix:** `errors.Is(err, syscall.EADDRINUSE)` (works through `*net.OpError.Unwrap()` since Go 1.20).
- **Effort:** S

### [SIMP-004] chainrouter-manual-forwarding
- **File:** `internal/http/chain.go:15-38`
- **Category:** Simplification
- **Description:** `chainRouter` manually forwards each `Router` method. New methods on `Router` silently break implementation.
- **Suggested fix:** `type chainRouter struct { Router }`. Drop the four forwarding methods.
- **Effort:** S

### [SIMP-005] pool-contextpool-redundant
- **File:** `internal/pool/context.go:50-71`
- **Category:** Simplification
- **Description:** `ContextPool[T]` is a thin wrapper around `Pool[T]` with no extra behavior.
- **Suggested fix:** Delete `ContextPool`; consumers use `Pool[T]` directly.
- **Effort:** S

### [SIMP-006] reflectutil-extracttypename-duplication
- **File:** `internal/reflect/types.go:10-64`
- **Category:** Simplification
- **Description:** `ExtractTypeName` and `ExtractTypeNameFromFunc` cover overlapping cases — the func-branch of the former duplicates the latter end-to-end.
- **Suggested fix:** Keep `ExtractTypeName`. Delete `ExtractTypeNameFromFunc` after a call-site sweep.
- **Effort:** S

### [SIMP-007] di-errduplicateprovider-unused
- **File:** `internal/di/errors.go:30-36`
- **Category:** Dead code
- **Description:** `ErrDuplicateProvider` is exported but never returned anywhere. `Container.Register` silently warns and ignores duplicates.
- **Suggested fix:** Delete pre-v1.0, or introduce `RegisterStrict` that actually returns it.
- **Effort:** S

### [SIMP-008] di-error-types-duplicate-fields
- **File:** `internal/di/errors.go`
- **Category:** Simplification
- **Description:** `DIError` and `ErrMissingDependency` carry identical fields (Type, RequiredBy, Cause).
- **Suggested fix:** Collapse to a single struct with a `Reason` enum, or factor the common formatting helper.
- **Effort:** M

### [SIMP-009] di-logger-nil-checks
- **File:** `internal/di/container.go:75, 82, 170, 249`
- **Category:** Simplification
- **Description:** Every log call site repeats `if c.logger != nil { ... }`.
- **Suggested fix:** Introduce `logger.Noop()` and assign it in `New` when no logger is supplied.
- **Effort:** S

---

## Test gaps

### [TEST-001] di-zero-coverage
- **File:** `internal/di/` (entire package)
- **Description:** 0.0% direct coverage. The 18 tests in `internal/core/container/` cover the dead duplicate, not the production code (see BLOCKER-001). Add `internal/di/container_test.go` for interface fallback ordering, ambiguous-dependency, circular detection across child↔parent, factory panics, transient vs singleton.
- **Effort:** M

### [TEST-002] lifecycle-coverage
- **File:** `internal/core/lifecycle/` (35.4%)
- **Description:** `AppLifecycle.Stop`, `Hooks.Refresh`/`HasRegistry`, most of `ModuleHookRegistry` untested. Add tests for concurrent Append/Stop, start-rollback, HookRegistry single-set vs ModuleHookRegistry append semantics.
- **Effort:** M

### [TEST-003] http-internals-low-coverage
- **File:** `internal/http/` (18.3%)
- **Description:** builder.go (entire chain), binder.go ErrControllerBinding tree-format, chain.go ChainRouter wrapper, pagination.go edge cases.
- **Effort:** L

### [TEST-004] http-subpackages-zero-coverage
- **File:** `internal/http/{guards,interceptors,pipes}/` (all 0.0%)
- **Description:** TimeoutInterceptor (success / timeout / handler-error / cancellation), ThrottleGuard (concurrent clients, eviction, cleanup), ValidationPipe (required+email confusion, nested struct, ptr vs value).
- **Effort:** L

### [TEST-005] leaf-packages-zero-coverage
- **File:** `internal/pool/`, `internal/reflect/`, `internal/validation/`, `internal/errors/` (all 0.0%)
- **Description:** `FormatChain` recursion bug (MAJOR-021), `validateExhaustive` two-pass logic, Pool Get/Put — all easy targets.
- **Effort:** M

### [TEST-006] echo-adapter-low-coverage
- **File:** `adapters/echo/router.go` (17.2%)
- **Description:** Error-response helpers (22 of them) untested. `Shutdown`, `Stream`, `requestScopeMiddleware`, addrInUse branch uncovered.
- **Effort:** M

### [TEST-007] testing-zero-coverage
- **File:** `internal/testing/` (0.0%)
- **Description:** Test helpers themselves have zero tests. Combined with BLOCKER-002 and MAJOR-025, the package is untrustworthy.
- **Effort:** S

---

## Recommended sequencing

### Phase A — pre-v1.0.0 blockers
Must land before any v1.0.0 tag:

- BLOCKER-001 (duplicate-container-packages) — collapse `internal/core/container` into `internal/di`, port tests
- BLOCKER-002 (mockcontext-missing-imateapot) — add method + static interface assertion
- BLOCKER-003 (echo-contextpool-get-never-put) — either Put properly or remove the pool
- BLOCKER-004 (timeoutinterceptor-ctx-and-leak) — request-context derivation, document leak window
- BLOCKER-005 (binder-module-mw-prefixes-routes) — stop using module name as URL prefix
- BLOCKER-006 (duplicate-validateexhaustive) — collapse to one implementation

### Phase B — pre-v1.0.0 majors worth shipping
Correctness / Concurrency / Resource-safety / Error-handling issues that block a defensible v1.0 release:

- MAJOR-001..005 (lifecycle thread-safety, rollback; logger option clobbering, race)
- MAJOR-007..009 (app reflective Fn lookup, discarded type assertion, dead modContainer branch)
- MAJOR-010..013 (shutdown error propagation, addrinuse contract, guard denial sentinel, silent nil handler)
- MAJOR-015 (Stream io.Reader)
- MAJOR-016 (resolveConstructor generics)
- MAJOR-017..020 (throttle globals/eviction/lock; validation `"x"` substitution)
- MAJOR-021 (FormatChain recursion cap)
- MAJOR-022..024 (echo SetContainer idempotency, group Serve, Stream classification)
- MAJOR-026..031 (parallel-hook error join; app shutdown error join; DI nil/nil, Unwrap, empty RequiredBy, factory error context)

### Phase C — post-v1.0.0 backlog
Improvements that don't block the release but should be on the v1.x roadmap:

- MAJOR-006 (module two hook paths — API rework)
- MAJOR-014 (Context interface bloat — coordinate with second-adapter work)
- MAJOR-025 (MockContext usability)
- MINOR-001..004 (convention/doc drift, registry semantics, refresh value-receiver)
- SIMP-001..009
- TEST-001..007 (parallel track — start during Phase B to prevent regressions on the fixes that land there)

---

## Files reviewed (30)

**Findings counts in parens.**

`adapters/echo/router.go` (5)
`internal/app/app.go` (3)
`internal/app/server.go` (2)
`internal/core/container/container.go` (1 — duplicate)
`internal/core/lifecycle/hooks.go` (1)
`internal/core/lifecycle/lifecycle.go` (3)
`internal/core/lifecycle/registry.go` (2)
`internal/core/logger/logger.go` (3)
`internal/core/module/module.go` (2)
`internal/di/container.go` (4)
`internal/di/errors.go` (4)
`internal/errors/chain.go` (1)
`internal/http/binder.go` (2)
`internal/http/builder.go` (2)
`internal/http/chain.go` (1)
`internal/http/context.go` (2)
`internal/http/guards.go` (clean — re-export wrapper)
`internal/http/guards/guards.go` (3)
`internal/http/interceptors.go` (clean — re-export wrapper)
`internal/http/interceptors/interceptors.go` (1)
`internal/http/pagination.go` (clean)
`internal/http/pipes.go` (clean — re-export wrapper)
`internal/http/pipes/pipes.go` (1 + shared BLOCKER-006)
`internal/http/query.go` (clean)
`internal/http/router.go` (clean — interface definitions only)
`internal/pool/context.go` (1)
`internal/reflect/types.go` (1)
`internal/testing/app.go` (clean)
`internal/testing/mocks.go` (2)
`internal/validation/validator.go` (shared BLOCKER-006)

Root files (`app.go`, `provider.go`, `lifecycle.go`, `module.go`, `options.go`, `router.go`, `errors.go`, `pagination.go`, `query.go`) inspected for the "thin re-export" convention — see MINOR-001.

---

## Appendix: Linter baseline (2026-05-17)

All three tools clean against `main`:

```
=== go vet ./... ===
(no output)

=== staticcheck ./... ===
(no output)

=== golangci-lint run --timeout=5m ===
0 issues.
```

This baseline matters because nearly every finding in this report is invisible to the current static-analysis stack. Adding `bodyclose`, `contextcheck`, and a custom analyzer for "function with the same name in two packages" would catch BLOCKER-001 and BLOCKER-006 automatically.

Coverage snapshot (same run):

```
github.com/linkeunid/ligo                              74.6%
github.com/linkeunid/ligo/adapters/echo                17.2%
github.com/linkeunid/ligo/internal/app                 54.2%
github.com/linkeunid/ligo/internal/core/container      68.3%   (dead — see BLOCKER-001)
github.com/linkeunid/ligo/internal/core/lifecycle      35.4%
github.com/linkeunid/ligo/internal/core/logger         87.9%
github.com/linkeunid/ligo/internal/core/module         90.9%
github.com/linkeunid/ligo/internal/di                   0.0%   (the live one — see BLOCKER-001)
github.com/linkeunid/ligo/internal/errors               0.0%
github.com/linkeunid/ligo/internal/http                18.3%
github.com/linkeunid/ligo/internal/http/guards          0.0%
github.com/linkeunid/ligo/internal/http/interceptors    0.0%
github.com/linkeunid/ligo/internal/http/pipes           0.0%
github.com/linkeunid/ligo/internal/pool                 0.0%
github.com/linkeunid/ligo/internal/reflect              0.0%
github.com/linkeunid/ligo/internal/testing              0.0%
github.com/linkeunid/ligo/internal/validation           0.0%
Total                                                  46.8%
```
