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

#### One-Line Install (Linux/macOS/Termux)
```bash
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/install.sh | bash
```

This script will:
- Detect your platform (Linux/macOS, amd64/arm64/armv7)
- Download the appropriate binary from latest GitHub Release
- Install it to `/usr/local/bin` (or `$HOME/.local/bin` if no sudo)
- Download default modules (detect_os, git_setup, docker_install)
- Initialize the database
- Set up configuration directory at `~/.clipilot`

**Supported Platforms:**
- Linux: amd64 (x86_64), arm64 (aarch64), armv7
- macOS: amd64 (Intel), arm64 (Apple Silicon)

#### Manual Installation
```bash
# Download from GitHub Releases
# Visit: https://github.com/themobileprof/clipilot/releases/latest
# Download the appropriate binary for your platform

# Example for Linux amd64:
curl -LO https://github.com/themobileprof/clipilot/releases/latest/download/clipilot-linux-amd64.tar.gz
tar -xzf clipilot-linux-amd64.tar.gz
chmod +x clipilot-linux-amd64
sudo mv clipilot-linux-amd64 /usr/local/bin/clipilot

# Or for Termux (no sudo needed)
mv clipilot-linux-arm64 $PREFIX/bin/clipilot

# Initialize with default modules
mkdir -p ~/.clipilot/modules
cd ~/.clipilot/modules
curl -LO https://github.com/themobileprof/clipilot/releases/latest/download/clipilot-modules.tar.gz
tar -xzf clipilot-modules.tar.gz
clipilot --init --load=~/.clipilot/modules
```

#### From Source (For Developers)
```bash
git clone https://github.com/themobileprof/clipilot.git
cd clipilot
go mod download
go build -o clipilot ./cmd/clipilot
sudo mv clipilot /usr/local/bin/

# Download modules and initialize
mkdir -p ~/.clipilot/modules
cp modules/*.yaml ~/.clipilot/modules/
clipilot --init --load=~/.clipilot/modules
```

### First Run

```bash
# Start interactive mode
clipilot

# Or run directly
clipilot "setup git"

# Show available commands
clipilot help
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

CLIPilot creates a configuration file at `~/.clipilot/config.yaml` on first run with sensible defaults.

**Why `~/.clipilot/`?**
- Standard Linux convention for user-specific app data
- Keeps your home directory clean
- Easy to find and backup
- Follows XDG Base Directory spirit

**Default configuration:**

```yaml
# Enable/disable online LLM fallback
online_mode: false

# Auto-confirm commands (use with caution!)
auto_confirm: false

# Online API configuration (optional, only used if online_mode is true)
api_key: ""
api_endpoint: ""

# Confidence thresholds for intent detection
thresholds:
  keyword_search: 0.6  # Minimum confidence for keyword matches
  local_llm: 0.6       # Minimum confidence for local LLM classification

# Database location
db_path: ~/.clipilot/clipilot.db

# Anonymous telemetry (opt-in only)
telemetry_enabled: false

# Colored terminal output
color_output: false
```

**To customize:**
```bash
# Edit the config file
nano ~/.clipilot/config.yaml

# Or use a different config file
clipilot --config=/path/to/config.yaml
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

# Build binary
go build -o clipilot ./cmd/clipilot

# Run locally
./clipilot --init --load=modules
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
