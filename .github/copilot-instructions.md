# CLIPilot AI Agent Instructions

## Project Overview

CLIPilot is a **lightweight, offline-first CLI assistant** for Linux and mobile (Android/Termux) environments. It executes deterministic multi-step workflows defined in YAML modules using a 3-layer hybrid intent detection pipeline (keyword search → local LLM → online LLM fallback).

**Architecture:** Go 1.24+ with embedded SQLite (modernc.org/sqlite, pure Go - no CGO needed for cross-platform builds), YAML-based module system, multi-platform support (Linux/macOS/Android).

## Critical Build & Development Workflow

### Building
```bash
# Main CLI binary
go build -o clipilot ./cmd/clipilot

# Registry server (web-based module sharing)
go build -o registry ./cmd/registry

# Release builds (strip debug symbols)
go build -ldflags="-s -w" -o clipilot ./cmd/clipilot

# Android/Termux builds (no CGO, pure Go)
GOOS=linux GOARCH=arm64 go build -o clipilot-android-arm64 ./cmd/clipilot
```

**Important:** SQLite is embedded via `modernc.org/sqlite` (pure Go), enabling zero-dependency cross-compilation. Never use `mattn/go-sqlite3` (requires CGO).

### Testing
```bash
go test ./...                          # All tests
go test -cover ./...                   # With coverage
go test -coverprofile=coverage.txt ./... # Generate coverage report
go test -tags=integration ./...        # Integration tests
go test -v -race ./...                 # Race detection (CI/CD)
```

**Test structure:**
- Unit tests: `*_test.go` in each package (db, intent, modules, engine, config)
- Integration tests: `integration_test.go` (requires build tag)
- CI runs: unit tests + race detector + coverage upload to Codecov

### Initialization
```bash
# Initialize database and load modules
clipilot --init --load=modules/

# Reset database (dev workflow)
clipilot --reset --load=modules/
```

## Module System (Core Architecture)

### YAML Module Structure
Modules define multi-step workflows with branching logic. Located in `modules/*.yaml`.

**Key concepts:**
- **Flows**: Named execution graphs (typically `main`)
- **Steps**: Typed nodes (action, branch, instruction, terminal)
- **State**: Passed between steps via template variables `{{.variable}}`
- **Dependencies**: `requires` and `provides` for module relationships

**Example pattern (see [modules/copy_file.yaml](../modules/copy_file.yaml)):**
```yaml
name: copy_file
id: org.themobileprof.copy_file
version: 1.0.0
tags: [file-ops, atomic, filesystem, copy]
requires: [check_file_exists, check_disk_space]
provides: [file_copied]

flows:
  main:
    start: validate_source
    steps:
      validate_source:
        type: action
        message: "Validating source file: {{.source}}"
        command: |
          # Expand tilde, check file exists
          echo "$SOURCE" > /tmp/clipilot_copy_source.txt
        next: check_destination
      
      check_destination:
        type: branch
        based_on: file_exists
        map:
          true: confirm_overwrite
          false: copy_file
```

**Step Types:**
- `action`: Execute shell command, capture output
- `branch`: Conditional routing based on state variables
- `instruction`: Display message with optional command execution
- `terminal`: End flow with message

### Module Loading Pipeline
1. YAML parsed via `internal/modules/loader.go` → `pkg/models/module.go`
2. Module metadata + JSON-serialized content stored in SQLite `modules` table
3. Keywords extracted to `intent_patterns` table for search (weighted by pattern type)
4. Flow steps indexed in `steps` table for execution

## Intent Detection (3-Layer Hybrid)

**Pipeline implementation:** `internal/intent/keyword.go`, `internal/intent/tinyllm.go`

### Layer 1: Keyword Search (Always Available)
- Tokenizes user input, queries `intent_patterns` table
- Weighted scoring: exact matches > tags > descriptions
- Threshold: 0.6 (configurable via `~/.clipilot/config.yaml`)
- Returns top candidates with confidence scores

### Layer 2: Tiny Local LLM (Optional, TODO)
- Label classifier for ambiguous queries (1-10MB model)
- Inference: 100-500ms
- Threshold: 0.6

### Layer 3: Online LLM Fallback (Opt-in)
- Triggered when confidence < threshold
- Requires `online_mode: true` in config

**Development pattern:** Always implement Layer 1 first. Layers 2-3 are optional enhancements.

## Database Schema & Conventions

**SQLite migrations:** Embedded in `internal/db/migration.sql` via `//go:embed`.

**Key tables:**
- `modules`: Module metadata + JSON content (single source of truth)
- `intent_patterns`: Tokenized keywords for search (pattern, weight, module_id)
- `steps`: Parsed flow steps for execution
- `logs`: Execution history (session_id, status, duration_ms)
- `state`: Runtime state (key-value pairs, session-scoped)
- `settings`: User preferences (online_mode, thresholds, registry_url)

**Access pattern:** Use `internal/db/db.go` wrapper methods, not raw SQL in business logic.

## Flow Execution (Engine)

**Implementation:** `internal/engine/runner.go`

**Execution model:**
1. Create `ExecutionContext` with session_id, module_id, state map
2. Start at flow's `start` step
3. Execute steps sequentially, update context state
4. Branch/loop via `next` field or `branch.map` routing
5. User confirmation required unless `--dry-run` or `auto_confirm: true`
6. Log all executions to `logs` table (status: started → completed/failed)

**State management:**
- Template variables: `{{.variable}}` expanded from context state
- Commands capture stdout/stderr, can populate state
- Temp files convention: `/tmp/clipilot_*` for inter-step communication

## Registry Server (Module Sharing)

