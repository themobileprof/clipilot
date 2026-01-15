# 📱 CLIPilot on Termux (Android)

**CLIPilot is designed with Termux as a first-class platform!** This guide covers everything you need to know about running CLIPilot on your Android device.

## 🌟 Why CLIPilot + Termux?

Transform your Android phone into a powerful portable development environment:

- 🚀 **Complete Dev Environment**: Set up Git, Python, Node.js, databases - all on your phone
- 🛠️ **Modern CLI Tools**: Install bat, eza, ripgrep, fzf, and 60+ other utilities
- ☁️ **Cloud Management**: Manage AWS, GCP, Azure infrastructure from anywhere
- 🗄️ **Database Clients**: PostgreSQL, MySQL, MongoDB, Redis - all in your pocket
- 🔒 **Network Tools**: nmap, tcpdump, iperf3 for network diagnostics
- 📦 **66+ Modules**: All modules are Termux-tested and optimized

## 🎯 Quick Start

### Prerequisites

1. **Install Termux** from F-Droid (recommended) or GitHub
   - 📥 F-Droid: https://f-droid.org/en/packages/com.termux/
   - 📥 GitHub: https://github.com/termux/termux-app/releases
   - ⚠️ **Do NOT use Google Play version** (outdated and unsupported)

2. **Grant Storage Permission** (recommended, for accessing Android files)
   ```bash
   termux-setup-storage
   ```

### 🚀 One-Line Installation

**Pure Go binary - downloads and runs instantly, no compilation needed!**

```bash
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/install.sh | bash
```

**What happens:**
- ✅ Detects your device architecture (ARM64/ARM32/x86_64)
- ✅ Downloads pre-built binary (no compilation!)
- ✅ Installs to `$PREFIX/bin` (already in PATH)
- ✅ Installs man pages (`man` and `man-pages` packages)
- ✅ Indexes all system commands for smart search
- ✅ Copies all 66+ modules including Termux-optimized ones
- ✅ Initializes database and configuration

**Installation time:** 10-30 seconds (just download, no build!)

**First steps after installation:**
```bash
# Configure your Termux environment
clipilot run termux_setup

# Search for any command or module
clipilot search git
clipilot search "copy files"

# Install development tools
clipilot run setup_development_environment

# Find mobile-specific modules
clipilot search phone
clipilot search android
clipilot search mobile
```

### Manual Build (Optional)

Only if you want to build from source (rarely needed):

```bash
# Install build tools
pkg update && pkg upgrade
pkg install golang git clang

# Clone and build
git clone https://github.com/themobileprof/clipilot.git
cd clipilot
go build -o clipilot ./cmd/clipilot

# Install
mv clipilot $PREFIX/bin/
chmod +x $PREFIX/bin/clipilot

# Initialize
clipilot --init --load modules
```

## Troubleshooting

### Error: "cannot execute: required file not found"

**This error should NOT happen anymore!** CLIPilot now uses pure Go (no CGO), so binaries work on all Termux devices.

**If you still see this:**
1. Make sure you're using the latest installer:
```bash
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/install.sh | bash
```

2. Verify the binary downloaded correctly:
```bash
file $PREFIX/bin/clipilot
# Should show: ELF 64-bit LSB executable, ARM aarch64 (or similar)
```

### Error: "go: command not found"

**Cause:** Go is not installed.

**Solution:**
```bash
pkg update
pkg install golang
```

### Error: Build fails (if building from source)

**Cause:** Missing Go compiler.

**Solution:**
```bash
pkg install golang git
```

Note: CGO is not needed anymore - CLIPilot uses pure Go!

### Error: "pkg: command not found"

**Cause:** Not running in Termux or Termux is not properly installed.

**Solution:** 
- Install Termux from F-Droid (not Google Play)
- Make sure you're running commands inside the Termux app

### Error: Out of storage space

**Cause:** Go toolchain and build requires ~500MB.

**Solution:**
```bash
# Check available space
df -h $PREFIX

# Clean package cache
pkg clean

# Remove unnecessary packages
pkg uninstall <package_name>
```

### Build is very slow

This is normal on mobile devices. Building takes 2-5 minutes depending on your phone's CPU.

**Tips:**
- Close other apps to free RAM
- Keep phone plugged in during build
- Consider using a tablet or newer device for faster builds

## Architecture Detection

CLIPilot automatically detects your device architecture:

```bash
# Check your architecture
uname -m
```

**Common architectures:**
- `aarch64` - ARM64 (64-bit) - Best support
- `armv7l` / `armv8l` - ARM32 (32-bit) - Limited cloud CLI support
- `x86_64` - x86 64-bit (rare on phones)
- `i686` - x86 32-bit (very rare)

### ARM64 vs ARM32 Tool Support

**ARM64 (64-bit)** - Full support:
- ✅ All development tools
- ✅ Modern CLI tools
- ✅ Database clients
- ✅ Network/security tools
- ✅ Cloud CLI tools (AWS, GCP, Azure, Terraform, kubectl)

**ARM32 (32-bit)** - Limited support:
- ✅ Development tools (Python, Node.js via source)
- ✅ Modern CLI tools
- ✅ Database clients (limited)
- ✅ Network/security tools
- ⚠️ Cloud CLI tools (only Python-based: AWS CLI, Azure CLI)
- ❌ Terraform, kubectl (no official ARM32 binaries)

## Recommended Setup Workflow

After installing CLIPilot, use these modules to set up your Termux environment:

```bash
# 1. Initial Termux setup
clipilot run termux_setup

# 2. Install Zsh with Oh My Zsh
clipilot run zsh_setup

# 3. Set up complete development environment (wizard)
clipilot run setup_development_environment
```

