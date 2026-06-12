# CLIPilot Registry 🚀

**A module registry server for CLI automation workflows**

CLIPilot Registry is the backend server that powers [Clio](https://github.com/themobileprof/clio), a lightweight CLI assistant for developers and operations teams. The registry hosts, validates, and distributes YAML workflow modules that Clio executes on client systems.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/themobileprof/clipilot/workflows/CI/CD/badge.svg)](https://github.com/themobileprof/clipilot/actions)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

## 🏗️ Architecture

**CLIPilot** is the **server** (this repository) - a web application for browsing, uploading, and managing workflow modules.

**Clio** is the **client** - an offline-first CLI assistant that turns plain English into shell commands, runs **setup wizards** (Termux, Vim, Git, dev tools, databases), and syncs **automation modules** from this registry.

```
┌─────────────────────┐          ┌──────────────────────┐
│   Clio Client       │  HTTPS   │  CLIPilot Registry   │
│   (User's Machine)  │ <──────> │  (Server)            │
│                     │  Sync    │                      │
│  - Intent Detection │          │  - Module Storage    │
│  - Module Execution │          │  - Web UI            │
│  - Offline Ready    │          │  - API Endpoints     │
└─────────────────────┘          └──────────────────────┘
```

## ✨ Features

- **📦 Module Registry**: Centralized storage for workflow modules
- **🔍 Semantic Search**: Catalog + optional Gemini for Clio remote fallback
- **🌐 Web UI**: Browse, search, and upload modules via browser
- **🔐 Authentication**: GitHub OAuth + session-based auth
- **📊 Analytics**: Track module downloads and popularity
- **🚀 Delta Sync**: Efficient incremental updates for clients
- **🔑 API Keys**: Secure CI/CD integration for automated uploads
- **📱 Clio Install**: Hosts the Clio installation script at `/clio`
- **⭐ Setup Wizards**: First-class install/configure modules (`termux_setup`, `vim_setup`, `git_setup`, `devtools_setup`, `database_setup`)

## 🚀 Quick Start

### For Clio Users

If you want to **use** CLI automation, install the Clio client:

```bash
curl -fsSL clipilot.themobileprof.com/clio | sh
```

See the [Clio repository](https://github.com/themobileprof/clio) for documentation.

### For Registry Administrators

If you want to **host your own registry server**, continue below.

## 🚀 Deployment

### Local Development

1. Clone the repository:
```bash
git clone https://github.com/themobileprof/clipilot.git
cd clipilot
```

2. Create a `.env` file from `.env.example` and set at least `ADMIN_PASSWORD`.

3. Build and run:
```bash
go build -o clipilot-server ./cmd/registry
./clipilot-server
```

The registry will be available at `http://localhost:8080` (or the port set in `PORT`).

### Production

Production deploys run automatically on push to `main` via GitHub Actions (`.github/workflows/ci.yml`).

Required GitHub secrets:
- `SSH_HOST`, `SSH_USERNAME`, `SSH_PRIVATE_KEY`
- `ENV_FILE` (production environment variables; paths are set automatically under `$HOME`)

`SSH_USERNAME` should be your existing server user (no special deploy account needed). That user needs **no sudo**. One-time root setup:

```bash
sudo loginctl enable-linger YOUR_USERNAME
```

Deploy layout (under that user's home directory):
- `~/clipilot-registry/` — binary, static assets, env file
- `~/clipilot-data/` — SQLite database and uploads

For manual server deployment (SSH in as that user, then):
```bash
./scripts/deploy.sh
```

See [docs/REGISTRY.md](docs/REGISTRY.md) and [scripts/ADMIN_SETUP.md](scripts/ADMIN_SETUP.md) for configuration details.

## 🔌 API Endpoints

### Public Endpoints

- `GET /` - Home page
- `GET /modules` - Browse modules (web UI)
- `GET /clio` - Download Clio installation script
- `GET /health` - Health check endpoint
- `GET /api/v1/modules` - List modules (JSON)
- `GET /api/v1/modules/:id` - Get module metadata
- `GET /api/v1/modules/:id/download` - Download module YAML
- `GET /api/v1/modules/changed?since=<timestamp>` - Delta sync

### Authenticated Endpoints

- `POST /upload` - Upload a module (web UI)
- `POST /api/upload` - Upload a module (API)
- `GET /my-modules` - View your uploaded modules

### Admin Endpoints (Require API Key)

- `POST /api/install-script/upload` - Upload Clio install script (CI/CD)
- `GET /api/install-scripts` - List install script versions
- `POST /api/install-scripts/:id/activate` - Activate a script version

## 📖 Module Development

### Creating a Module

Modules are YAML files that define multi-step workflows. Example:

```yaml
name: git_setup
id: org.example.git_setup
version: 1.0.0
description: Install and configure Git
tags: [git, setup, devtools]
requires: [check_sudo]
provides: [git_installed]

flows:
  main:
    start: check_git
    steps:
      check_git:
        type: action
        message: "Checking if Git is installed..."
        command: command -v git && git --version || echo "missing"
        next: install_git
      
      install_git:
        type: action
        message: "Installing Git..."
        command: |
          sudo apt-get update
          sudo apt-get install -y git
        next: end
      
      end:
        type: terminal
        message: "Git installed successfully!"
```

### Uploading to Registry

**Via Web UI:**
1. Log in at `https://your-registry.com/login`
2. Navigate to `/upload`
3. Select your YAML file and submit

**Via API:**
```bash
curl -X POST https://your-registry.com/api/upload \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -F "file=@git_setup.yaml"
```

## 🔐 Security

### Authentication Methods

1. **Session-based** (Web UI): Username/password or GitHub OAuth
2. **API Key** (CI/CD): Bearer token authentication

### Best Practices

- Use HTTPS in production (configure reverse proxy)
- Rotate API keys regularly
- Enable GitHub OAuth for contributor authentication
- Set strong admin passwords (16+ characters)
- Keep the server updated

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup
- Module specification
- Testing guidelines
- Code conventions

For Clio client changes, visit the [Clio repository](https://github.com/themobileprof/clio).

## 📚 Documentation

- [CLIO_API_REQUIREMENTS.md](docs/CLIO_API_REQUIREMENTS.md) - API specification for clients
- [CLIO_MIGRATION_GUIDE.md](docs/CLIO_MIGRATION_GUIDE.md) - Migration from CLIPilot to Clio
- [TERMUX.md](docs/TERMUX.md) - Android/Termux deployment guide
- [ADMIN_SETUP.md](scripts/ADMIN_SETUP.md) - Admin user creation guide
- [TESTING.md](TESTING.md) - Testing procedures

## 🐛 Troubleshooting

### Module Sync Fails

Check that the registry is accessible:
```bash
curl https://your-registry.com/health
```

### Admin Login Issues

Reset admin password:
```bash
DB_PATH=./data/registry.db ./scripts/create-admin.sh
```

### API Key Not Working

Verify the key is active:
```bash
sqlite3 ./data/registry.db "SELECT * FROM api_keys WHERE revoked = 0"
```

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

---

**Part of the CLIPilot ecosystem:**
- [CLIPilot Registry](https://github.com/themobileprof/clipilot) (this repo) - Server
- [Clio](https://github.com/themobileprof/clio) - CLI Client

Made with ❤️ for developers automating their workflows
