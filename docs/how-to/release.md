# How to Release ligo

## Prerequisites

- `git` and `gh` CLI installed and authenticated
- All changes committed and pushed to `main`

## Release

Run the release script from the project root:

```bash
./scripts/release.sh          # patch bump: v0.1.0 → v0.1.1
./scripts/release.sh minor    # minor bump: v0.1.0 → v0.2.0
./scripts/release.sh major    # major bump: v0.1.0 → v1.0.0
```

The script will:
1. Detect the latest semver tag automatically
2. Increment the version
3. Push the current branch
4. Create and push the annotated tag
5. Create a GitHub Release

## What gets published

Go modules are published via the [Go module proxy](https://proxy.golang.org). Once the tag is pushed, the new version becomes available automatically — no manual upload needed.

Users install with:

```bash
go get github.com/linkeunid/ligo@v0.1.1
```

## Verifying the release

```bash
GOPROXY=direct go list -m github.com/linkeunid/ligo@latest
```

> The module proxy caches `@latest` for a few minutes. Use an explicit version tag to install immediately.

## Dependency order

When releasing multiple ligo packages together, release in this order to avoid dependency resolution issues:

1. `ligo`
2. `ligo-memory`, `ligo-validator` (depend on ligo)
3. `ligo-boilerplate` (depends on all above)
4. `ligo-cli` (references boilerplate templates)
