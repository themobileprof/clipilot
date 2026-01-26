# Contributing to CLIPilot

We welcome contributions to CLIPilot! Whether you're fixing bugs, adding features, or improving documentation, your help is appreciated.

## Development Setup

### Prerequisites
- **Go 1.24+**: Required for finding the latest features.
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

## Architecture Overview

CLIPilot is composed of several internal packages:
- **`internal/intent`**: Handles converting user natural language into "Intents" (Module IDs or Commands). Uses a hybrid approach (Keyword + TF-IDF).
- **`internal/engine`**: The execution engine for Modules (`.yaml` workflow files).
- **`internal/ui`**: The interactive REPL and text integration.
- **`internal/commands`**: Indexing system for indexing local man pages.

## Creating Modules

Modules are defined in YAML files in the `modules/` directory.

### Structure
```yaml
id: my_module
name: My Utility
description: Does something useful
steps:
  - type: action
    command: echo "Hello World"
```

## Running Locally

To run the CLI without installing:
```bash
go run ./cmd/clipilot
```

## Submitting Pull Requests
1. Create a new branch (`git checkout -b feature/my-feature`).
2. Make your changes.
3. specific tests for your changes.
4. Run the full test suite (`./scripts/test.sh`).
5. Commit and push.
6. Open a PR on GitHub.

Thank you for helping make the CLI smarter!
