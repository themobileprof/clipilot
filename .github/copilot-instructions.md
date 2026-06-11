# CLIPilot Registry — AI Agent Instructions

**CLIPilot is a server-only project.** The CLI client is **[Clio](https://github.com/themobileprof/clio)** — never add `cmd/clipilot` or a REPL here.

## Build

```bash
go build -o clipilot-server ./cmd/registry
go test ./server/...
```

## Key paths

- `cmd/registry/main.go` — routes, env config
- `server/handlers/commands.go` — `POST /api/commands/search` for Clio
- `server/catalog/` — embedded command catalog + Pidgin slang
- `server/handlers/api_v1.go` — module sync API for Clio
- `modules/` — YAML workflows synced to clients

## Clio integration

Clio sends:
```json
{"query": "data no dey work", "os": "linux", "arch": "arm64"}
```

Response:
```json
{
  "candidates": [{"name": "ping", "description": "...", "usage": "ping -c 4 host"}],
  "results": [...],
  "source": "catalog",
  "cached": false
}
```

## Do not

- Add CLI binaries, REPL, or intent detection to this repo
- Delete `internal/models/` or `server/bootstrap/` (server needs them)
- Use CGO SQLite — use `modernc.org/sqlite`