**Purpose:** Web-based module upload/download for community sharing.

**Architecture:**
- Entry: `cmd/registry/main.go`
- Handlers: `server/handlers/handlers.go`
- Storage: Separate SQLite DB (`data/registry.db`), file uploads in `data/uploads/`
- Auth: Basic username/password (session-based)

**Key routes:**
- `GET /modules` → Browse modules (public)
- `POST /api/upload` → Upload module YAML (authenticated)
- `GET /modules/:id` → Download module YAML
- `GET /my-modules` → User's uploaded modules

**Docker deployment:** `Dockerfile.registry`, `docker-compose.yml` (port 8082)

## Termux/Android First-Class Support

**Philosophy:** Android/Termux is NOT an afterthought - it's a primary target.

**Optimizations:**
- Pure Go builds (no CGO) via `modernc.org/sqlite`
- Dedicated `clipilot-android-arm64` / `clipilot-android-arm` binaries
- Termux-specific modules: `termux_setup.yaml`, `termux_*.yaml`
- Detection: `TERMUX_VERSION` or `PREFIX` env vars
- Installation: `$PREFIX/bin` instead of `/usr/local/bin`

**Release workflow:** `.github/workflows/release.yml` builds 7 platforms (Linux x3, macOS x2, Android x2).

## Code Conventions & Patterns

### Error Handling
```go
// Wrap errors with context
return fmt.Errorf("failed to load module: %w", err)

// Check before defer rollback
defer func() {
    _ = tx.Rollback() // Ignore error (might be committed)
}()
```

### Configuration Loading
- YAML config at `~/.clipilot/config.yaml` (created with defaults if missing)
- Never hardcode paths - use `os.UserHomeDir()` + `filepath.Join()`
- Respect `--config` flag override

### Module Development
- **Namespace IDs:** `org.themobileprof.module_name` (prevent collisions)
- **Tags:** Include platform tags (`termux`, `linux`, `macos`) when applicable
- **Size estimation:** `size_kb` field for dependency planning
- **Test modules:** Use `--dry-run` to validate without execution

### SQL Queries
- Use prepared statements (`?` placeholders)
- `ON CONFLICT DO UPDATE` for idempotent imports
- Index on `intent_patterns.pattern` for fast keyword search

## When Making Changes

### Adding New Modules
1. Create YAML in `modules/` with unique `id`
2. Define clear `tags`, `provides`, `requires`
3. Use `action` steps with validation commands
4. Test: `clipilot --reset --load=modules/` then `clipilot run module_id`

### Modifying Core Engine
1. Update `internal/engine/runner.go` for execution logic
2. Extend `pkg/models/module.go` if adding step types
3. Update `internal/db/migration.sql` for schema changes
4. Re-run tests: `go test ./internal/engine/...`

### Registry Changes
1. Handlers: `server/handlers/handlers.go`
2. Templates: `server/templates/*.html` (embedded via `//go:embed`)
3. Rebuild Docker: `docker build -f Dockerfile.registry -t clipilot-registry .`

## Common Pitfalls

1. **CGO dependency:** Never introduce `mattn/go-sqlite3` - breaks Android builds
2. **Path assumptions:** Always expand `~` and handle `$PREFIX` (Termux)
3. **Command confirmation:** Respect `dryRun` and `autoYes` in runner
4. **Module conflicts:** Check `id` uniqueness before import
5. **Template syntax:** Use `{{.var}}` not `${var}` in YAML commands

## CI/CD Pipeline

**GitHub Actions workflows:** `.github/workflows/`

### Test Workflow (`test.yml`)
Runs on every push/PR to main/develop:
1. **Unit Tests** - Go tests with race detector and coverage
2. **Build Verification** - Compiles CLI and registry binaries
3. **Linting** - golangci-lint with 5min timeout
4. **Coverage Upload** - Results sent to Codecov

**Matrix:** Ubuntu + macOS, Go 1.24

### Release Workflow (`release.yml`)
Triggered on version tags (`v*`):
1. Cross-compile for 7 platforms (Linux x3, macOS x2, Android x2)
2. Strip debug symbols with `-ldflags="-s -w"`
3. Create GitHub Release with binaries + modules tarball

**Key flag:** `CGO_ENABLED=0` for pure Go builds (required for Android)

### Writing Tests
- **Unit tests:** Test individual functions, use temp DBs
- **Integration tests:** Tag with `// +build integration`, test CLI workflows
- **Benchmarks:** Use `func Benchmark*` for performance testing
- **Test setup:** Use `setupTestDB()` helper for database tests

**Run before PR:**
```bash
go test -v -race -cover ./...
go test -tags=integration ./...
```

## Key Files Quick Reference

- [cmd/clipilot/main.go](../cmd/clipilot/main.go) - CLI entry point, flag parsing
- [internal/engine/runner.go](../internal/engine/runner.go) - Flow execution engine
- [internal/intent/keyword.go](../internal/intent/keyword.go) - Intent detection Layer 1
- [internal/modules/loader.go](../internal/modules/loader.go) - YAML parsing & DB import
- [internal/ui/repl.go](../internal/ui/repl.go) - Interactive mode (commands: help, search, run, logs)
- [modules/copy_file.yaml](../modules/copy_file.yaml) - Example module with branching
- [docs/TERMUX.md](../docs/TERMUX.md) - Termux-specific development guide
- [integration_test.go](../integration_test.go) - End-to-end CLI tests

---
**Questions?** Check [CONTRIBUTING.md](../CONTRIBUTING.md) for PR workflow, [TESTING.md](../TESTING.md) for registry testing, and `docs/` for detailed architecture.
