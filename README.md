# CLIPilot Registry 🚀

**A module registry server for CLI automation workflows**

CLIPilot Registry is the backend server that powers [Clio](https://github.com/themobileprof/clio), a lightweight CLI assistant for developers and operations teams. The registry hosts, validates, and distributes YAML workflow modules that Clio executes on client systems.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/themobileprof/clipilot/workflows/Build%20and%20Test/badge.svg)](https://github.com/themobileprof/clipilot/actions)
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

## 🐳 Deployment (Docker)

### Quick Deploy with Docker Compose

1. Clone the repository:
```bash
git clone https://github.com/themobileprof/clipilot.git
cd clipilot
```

2. Create a `.env` file:
```bash
cat > .env << EOF
ADMIN_USER=admin
ADMIN_PASSWORD=your-secure-password
BASE_URL=https://your-domain.com
GITHUB_CLIENT_ID=your-github-oauth-client-id
GITHUB_CLIENT_SECRET=your-github-oauth-client-secret
GEMINI_API_KEY=your-gemini-api-key  # Optional, for semantic search
EOF
```

3. Start the registry:
```bash
docker compose up -d
```

4. Create an admin user and API key:
```bash
# Option 1: Environment variables (automatic)
# Admin user is created from ADMIN_USER/ADMIN_PASSWORD in .env

# Option 2: Manual creation with script
./scripts/create-admin.sh
```

The registry will be available at `http://localhost:8082`

### Manual Deployment

#### Prerequisites
- Go 1.24+
- SQLite3

#### Build

```bash
# Clone the repository
git clone https://github.com/themobileprof/clipilot.git
cd clipilot

# Build the server
go build -o clipilot-server ./cmd/registry

# Run the server
./clipilot-server \
  --port=8080 \
  --data=./data \
  --admin=admin \
  --password=your-secure-password
```

#### Admin Setup

After starting the server, create an admin user:

```bash
# Run the admin creation script
./scripts/create-admin.sh

# Or use environment variables before starting
export ADMIN_USER=admin
export ADMIN_PASSWORD=your-secure-password
./clipilot-server
```

See [scripts/ADMIN_SETUP.md](scripts/ADMIN_SETUP.md) for detailed instructions.

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
name: setup_docker
id: org.example.setup_docker
version: 1.0.0
description: Install Docker Engine on Linux
tags: [docker, container, setup]
requires: [check_os, check_sudo]
provides: [docker_installed]

flows:
  main:
    start: check_platform
    steps:
      check_platform:
        type: action
        message: "Checking platform compatibility..."
        command: |
          if [ -f /etc/debian_version ]; then
            echo "debian"
          elif [ -f /etc/redhat-release ]; then
            echo "rhel"
          else
            echo "unsupported"
          fi
        next: install_docker
      
      install_docker:
        type: action
        message: "Installing Docker..."
        command: |
          sudo apt-get update
          sudo apt-get install -y docker.io
        next: end
      
      end:
        type: terminal
        message: "Docker installed successfully!"
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
  -F "file=@setup_docker.yaml"
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
