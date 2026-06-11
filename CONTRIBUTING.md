# Contributing to CLIPilot Registry

CLIPilot is the **server** (module registry + semantic search API). The **CLI client** is [Clio](https://github.com/themobileprof/clio) — do not add a second CLI to this repo.

## Development Setup

```bash
git clone https://github.com/themobileprof/clipilot.git
cd clipilot
go mod download
go test ./...
go build -o clipilot-server ./cmd/registry
```

## Architecture

```
Clio (client)  ──HTTPS──▶  CLIPilot Registry (this repo)
                           ├── Web UI (modules, upload)
                           ├── /api/v1/modules/* (sync)
                           └── /api/commands/search (remote fallback)
```

### Server packages

| Path | Role |
|------|------|
| `cmd/registry/` | HTTP server entry point |
| `server/handlers/` | API + web routes |
| `server/catalog/` | Common commands search (embedded YAML) |
| `server/bootstrap/` | Seed builtin modules on startup |
| `internal/models/` | Shared module types |

### Semantic search (`POST /api/commands/search`)

1. **Catalog** — keyword search over `server/catalog/common_commands.yaml` (+ Termux essentials)
2. **Gemini** (optional, `GEMINI_API_KEY`) — when catalog confidence is low
3. **SQLite cache** — 7-day query cache, shared across users

### Modules

YAML workflows live in `modules/`. Clio syncs them via `/api/v1/modules/changed`.

## Making changes

- **More command matches**: edit `server/catalog/common_commands.yaml` or `server/catalog/tokens.go` (Pidgin slang)
- **API changes**: update `docs/CLIO_API_REQUIREMENTS.md` and [Clio](https://github.com/themobileprof/clio)
- **Do not** add `cmd/clipilot` or client REPL code — use Clio instead

## Tests

```bash
go test ./server/...
go test -tags=integration ./...   # Docker optional
```
