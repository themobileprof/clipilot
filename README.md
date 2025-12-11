# CLIPilot 🚀

**A lightweight, offline-first CLI assistant for Linux and mobile terminals**

CLIPilot is an intelligent command-line assistant designed for developers and operations teams working on resource-constrained devices like Android/Termux environments. It provides deterministic, safe multi-step workflows for common dev/ops tasks through a modular architecture with hybrid intent detection.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)

## ✨ Features

- **🔌 Offline-First**: Works without internet connectivity using local keyword search and optional tiny LLM
- **🎯 Hybrid Intent Detection**: 3-layer pipeline (keyword DB → local LLM → online LLM fallback)
- **📦 Modular Architecture**: Download and install task modules on demand
- **🔒 Safety-First**: All commands require explicit user confirmation before execution
- **💾 Lightweight**: Core binary <20MB, optimized for 2-4GB RAM devices
- **🗃️ SQLite Backend**: Fast local caching and state persistence
- **🔄 Flow Engine**: Deterministic multi-step workflows with branching and validation

## 🏗️ Architecture

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
    │ SQLite   │       │ Tiny LLM   │    │  Online   │
    │  Cache   │       │ (Optional) │    │    LLM    │
    └──────────┘       └────────────┘    └───────────┘
```

### Intent Detection Pipeline

1. **Layer 1 - Keyword/DB Search** (Fast, always available)
   - Tokenizes user input
   - Weighted keyword matching against patterns table
   - Scores candidates by relevance, tags, and popularity

2. **Layer 2 - Tiny Local LLM** (Optional, 1-10MB model)
   - Label classifier for ambiguous queries
   - Returns confidence scores
   - 100-500ms inference time

3. **Layer 3 - Online LLM Fallback** (Opt-in)
   - Triggered when offline layers fail
   - Provides explanations and handles novel queries

## 🚀 Quick Start

### Installation

#### From Source
```bash
git clone https://github.com/themobileprof/clipilot.git
cd clipilot
go build -o clipilot ./cmd/clipilot
sudo mv clipilot /usr/local/bin/
```

#### For Termux (Android)
```bash
pkg install golang git
git clone https://github.com/themobileprof/clipilot.git
cd clipilot
go build -o clipilot ./cmd/clipilot
mv clipilot $PREFIX/bin/
```

### First Run

```bash
# Initialize database and download core modules
clipilot init

# Start interactive mode
clipilot

# Or run directly
clipilot "install mysql"
```

## 📖 Usage

### Interactive Mode (REPL)

```bash
$ clipilot
CLIPilot v1.0.0 - Type 'help' for commands

> help
Available commands:
  help                    - Show this help message
  search <query>          - Search for modules
  run <module_id>         - Execute a specific module
  modules list            - List installed modules
  modules install <id>    - Download and install a module
  modules remove <id>     - Remove a module
  settings                - Configure CLIPilot
  logs                    - View execution history
  exit                    - Exit CLIPilot

> install mysql
Found module: install_mysql (v1.0.0)
Description: Install and configure MySQL with secure defaults

Starting flow...
[Step 1/4] Detecting operating system...
  Command: uname -a
  Run this command? [y/N]: y
  ✓ Detected: Ubuntu 22.04

[Step 2/4] Installing MySQL server...
  Command: sudo apt-get update && sudo apt-get install -y mysql-server
  Run this command? [y/N]: y
  ✓ MySQL installed successfully

[Step 3/4] Securing MySQL installation...
  Command: sudo mysql_secure_installation
  Run this command? [y/N]: y
  ✓ Security configuration complete

[Step 4/4] Verifying installation...
  ✓ MySQL is running (version 8.0.35)

Module execution complete!
```

### Direct Command Mode

```bash
# Run a specific module
clipilot run install_mysql

# Dry run (show commands without executing)
clipilot --dry-run run install_mysql

# Search for modules
clipilot search "setup docker"

# List available modules
clipilot modules list

# Install a module
clipilot modules install docker_setup
```

## 🧩 Module System

### Module Structure

Modules are YAML files defining multi-step workflows:

```yaml
name: install_mysql
id: org.themobileprof.install_mysql
version: 1.0.0
description: Install and configure MySQL (with secure defaults)
tags: [mysql, database, install]
provides:
  - mysql_installed
