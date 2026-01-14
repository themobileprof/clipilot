# Common Commands Catalog

## Overview

CLIPilot includes a **categorized catalog of commonly-needed commands** that helps suggest installations when users search for tools they don't have installed.

## How It Works

When you search for a command:

1. **First**: CLIPilot checks **installed commands** on your system (via `compgen -c`)
2. **Second**: CLIPilot searches **installed modules** in the database
3. **Fallback**: If no strong match is found (score < 1.5), CLIPilot searches the **common commands catalog**

If a common command matches your query, CLIPilot will suggest it with:
- ✅ Command name with `(not installed)` label
- 📝 Description of what the command does
- 📦 **OS-specific installation instruction**

## Example Usage

```bash
> search ripgrep

Found 2 module(s):

1. ripgrep (not installed) (ID: common:ripgrep)
   Fast recursive grep alternative - sudo apt install ripgrep
   Score: 1.40 | Tags: installable, file-management

2. modern_cli_tools_install (ID: org.themobileprof.modern_cli_tools_install)
   Install modern CLI tools (bat, eza, fd, ripgrep, fzf, tldr, htop)...
   Score: 1.00 | Tags: installer, composite, system-ops, development
```

## Catalog Contents

The catalog includes ~70 commonly-needed commands across 10 categories:

### Development Tools (Priority: 75-95)
- `git` - Version control system
- `python3` - Python interpreter
- `node` / `npm` - Node.js runtime and package manager
- `go` - Go programming language
- `make` - Build automation tool
- `gcc` / `g++` - C/C++ compilers

### Modern CLI Tools (Priority: 65-75)
- `bat` - Cat clone with syntax highlighting
- `eza` - Modern ls replacement
- `fd` - Fast find alternative
- `ripgrep` (`rg`) - Fast recursive grep
- `fzf` - Fuzzy finder
- `tldr` - Simplified man pages
- `btop` / `htop` - Interactive process viewers

### Networking (Priority: 85-90)
- `curl` / `wget` - File download tools
- `ssh` - Secure shell
- `rsync` - File synchronization
- `nmap` - Network scanner
- `netcat` (`nc`) - Network utility

### Database Clients (Priority: 60-75)
- `psql` - PostgreSQL client
- `mysql` - MySQL client
- `redis-cli` - Redis client
- `mongosh` - MongoDB shell
- `sqlite3` - SQLite client

### File Management (Priority: 80-90)
- `tar` - Archive tool
- `zip` / `unzip` - Compression tools
- `gzip` / `gunzip` - Gzip compression
- `grep` / `sed` / `awk` - Text processing
- `tree` - Directory tree viewer
- `jq` - JSON processor

### System Monitoring (Priority: 75-80)
- `htop` - Interactive process viewer
- `btop` - Modern process viewer
- `iftop` - Network bandwidth monitor
- `iotop` - I/O monitor
- `dstat` - System resource stats

### Editors (Priority: 80-85)
- `vim` - Vi improved editor
- `nano` - Simple text editor
- `emacs` - GNU Emacs editor

### Containers & Cloud (Priority: 60-85)
- `docker` - Container platform
- `kubectl` - Kubernetes CLI
- `aws` - AWS CLI
- `gcloud` - Google Cloud SDK
- `az` - Azure CLI

### Terminal Multiplexers (Priority: 70-75)
- `tmux` - Terminal multiplexer
- `screen` - Terminal multiplexer

### Miscellaneous
- `man` - Manual pages
- `apropos` - Search man pages
- And more...

## OS-Specific Installation

The catalog includes package names for **5 different package managers**:

- **apt** (Debian/Ubuntu): `sudo apt install <package>`
- **pkg** (Termux/Android): `pkg install <package>`
- **dnf** (Fedora/RHEL): `sudo dnf install <package>`
- **brew** (macOS): `brew install <package>`
- **pacman** (Arch Linux): `sudo pacman -S <package>`

CLIPilot automatically detects your OS and shows the correct installation command!

### Termux Example
```bash
> search bat
1. bat (not installed) (ID: common:bat)
   Cat clone with syntax highlighting - pkg install bat
```

### Ubuntu Example
```bash
> search bat
1. bat (not installed) (ID: common:bat)
   Cat clone with syntax highlighting - sudo apt install bat
```

### macOS Example
```bash
> search bat
1. bat (not installed) (ID: common:bat)
   Cat clone with syntax highlighting - brew install bat
```

## Priority System

Commands are ranked by priority (0-100):

