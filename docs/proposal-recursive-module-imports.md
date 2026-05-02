# Proposal: Recursive Controller Binding for Module Imports

> **Status: Implemented** — shipped with expand-first architecture (`ExpandModules` + `bindModuleControllers` recursion). See `docs/features/modules.md` for usage.

## Problem

`ligo.Imports(...)` currently only shares **providers** recursively — it does not bind **controllers** or **middleware** from imported modules.

This means the following pattern silently drops all routes from child modules:

```go
// module/main.go
func Main() ligo.Module {
    return ligo.NewModule("main",
        ligo.Imports(
            Auth(),
            User(),
            File(),
            Health(),
            Root(),
        ),
    )
}

// cmd/api/main.go
app.Register(module.Main()) // ← routes from Auth, User, etc. are never registered
```

Only `"main module initialized"` is logged; all child routes are silently missing.

## Root Cause

`BuildModule` (DI side, `internal/app/app.go`) recurses into `mod.Imports` — so providers work correctly.

`bindModuleControllers` (HTTP side, `internal/http/binder.go`) does **not** recurse — it only binds controllers on the module it receives directly:

```go
// internal/http/binder.go — current
func (b *Binder) bindModuleControllers(mod module.Module) error {
    // ... binds mod.Controllers ...
    // mod.Imports is never visited
    return nil
}
```

## Proposed Fix

Add a single loop at the end of `bindModuleControllers` to recurse into imports, mirroring what `BuildModule` already does:

```go
// internal/http/binder.go — proposed
func (b *Binder) bindModuleControllers(mod module.Module) error {
    // Resolve module middleware
    var modMw []Middleware
    for _, mc := range mod.Middlewares {
        mw, err := b.resolveMiddleware(mc)
        if err != nil {
            return err
        }
        modMw = append(modMw, mw)
    }

    // Apply module middleware if present
    if len(modMw) > 0 {
        moduleRouter := b.router.Group("/" + mod.Name)
        for _, mw := range modMw {
            moduleRouter.Use(mw)
        }
        for _, cc := range mod.Controllers {
            if err := b.bindController(cc, moduleRouter, mod.Name); err != nil {
                return err
            }
        }
    } else {
        for _, cc := range mod.Controllers {
            if err := b.bindController(cc, b.router, mod.Name); err != nil {
                return err
            }
        }
    }

    // ✅ NEW: recurse into imported child modules
    for _, child := range mod.Imports {
        if err := b.bindModuleControllers(child); err != nil {
            return err
        }
    }

    return nil
}
```

The diff is exactly **5 lines**:

```diff
+   // Recurse into imported child modules
+   for _, child := range mod.Imports {
+       if err := b.bindModuleControllers(child); err != nil {
+           return err
+       }
+   }
```

## Expected Behaviour After Fix

```go
app.Register(module.Main()) // registers routes from all imported child modules
```

Startup log would show each child being initialized and all routes mapped:

```
level=INFO msg="main module initialized"   context=di.container
level=INFO msg="Mapped {GET, /users} route"
level=INFO msg="Mapped {GET, /users/:id} route"
level=INFO msg="Mapped {POST, /users} route"
...
```

## Design Considerations

| Concern | Notes |
|---------|-------|
| **Circular imports** | `BuildModule` already recurses without cycle detection. The same assumption (no cycles) applies here; the binder would loop infinitely on cycles just as the DI side would. A visited-set guard would be a nice addition to both sides. |
| **Module middleware scope** | A parent module with middleware currently creates a group prefixed with the module name. Child controllers would be registered under the parent group's router — this is the correct behavior since they already define their own path prefixes in `Routes()`. |
| **Backwards compatibility** | Fully backwards-compatible. Flat `app.Register(A, B, C)` is unaffected. The new behavior only activates when a module actually has `Imports`. |

## Workaround (Until Fixed)

Register all modules individually at the top level:

```go
app.Register(
    module.Auth(),
    module.User(),
    module.File(),
    module.Health(),
    module.Root(),
)
```