requires:
  - detect_os
flows:
  main:
    start: detect_os
    steps:
      detect_os:
        type: action
        run_module: detect_os
      install_branch:
        type: branch
        based_on: os_type
        map:
          ubuntu: install_ubuntu
          debian: install_debian
          termux: install_termux
      install_ubuntu:
        type: instruction
        message: "Installing MySQL on Ubuntu..."
        command: "sudo apt-get update && sudo apt-get install -y mysql-server"
        validate:
          - check_command: "mysql --version"
        next: secure_mysql
      secure_mysql:
        type: instruction
        message: "Securing MySQL installation..."
        command: "sudo mysql_secure_installation"
        next: done
      done:
        type: terminal
        message: "MySQL installation complete!"
```

### Built-in Modules

- `detect_os` - Detect operating system and distribution
- `git_setup` - Configure Git with best practices
- `docker_install` - Install Docker and Docker Compose
- `nginx_setup` - Install and configure Nginx web server

### Creating Custom Modules

1. Create a YAML file in `modules/` directory
2. Define metadata (name, id, version, description, tags)
3. Specify dependencies in `requires`
4. Design flow steps with actions, branches, and validations
5. Test with `clipilot run your_module_id`

See `docs/module_development.md` for detailed guide.

## ⚙️ Configuration

Configuration file: `~/.clipilot/config.yaml`

```yaml
# Enable/disable online LLM fallback
online_mode: false

# Online API configuration (optional)
api_key: ""
api_endpoint: "https://api.example.com/v1/chat"

# Confidence thresholds
thresholds:
  keyword_search: 0.6
  local_llm: 0.6

# Auto-confirm for safe commands (use with caution)
auto_confirm: false

# Database location
db_path: "~/.clipilot/clipilot.db"

# Telemetry (opt-in, anonymous)
telemetry_enabled: false
```

## 🔒 Security & Privacy

- **No Auto-Execution**: All commands require explicit user confirmation
- **Opt-in Telemetry**: Anonymous usage stats only with consent
- **No PII Collection**: We never collect personal information
- **Offline-First**: Works completely offline by default
- **Command Visibility**: See exactly what will run before confirming
- **Dry-Run Mode**: Preview all actions without execution

## 🧪 Development

### Prerequisites

- Go 1.21 or higher
- SQLite3
- (Optional) GCC for CGO if using tiny LLM

### Building from Source

```bash
# Clone repository
git clone https://github.com/themobileprof/clipilot.git
cd clipilot

# Install dependencies
go mod download

# Run tests
go test ./...

# Build binary
go build -o clipilot ./cmd/clipilot

# Run
./clipilot
```

### Running Tests

```bash
# Unit tests
go test ./internal/... ./pkg/...

# Integration tests
go test -tags=integration ./...

# With coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Project Structure

```
clipilot/
├── cmd/
│   └── clipilot/          # Main application entry point
│       └── main.go
├── internal/              # Private application code
│   ├── db/               # Database layer and migrations
│   ├── engine/           # Flow execution engine
│   ├── intent/           # Intent detection pipeline
│   ├── modules/          # Module loader and manager
│   └── ui/               # REPL and user interface
├── pkg/                   # Public libraries
│   └── models/           # Data structures and models
├── modules/               # Built-in module definitions
│   ├── detect_os.yaml
│   ├── git_setup.yaml
│   └── docker_install.yaml
├── docs/                  # Documentation
└── go.mod
```

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas for Contribution

- **New Modules**: Add support for more tools and workflows
- **OS Support**: Expand compatibility (Alpine, Fedora, etc.)
- **Performance**: Optimize for low-memory devices
- **Tiny LLM**: Improve local classification accuracy
- **Documentation**: Improve guides and examples

## 📝 License

This project is licensed under the MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by the needs of developers in emerging markets
- Built for the Termux and mobile Linux community
- Designed with privacy and offline-first principles

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/themobileprof/clipilot/issues)
- **Discussions**: [GitHub Discussions](https://github.com/themobileprof/clipilot/discussions)
- **Documentation**: [docs/](docs/)

---

**Made with ❤️ for developers working on constrained devices**
