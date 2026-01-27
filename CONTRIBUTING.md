# Contributing to CLIPilot

We welcome contributions to CLIPilot! Whether you're fixing bugs, adding features, or improving documentation, your help is appreciated.

## Development Setup

### Prerequisites
- **Go 1.21+**: Required for building the project.
- **Git**: For version control.
- **SQLite**: (Optional) For inspecting the database manually.

### Initial Setup
1. Fork and clone the repository:
   ```bash
   git clone https://github.com/themobileprof/clipilot.git
   cd clipilot
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Run the tests to ensure everything is working:
   ```bash
   ./scripts/test.sh
   # or
   go test ./...
   ```

## 🏗️ Architecture

CLIPilot follows a modular architecture designed for low-memory environments.

```
┌─────────────────────────────────────────────────────────┐
│                    User (REPL/CLI)                      │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                  Core Engine (Go)                       │
│  ┌──────────────┐  ┌─────────────┐  ┌────────────────┐ │
│  │   Intent     │  │    Flow     │  │    Module      │ │
│  │  Detection   │→ │   Runner    │→ │   Manager      │ │
│  └──────────────┘  └─────────────┘  └────────────────┘ │
└────────┬────────────────────┬─────────────────┬─────────┘
         │                    │                 │
    ┌────▼─────┐       ┌─────▼──────┐    ┌────▼──────┐
    │ SQLite   │       │  TF-IDF    │    │  Online   │
    │  Cache   │       │ Matcher    │    │    LLM    │
    └──────────┘       └────────────┘    └───────────┘
```

### Intent Detection Pipeline

1. **Layer 1 - In-Memory Catalog** (Primary)
   - Searches a curated list of common commands (`git`, `cp`, `tar`, etc.).
   - Very fast execution (< 1ms).
   - Resilient against database failures.

2. **Layer 2 - System Index** (Secondary)
   - Uses SQLite FTS to search thousands of installed system binaries (via map pages).
   - Weighted keyword matching.

3. **Layer 3 - Offline Intelligence** (Hybrid Matcher)
   - Pure Go TF-IDF similarity matching.
   - Intent extraction (show, find, kill, monitor).
   - Category boosting (networking, process, filesystem).

### Core Components

- **`internal/intent`**: Handles converting user natural language into "Intents" (Module IDs or Commands).
- **`internal/engine`**: The execution engine for Modules (`.yaml` workflow files).
- **`internal/ui`**: The interactive REPL and text integration.
- **`internal/commands`**: Manages the In-Memory Catalog and System Index.
- **`internal/db`**: SQLite migration and connection management.

## 🧩 Module System

Modules are defined in YAML files in the `modules/` directory. They define deterministic workflows.

### Structure
```yaml
id: my_module
name: My Utility
description: Does something useful
tags: [utility, example]
steps:
  step1:
    type: action
    command: echo "Hello World"
    next: done
  done:
    type: terminal
    message: "Done!"
```

See `docs/module_development.md` for a complete reference.

## Submitting Pull Requests
1. Create a new branch (`git checkout -b feature/my-feature`).
2. Make your changes.
3. Add specific tests for your changes.
4. Run the full test suite (`./scripts/test.sh`).
5. Commit and push.
6. Open a PR on GitHub.

Thank you for helping make the CLI smarter!