- **95**: Essential (git)
- **90**: Very common (curl, ssh, tar, grep, python3)
- **85**: Common (docker, vim, wget, node, npm)
- **80**: Useful (make, htop, nano, jq)
- **75**: Specialized (gcc, htop variants, psql)
- **70**: Niche (tmux, screen)
- **60-65**: Advanced tools (kubectl, cloud CLIs, mongosh)

Higher priority commands appear first in search results.

## Loading the Catalog

The catalog is automatically loaded when you run:

```bash
clipilot update-commands
```

Or during installation via `install.sh`.

**Output:**
```
=== Indexing System Commands ===
🔍 Discovering available commands...
📦 Found 2514 commands, indexing descriptions...
...
✅ Indexed 2514 commands successfully!

Loading common commands catalog...
✓ Loaded 49 common commands into catalog
✓ Common commands catalog loaded

💡 Indexed 2514 commands with descriptions from man pages
   You can now search for any command using natural language!
```

## Data Source

The catalog is stored in:
- **YAML file**: `data/common_commands.yaml`
- **Database table**: `common_commands` (SQLite)

Each command entry includes:
- `name`: Command name
- `description`: What it does
- `category`: Classification (development, networking, etc.)
- `keywords`: Searchable keywords
- `apt_package`, `pkg_package`, `dnf_package`, `brew_package`, `arch_package`: Package names for each OS
- `alternative_to`: What it replaces (e.g., `bat` is alternative to `cat`)
- `homepage`: Project website
- `priority`: Ranking (0-100)

## Contributing New Commands

To add a command to the catalog:

1. Edit `data/common_commands.yaml`
2. Add your command following the existing format:
   ```yaml
   - name: your_command
     description: What it does
     category: development  # or networking, file-management, etc.
     keywords: keyword1, keyword2, keyword3
     apt_package: package-name
     pkg_package: package-name
     dnf_package: package-name
     brew_package: package-name
     arch_package: package-name
     alternative_to: existing_command  # optional
     homepage: https://example.com      # optional
     priority: 75                        # 0-100
   ```
3. Test with `clipilot update-commands`
4. Submit a pull request

## Architecture

### Database Schema

```sql
CREATE TABLE common_commands (
    name TEXT PRIMARY KEY NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    keywords TEXT,
    apt_package TEXT,
    pkg_package TEXT,
    dnf_package TEXT,
    brew_package TEXT,
    arch_package TEXT,
    alternative_to TEXT,
    homepage TEXT,
    priority INTEGER DEFAULT 50,
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX idx_common_commands_name ON common_commands(name);
CREATE INDEX idx_common_commands_category ON common_commands(category);
CREATE INDEX idx_common_commands_priority ON common_commands(priority DESC);
```

### Code Flow

1. **Loading**: `internal/commands/indexer.go` → `LoadCommonCommands()`
   - Reads `data/common_commands.yaml`
   - Parses with `gopkg.in/yaml.v3`
   - Inserts into `common_commands` table with conflict resolution

2. **Searching**: `internal/intent/keyword.go` → `searchCommonCommands()`
   - Queries `common_commands` table with LIKE patterns
   - Matches against: name, description, keywords
   - Orders by priority DESC
   - Calls `getInstallCommand()` for OS detection

3. **OS Detection**: `internal/intent/keyword.go` → `getInstallCommand()`
   - Checks `TERMUX_VERSION` env var
   - Uses `exec.LookPath()` to detect package managers
   - Returns appropriate install command

4. **Integration**: `internal/intent/keyword.go` → `keywordSearch()`
   - Scores installed commands (weight 3.0)
   - Scores modules (weight 1.0-2.0)
   - If score < 1.5, searches common commands
   - Appends results with `common:` prefix

## Benefits

✅ **Discoverability**: Users find tools they didn't know existed  
✅ **No guesswork**: Exact installation command provided  
✅ **Cross-platform**: Works on Linux, macOS, and Android (Termux)  
✅ **Offline-first**: Catalog embedded with CLIPilot  
✅ **Smart fallback**: Only suggests when no strong match found  
✅ **Priority-based**: Most useful commands appear first  

## Limitations

- Catalog is curated (not exhaustive like package repositories)
- Package names may differ slightly across distros
- Requires manual updates to add new commands
- No automatic version checking

## Future Enhancements

- [ ] Auto-install command on user confirmation
- [ ] Check if command is already installed before suggesting
- [ ] Add more commands to catalog (currently ~70)
- [ ] Category-based filtering (`search database tools`)
- [ ] Integration with package manager search (e.g., `apt-cache search`)
- [ ] User-contributed commands via registry

---

**Questions?** See [CONTRIBUTING.md](../CONTRIBUTING.md) or open an issue.
