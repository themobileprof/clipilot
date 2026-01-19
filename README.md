# CLIPilot 🚀

**A lightweight, offline-first CLI assistant for Linux and mobile terminals**

CLIPilot is an intelligent command-line assistant designed for developers and operations teams working on resource-constrained devices like Android/Termux environments. It provides deterministic, safe multi-step workflows for common dev/ops tasks through a modular architecture with hybrid intent detection.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![Build Status](https://github.com/themobileprof/clipilot/workflows/Build%20and%20Test/badge.svg)](https://github.com/themobileprof/clipilot/actions)
[![GitHub Stars](https://img.shields.io/github/stars/themobileprof/clipilot?style=social)](https://github.com/themobileprof/clipilot)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![GitHub Issues](https://img.shields.io/github/issues/themobileprof/clipilot)](https://github.com/themobileprof/clipilot/issues)
[![GitHub Pull Requests](https://img.shields.io/github/issues-pr/themobileprof/clipilot)](https://github.com/themobileprof/clipilot/pulls)

> **🌟 Open Source Project**: CLIPilot is free and open source under the MIT License. Community contributions welcome!

## ✨ Features

- **📱 Termux-First Design**: Optimized for Android/Termux as a first-class platform
- **🔌 Offline-First**: Works without internet connectivity using TF-IDF offline intelligence (no CGO, pure Go)
- **🎯 Hybrid Intent Detection**: TF-IDF + text normalization + intent extraction + category boost
- **📦 Modular Architecture**: Download and install task modules on demand
- **🔒 Safety-First**: All commands require explicit user confirmation before execution
- **💾 Lightweight**: Core binary <15MB, optimized for 2-4GB RAM devices (perfect for phones)
- **🗃️ SQLite Backend**: Fast local caching and state persistence (embedded, no installation needed)
- **🔄 Flow Engine**: Deterministic multi-step workflows with branching and validation
- **📱 Zero Dependencies**: SQLite is compiled into the binary - just download and run!
- **📚 Smart Command Discovery**: Automatically indexes system commands from man pages for natural language search
- **🚀 Cross-Platform**: Linux, macOS, Android/Termux - single Go binary, no CGO required

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
    │ SQLite   │       │  TF-IDF    │    │  Online   │
    │  Cache   │       │ Matcher    │    │    LLM    │
    └──────────┘       └────────────┘    └───────────┘
```

### Intent Detection Pipeline

1. **Layer 1 - Keyword/DB Search** (Fast, always available)
   - Tokenizes user input
   - Weighted keyword matching against patterns table
   - Scores candidates by relevance, tags, and popularity

2. **Layer 2 - Offline Intelligence** (Pure Go, no CGO)
   - TF-IDF similarity matching
   - Intent extraction (show, find, kill, monitor, etc.)
   - Category boosting (networking, process, filesystem, etc.)
   - 10-50ms query time

3. **Layer 3 - Online LLM Fallback** (Opt-in)
   - Triggered when offline layers fail
   - Provides explanations and handles novel queries

## 🚀 Quick Start

### Installation

#### 📱 Termux (Android) - Recommended for Mobile Devices

CLIPilot is designed with Termux as a **first-class platform**! Pure Go binaries work instantly:

```bash
# One-line install (downloads pre-built binary, no compilation!)
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/install.sh | bash
```

**What happens:**
- ✅ Detects your device architecture automatically
- ✅ Downloads pre-built binary (10-30 seconds!)
- ✅ No compilation needed (pure Go, no CGO)
- ✅ Installs to `$PREFIX/bin` 
- ✅ Copies **all 66+ modules** including Termux-optimized ones
- ✅ Sets up database and configuration

**First steps after install:**
```bash
clipilot run termux_setup                    # Configure Termux environment
clipilot run setup_development_environment   # Install dev tools
clipilot search phone                        # Find mobile-optimized modules
```

📚 **Full Termux guide:** [docs/TERMUX.md](docs/TERMUX.md)

#### One-Line Install (Linux/macOS)
```bash
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/install.sh | bash
```

This script will:
- Detect your platform (Linux/macOS, amd64/arm64/armv7)
- Download the appropriate pre-built binary from GitHub Release
- Install it to `/usr/local/bin` (or `$HOME/.local/bin` if no sudo)
- Download and install all available modules
- Initialize the database
- Set up configuration directory at `~/.clipilot`

**Supported Platforms:**
- **Termux/Android**: ARM64, ARM32 (dedicated android builds)
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

# Or for Termux/Android (use android binaries, no sudo needed)
curl -LO https://github.com/themobileprof/clipilot/releases/latest/download/clipilot-android-arm64.tar.gz
tar -xzf clipilot-android-arm64.tar.gz
chmod +x clipilot-android-arm64
mv clipilot-android-arm64 $PREFIX/bin/clipilot

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
# Index system commands (done automatically during install)
# Run manually if you install new software later
clipolot
> update-commands
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
  search <query>          - Search for modules and system commands
  run <module_id>         - Execute a specific module
  update-commands         - Index available system commands
  modules list            - List installed modules
  modules install <id>    - Download and install a module
  modules remove <id>     - Remove a module
  settings                - Configure CLIPilot
  logs                    - View execution history
  exit                    - Exit CLIPilot

**🤖 Smart Module Requests:**
When CLIPilot can't find a module for your query, it automatically submits your request to the community registry. This helps contributors know what modules are needed most!

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

# Search for modules and system commands
clipolot search "setup docker"
clipolot search "git"  # Also finds system commands like 'git'

# Update command index (after installing new software)
clipolot update-commands

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

## 📚 System Command Discovery

CLIPilot automatically indexes all available system commands from man pages **and suggests commonly-needed commands you don't have installed yet**, making command discovery effortless.

### How It Works

1. **Automatic Indexing**: During installation, CLIPilot runs `compgen -c` to discover all installed commands
2. **Description Extraction**: Uses `whatis` to fetch one-line descriptions from man pages
3. **Common Commands Catalog**: Maintains a curated list of ~70 commonly-needed commands with OS-specific installation instructions
4. **Smart Fallback**: When no strong match is found in installed commands, suggests from the catalog
5. **SQLite Storage**: Commands cached locally for instant search (~2500 commands typical)
6. **Priority-Based Ranking**: System commands prioritized over modules in search results

### Usage

```bash
# Index commands (done automatically during install)
clipilot update-commands

# Search for installed commands
clipilot search git        # Finds git command with description
clipilot search "list files"  # Finds ls, dir, and related commands

# Search for uninstalled commands - get installation suggestions!
clipilot search ripgrep    # Shows: ripgrep (not installed) - sudo apt install ripgrep
clipilot search bat        # Shows: bat (not installed) - pkg install bat (on Termux)

# Commands appear in search results with scores
> search grep
Found 3 results:
1. grep (ID: cmd:grep)
   print lines that match patterns
   Score: 3.00 | Tags: command

> search ripgrep
Found 2 results:
1. ripgrep (not installed) (ID: common:ripgrep)
   Fast recursive grep alternative - sudo apt install ripgrep
   Score: 1.40 | Tags: installable, file-management
2. modern_cli_tools_install (ID: org.themobileprof.modern_cli_tools_install)
   Install modern CLI tools (bat, eza, fd, ripgrep, fzf, tldr, htop)...
   Score: 1.00 | Tags: installer, composite
```

### Common Commands Catalog

The catalog includes ~70 commonly-needed commands across 10 categories:

- **Development**: git, python3, node, npm, go, make, gcc
- **Modern CLI**: bat, eza, fd, ripgrep, fzf, tldr, htop, btop
- **Networking**: curl, wget, ssh, rsync, nmap
- **Databases**: psql, mysql, redis-cli, mongosh, sqlite3
- **File Management**: tar, zip, grep, sed, awk, jq
- **System**: htop, btop, iftop
- **Editors**: vim, nano, emacs
- **Containers**: docker, kubectl
- **Cloud**: aws, gcloud, az
- **Misc**: tmux, screen, tree

Each command includes **OS-specific installation instructions** for:
- **apt** (Debian/Ubuntu)
- **pkg** (Termux/Android)
- **dnf** (Fedora/RHEL)
- **brew** (macOS)
- **pacman** (Arch Linux)

CLIPilot automatically detects your OS and shows the correct command!

📖 **Learn more**: See [docs/COMMON_COMMANDS.md](docs/COMMON_COMMANDS.md) for the complete catalog.

### When to Re-index

Run `update-commands` after:
- Installing new software packages
- Adding custom scripts to PATH
- Setting up new development tools

**Note**: Man pages are automatically installed during CLIPilot setup on both Linux and Termux.

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

## 🌐 Module Registry (Self-Hosted)

Host your own module registry for sharing CLIPilot modules within your organization.

### Quick Start

```bash
# Pull and run the Docker image
docker pull themobileprof/clipilot-registry:latest

docker run -d \
  --name clipilot-registry \
  -p 8082:8080 \
  -v registry-data:/app/data \
  -e ADMIN_USER=admin \
  -e ADMIN_PASSWORD=your_password \
  -e BASE_URL=http://localhost:8082 \
  themobileprof/clipilot-registry:latest

# Note: Using port 8082 to avoid conflicts. Change to any available port.
```

### Build Your Own

```bash
# Build the image
docker build -f Dockerfile.registry -t your-username/clipilot-registry .

# Push to Docker Hub
docker login
docker push your-username/clipilot-registry:latest

# Pull and run anywhere
docker pull your-username/clipilot-registry:latest
docker run -d -p 8080:8080 your-username/clipilot-registry:latest
```

### Features
- 🌐 Web UI for browsing modules
- 📦 REST API for module distribution
- 🔐 Basic authentication for uploads
- 💾 SQLite database (no external DB needed)
- 🐳 Multi-arch support (amd64/arm64)

**Full documentation:** [docs/REGISTRY_DOCKER.md](docs/REGISTRY_DOCKER.md)

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
- No CGO required - pure Go implementation

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


## 🌐 Community Module Registry

CLIPilot includes a web-based module registry where users can share and discover community-created modules.

### Official Public Registry

**🌍 https://clipilot.themobileprof.com**

The official public registry is pre-configured in CLIPilot. You can:
- Browse available modules in the web interface
- Login with GitHub to contribute modules
- Install modules via CLI

```bash
# Sync with registry (fetches available modules)
clipilot sync

# Index system commands (indexes all available commands from man pages)
clipolot update-commands

# Search for modules and system commands
clipilot search <keyword>

# Install modules from registry
clipilot modules install <module_id>

# Check registry configuration
clipilot settings get registry_url
```

### Using the Registry

By default, CLIPilot is configured to use the official registry at `https://clipilot.themobileprof.com`. You can:

```bash
# List available modules from registry
clipilot modules list

# Install a module
clipilot modules install where_is_web_root

# Browse the web interface
# Visit https://clipilot.themobileprof.com
```

To configure a different registry:
```bash
# Set custom registry URL
clipilot settings set registry_url https://your-registry.com

# Or use environment variable
export REGISTRY_URL=https://your-registry.com
```

### Running Your Own Registry

```bash
# Build the registry server
go build -o registry ./cmd/registry

# Build the enhancement CLI tool
go build -o bin/enhance ./cmd/enhance

# Run with admin credentials and custom port
PORT=8082 ADMIN_PASSWORD=your_secure_password GEMINI_API_KEY=your_key ./registry

# Registry auto-discovers commands on first startup (< 50 enhanced commands)
# Then enhance them: ./bin/enhance --auto --limit=100

# Access at http://localhost:8082 (or your chosen port)
# For production, set BASE_URL=https://your-domain.com
```

See [docs/REGISTRY.md](docs/REGISTRY.md) for full documentation on:
- Setting up the registry server
- Server bootstrap (automatic command discovery)
- Uploading modules
- Using ChatGPT to generate modules
- API documentation
- Deployment guides

**New:** Registry automatically discovers and submits its own commands on first startup. See [docs/SERVER_BOOTSTRAP.md](docs/SERVER_BOOTSTRAP.md) for details.

### Uploading Modules to Production Registry

If you're managing the production registry, you can bulk-upload all modules using the provided script:

```bash
# Upload all modules from the modules/ directory to the registry
./scripts/upload-modules.sh https://clipilot.themobileprof.com admin YOUR_PASSWORD
```

The script will:
- ✅ Authenticate with the registry admin credentials
- ✅ Upload all YAML module files from `modules/` directory
- ✅ Show progress for each module upload
- ✅ Provide a summary report of successes and failures

**Note**: Some modules require tags for validation. Modules without tags will be rejected with an error message. Update them to include at least one tag before uploading.

After uploading, users can sync their local database with:
```bash
clipilot sync
```

## 🗑️ Uninstalling CLIPilot

If you need to remove CLIPilot from your system, you have two convenient options:

### Built-in Uninstall Command (Easiest)

Simply run the uninstall command from within CLIPilot:

```bash
clipilot
> uninstall
```

This will:
- 🔍 Automatically detect your installation location
- 📋 Show you exactly what will be removed
- ⚠️  Ask for confirmation before deleting anything
- 🗑️  Remove the binary and all data
- ✅ Clean exit and self-removal

### Uninstall Script

Alternatively, use our standalone uninstall script:

```bash
# Download and run the uninstall script
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/uninstall.sh | bash

# Or if you have the repository cloned:
./uninstall.sh
```

**What gets removed:**
- CLIPilot binary (`/usr/local/bin/clipilot` or `$PREFIX/bin/clipilot`)
- Configuration directory (`~/.clipilot/`)
- All installed modules
- Command index and search history
- Settings and execution logs

**What does NOT get removed:**
- Packages you installed using CLIPilot modules
- System commands indexed by CLIPilot (they're still on your system)
- Files created using CLIPilot
- The source code (if you cloned the repository)

### Manual Uninstall

If you prefer to uninstall manually or the script doesn't work:

**Linux/macOS:**
```bash
# Remove binary (choose the location where it was installed)
sudo rm /usr/local/bin/clipilot
# OR
rm $HOME/.local/bin/clipilot

# Remove data directory
rm -rf ~/.clipilot

# Optional: Remove cloned source
rm -rf ~/clipilot  # or wherever you cloned it
```

**Termux (Android):**
```bash
# Remove binary
rm $PREFIX/bin/clipilot

# Remove data directory
rm -rf ~/.clipilot

# Optional: Remove cloned source
rm -rf ~/clipilot
```

### Reinstalling Later

To reinstall CLIPilot after uninstalling:
```bash
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/install.sh | bash
```

Or visit: https://github.com/themobileprof/clipilot

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas for Contribution

- **New Modules**: Add support for more tools and workflows - share them on the registry!
- **OS Support**: Expand compatibility (Alpine, Fedora, etc.)
- **Performance**: Optimize for low-memory devices
- **Offline Intelligence Tuning**: Improve TF-IDF weighting and synonym expansion
- **Documentation**: Improve guides and examples
- **Registry Features**: Enhance the module registry with search, ratings, categories

## 🤝 Contributing

We welcome contributions from the community! CLIPilot is open source and thrives on community involvement.

**Ways to Contribute:**
- 🐛 Report bugs and suggest features
- 💻 Submit pull requests for improvements
- 📝 Improve documentation
- 📦 Create and share YAML modules
- ⭐ Star the project on GitHub
- 🗣️ Spread the word

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on how to contribute.

## 📝 License

This project is licensed under the **MIT License** - see [LICENSE](LICENSE) file for details.

**CLIPilot is free and open source software.** You are free to use, modify, and distribute it according to the terms of the MIT License.

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