The `setup_development_environment` wizard lets you choose:
- Development Tools (Python, Node.js, Go, Make, C/C++)
- Modern CLI Tools (bat, eza, fd, ripgrep, fzf, tldr, htop, btop)
- Database Clients (PostgreSQL, MySQL, MongoDB, Redis, SQLite)
- Network & Security Tools (nmap, netcat, tcpdump, iftop, iperf3)
- Cloud CLI Tools (AWS CLI, gcloud, Azure CLI, Terraform, kubectl)

Each category has individual tool selection, so you only install what you need!

## System Command Discovery

CLIPilot automatically indexes all available Termux commands during installation:

```bash
# Commands are indexed automatically, but you can re-index anytime
clipilot update-commands

# Search for any command
clipilot search pkg       # Find package manager commands
clipilot search termux    # Find Termux-specific commands
clipilot search ssh       # Find SSH-related commands

# After installing new packages, update the index
pkg install git
clipilot update-commands  # Picks up newly installed git commands
```

**What gets indexed:**
- All commands in `$PREFIX/bin` (~200-500 on typical Termux)
- Built-in shell commands
- Custom scripts you add to PATH
- Descriptions from man pages (automatically installed)

**Smart search examples:**
```bash
> search "copy files"
  Finds: cp, rsync, scp commands with descriptions

> search git
  Finds: git, git-shell, git-upload-pack, etc. with man page descriptions
```

## Termux-Specific Features

CLIPilot includes special handling for Termux:

### Package Manager Detection
Automatically uses `pkg` instead of `apt`/`dnf`/`yum`

### Storage Paths
Uses `$PREFIX` for installation directories:
- Binaries: `$PREFIX/bin`
- Libraries: `$PREFIX/lib`
- Config: `$PREFIX/etc`

### Android Integration
Access Android storage from Termux:
```bash
~/storage/dcim       # Camera photos
~/storage/downloads  # Downloads folder
~/storage/shared     # Shared storage
```

### SSH Setup
The `termux_setup` module configures SSH on port 8022:
```bash
# Start SSH server
sshd

# Connect from PC
ssh -p 8022 user@phone-ip
```

## Performance Tips

1. **Use Zsh with plugins** - Run `clipilot run zsh_setup` for better shell experience

2. **Install modern CLI tools** - Faster alternatives to classic Unix tools:
   ```bash
   clipilot run modern_cli_tools_install
   ```

3. **Close background apps** - Free RAM for faster CLIPilot execution

4. **Use WiFi** - When syncing modules or installing packages

5. **Keep screen on** - Prevents process interruption during long operations

## Common Use Cases

### Web Development on Phone
```bash
clipilot run nodejs_setup          # Install Node.js
clipilot run git_setup              # Configure Git
clipilot run vim_dev_setup          # Set up Vim for coding
```

### Database Administration
```bash
clipilot run database_clients_install    # Install db clients
# Then use: psql, mysql, mongosh, redis-cli
```

### Cloud Infrastructure Management
```bash
clipilot run cloud_cli_tools_install    # Install AWS/GCP/Azure CLIs
# Configure: aws configure, gcloud init, az login
```

### Network Diagnostics
```bash
clipilot run network_security_tools_install
# Then use: nmap, tcpdump, iftop, iperf3
```

## Backing Up Your Setup

```bash
# Backup CLIPilot data
tar -czf clipilot-backup.tar.gz ~/.clipilot

# Restore on another device
tar -xzf clipilot-backup.tar.gz -C ~
```

## Updating CLIPilot

```bash
cd ~/clipilot  # or wherever you cloned it
git pull
go build -o clipilot ./cmd/clipilot
mv clipilot $PREFIX/bin/
```

## Uninstalling

### Built-in Uninstall Command (Easiest)

The simplest way to uninstall is using the built-in command:

```bash
clipilot
> uninstall
```

CLIPilot will:
- 🔍 Automatically detect your Termux installation
- 📋 Show you what will be removed
- ⚠️  Ask for confirmation before deleting
- 🗑️  Remove the binary from `$PREFIX/bin`
- 🗑️  Remove all data from `~/.clipilot`
- ✅ Self-remove cleanly

### Uninstall Script (Alternative)

You can also use the standalone script:

```bash
# Download and run the uninstall script
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/uninstall.sh | bash

# Or if you have the repository cloned:
./uninstall.sh
```

### Manual Uninstall

If you prefer to uninstall manually:

```bash
# Remove binary
rm $PREFIX/bin/clipilot

# Remove data (optional - contains all your modules and settings)
rm -rf ~/.clipilot

# Remove source (if you built from source)
rm -rf ~/clipilot
```

**Note**: This only removes CLIPilot itself. Packages you installed using CLIPilot modules (like git, python3, etc.) will remain on your device. To remove those, use `pkg uninstall <package-name>`.

## Getting Help

If you encounter issues:

1. **Check Termux version**: `termux-info`
2. **Update packages**: `pkg update && pkg upgrade`
3. **Check logs**: `clipilot logs`
4. **Search GitHub Issues**: https://github.com/themobileprof/clipilot/issues
5. **Ask for help**: Open a new issue with `termux-info` output

## Resources

- **Termux Wiki**: https://wiki.termux.com
- **CLIPilot Docs**: https://github.com/themobileprof/clipilot
- **Module List**: Use `clipilot modules list` or see docs/MODULE_TAGS.md
- **Termux Community**: https://reddit.com/r/termux

## Contributing

Found a Termux-specific bug? Have ideas for Termux modules? Contributions welcome!

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

---

**Happy Hacking on Mobile! 📱🚀**
