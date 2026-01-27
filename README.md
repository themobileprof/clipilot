# CLIPilot 🚀

**A lightweight, offline-first CLI assistant for Linux and mobile terminals**

CLIPilot is an intelligent command-line assistant designed for developers and operations teams. It helps you find and execute complex commands, manage workflows, and learn new tools—all without leaving your terminal.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/themobileprof/clipilot/workflows/Build%20and%20Test/badge.svg)](https://github.com/themobileprof/clipilot/actions)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

## ✨ Features

- **📱 Termux Native**: First-class support for Android/Termux environments.
- **🔌 Offline-First**: Works instantly without internet connectivity.
- **🎯 Smart Search**: Finds commands by description (e.g., "copy files recursively").
- **🔒 Safety-First**: Explicit confirmation for every command execution.
- **📦 Modular**: Expandable workflow modules for complex tasks.
- **⚡ Lightweight**: Optimized for low-memory devices (2-4GB RAM).

## 🚀 Quick Start

### Installation

**One-Line Install (Linux, macOS, Android/Termux)**:

```bash
curl -fsSL https://raw.githubusercontent.com/themobileprof/clipilot/main/install.sh | bash
```

This script automatically detects your platform, downloads the optimized binary, and sets up the environment.

### First Run

Start the interactive assistant:

```bash
clipilot
```

Or run a command directly:

```bash
clipilot "setup git"
```

## 📖 Usage

### Interactive Search
Simply type what you want to do:

```bash
$ clipilot
> copy directory recursively
```

CLIPilot will find the best command (e.g., `cp -r`) or a dedicated workflow module.

### Common Commands Catalog
CLIPilot includes a built-in catalog of commonly used tools like `git`, `tar`, `docker`, and more. Even if a tool isn't installed, CLIPilot can suggest it and show you how to install it.

### Module Registry
Access a library of community-driven automation modules:

```bash
# Sync new modules
clipilot sync

# Install a specific module
clipilot modules install nginx_setup
```

## 🛠️ Advanced & Technical

For details on **Architecture**, **Module Development**, and **Offline Intelligence**, please see our [Contributor Guide](CONTRIBUTING.md).

## 🗑️ Uninstalling

```bash
# Run the built-in uninstall command
clipilot
> uninstall
```

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) to get started with:
- Building from source
- Creating custom modules
- Understanding the internal architecture

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

---
**Made with ❤️ for developers working on restricted devices**
