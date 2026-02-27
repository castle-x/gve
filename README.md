# gve

A single CLI that scaffolds, develops, builds, and ships Go + Vite projects as one self-contained binary — frontend embedded via `go:embed`, no nginx or separate static hosting needed.

## About

Full-stack Go projects typically require juggling separate frontend and backend toolchains, deployment configs, and asset management. **gve** collapses that into one workflow: `init` a project, `add` UI components and API contracts from shared asset libraries, `dev` with hot-reload for both Go and Vite, then `build` a single binary that serves everything.

Two companion asset libraries keep teams consistent:

| Repository | Purpose |
|---|---|
| [wk-ui](https://github.com/castle-x/wk-ui) | Shared UI components (React + Tailwind wrappers around Radix UI) |
| [wk-api](https://github.com/castle-x/wk-api) | Shared API contracts (Thrift IDL + pre-generated Go/TypeScript clients) |

## Features

- **Project scaffolding** — `gve init` generates a Go backend + React/Vite frontend with sane defaults
- **UI asset management** — install, diff, sync, and upgrade shared UI components across projects
- **API contract management** — pull pre-generated Thrift clients (Go + TypeScript) without local toolchains
- **One-command dev** — `gve dev` runs Go (with Air hot-reload) and Vite concurrently, prefixed output
- **Single-binary build** — `gve build` compiles frontend into Go via `embed`, supports cross-compilation
- **Background service** — `gve run` with smart rebuild, PID management, and daily log rotation
- **Team sync** — `gve.lock` tracks asset versions; `gve sync` restores them on `git pull`
- **Environment check** — `gve doctor` verifies Go, Node, pnpm, Git, and Air

## Installation

**Requires:** Go 1.22+

```bash
go install github.com/castle-x/gve/cmd/gve@latest
```

Verify:

```bash
gve version
gve doctor
```

## Quick Start

```bash
gve init my-app && cd my-app

# Install frontend dependencies
cd site && pnpm install && cd ..

# Add a UI component and an API contract
gve ui add button
gve api add example-project/user@v1

# Start developing
gve dev
# [go]   Server starting on :8080
# [vite] Local: http://localhost:5173

# Build a single binary
gve build
./dist/my-app
```

## Usage

### Project Lifecycle

```bash
gve init <name>          # Scaffold project
gve dev                  # Go + Vite hot-reload
gve build [--os --arch]  # Single binary (cross-compile)
gve run                  # Background with log rotation
gve run stop|restart|status|logs
```

### Asset Management

```bash
gve ui add <asset>[@ver]                    # Install UI component
gve ui list                                 # List installed assets
gve ui diff <asset>                         # Local changes vs. library
gve ui sync [asset]                         # Upgrade with conflict detection

gve api add <project>/<resource>[@ver]      # Install API contract
gve api sync                                # Upgrade API contracts

gve sync                                    # Restore all from gve.lock
gve status                                  # Show available updates
```

### Asset Library Maintenance

```bash
# Inside wk-ui or wk-api repository
gve registry build       # Scan assets/ → generate registry.json
```

## Project Structure

After `gve init my-app`:

```
my-app/
├── cmd/server/main.go        # Go entry point
├── internal/                  # Business logic
├── api/                       # API contracts (gve api add)
├── site/                      # Frontend (React + Vite)
│   ├── embed.go               # go:embed all:dist
│   ├── package.json
│   ├── src/
│   │   ├── app/               # Routes, providers, styles
│   │   ├── views/             # Pages
│   │   └── shared/ui/         # UI assets (gve ui add)
│   └── ...
├── gve.lock                   # Asset version lock (commit this)
└── Makefile
```

## Contributing

```bash
git clone git@github.com:castle-x/gve.git
cd gve
make test      # Run tests
make install   # Build and install to $GOPATH/bin
```

### Cursor Skill

This repo ships a Cursor Agent skill with GVE command reference and conventions:

```bash
cp -r skills/gve ~/.cursor/skills/gve
```

## License

MIT
