# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
make build       # produces ./gve binary with version/commit/date ldflags
make install     # installs to $GOPATH/bin

# Test
make test        # go test ./...
go test ./internal/cmd/... -run TestUserJourney  # run a specific test

# Clean
make clean
```

## Architecture

GVE is a CLI tool (built with [cobra](https://github.com/spf13/cobra)) that scaffolds, develops, builds, and ships full-stack Go + Vite projects as single self-contained binaries (Go backend embeds the compiled frontend via `go:embed`).

### Package layout

| Package | Role |
|---|---|
| `cmd/gve/` | Entry point — calls `internal/cmd.Execute()` |
| `internal/cmd/` | All cobra commands; commands directly invoke asset/template/lock/runner packages |
| `internal/asset/` | Registry lookup, file copying, Thrift codegen, diff/sync logic for UI and API assets |
| `internal/template/` | Go text/template scaffolding for new projects and generated API files |
| `internal/lock/` | Read/write `gve.lock` — records pinned UI/API asset versions |
| `internal/config/` | Default registry URLs, cache directory resolution |
| `internal/runner/` | Concurrent process runner for `gve dev` (Go + Vite with prefixed output) |
| `internal/logrotate/` | Log-rotating writer used by the runner |
| `internal/semver/` | Semver comparison helpers |
| `internal/version/` | Build-time version variables (injected via ldflags) |

### Asset system (UI and API)

Two asset types share a common manager (`internal/asset/manager.go`):

- **UI assets** (`wk-ui` registry): React/Tailwind components copied to `site/src/shared/ui/<name>/`. A `meta.json` beside each versioned asset describes its files and optional npm deps. Installing a UI asset also injects its `deps` into the project's `site/package.json`.
- **API assets** (`wk-api` registry): Thrift IDL files (`.thrift`) plus pre-built stubs (`.go`, `.ts`) copied to `api/<namespace>/<name>/<version>/`. Only the `.thrift` source is placed in `api/`; generated Go and TypeScript code is produced by `gve api generate`.

Both registries are git-cloned into a local cache (default `~/.cache/gve/`) on first use. A `registry.json` at the root of each cache repo maps asset names to versioned paths.

### API code generation (`gve api generate`)

`GenerateThriftArtifacts` in `internal/asset/thrift_gen.go`:
1. Parses a `.thrift` file using `cloudwego/thriftgo/parser` (AST only, no subprocess).
2. Renders Go struct definitions via `internal/template` (`api_types_go.tmpl`).
3. Renders a Go HTTP client (`api_client_go.tmpl`) into `internal/api/<rel-path>/`.
4. Renders a TypeScript client (`api_client_ts.tmpl`) into `site/src/api/<rel-path>/`.

### Lock file (`gve.lock`)

`gve.lock` records the registry URLs and the pinned version of every installed UI and API asset. The `gve sync` command uses it to restore or verify the working tree. Tests create fake lock files with `lock.New(...)` / `lock.Save(...)`.

### Testing conventions

- Tests in `internal/cmd/` that need a full asset environment call `setupFakeAssetCache(t)` and `setupProject(t, cacheDir)` (defined in `user_journey_test.go`) to create temporary fixture trees — no network or git operations.
- Table-driven tests with `t.Run(...)` subtests are the norm throughout.
