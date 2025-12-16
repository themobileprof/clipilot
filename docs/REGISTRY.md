# CLIPilot Registry

Web-based module registry for CLIPilot community modules.

## Overview

The registry server allows authenticated users to upload and share YAML modules for CLIPilot. Users can browse available modules, download them directly, or install them via the CLI tool.

## Quick Start

### 1. Build and Run the Registry

```bash
# Build the registry binary
go build -o registry ./cmd/registry

# Run with admin credentials
./registry --password=your_secure_password

# Custom configuration
./registry \
  --port=8080 \
  --admin=admin \
  --password=your_secure_password \
  --data=./data \
  --static=./server/static \
  --templates=./server/templates
```

### 2. Access the Registry

Open your browser to `http://localhost:8080`

- Browse modules: `/modules`
- Login: `/login` (use admin credentials)
- Upload: `/upload` (requires login)

### 3. Install Modules from CLI

```bash
# Set registry URL (required for registry features)
clipilot settings set registry_url http://your-registry.com
# Or use environment variable: export REGISTRY_URL=http://your-registry.com

# Install a module by ID
clipilot modules install 1

# List installed modules
clipilot modules list
```

## Features

- **User Authentication**: Simple session-based authentication system
- **Module Upload**: Web interface for uploading YAML modules
- **Validation**: Automatic YAML validation and duplicate detection
- **Module Browser**: Public-facing module listing with download counters
- **REST API**: JSON API for CLI integration
- **Documentation**: Built-in specification and ChatGPT prompt examples

## Configuration

### Command-line Flags

- `--port`: Server port (default: 8080)
- `--admin`: Admin username (default: admin)
- `--password`: Admin password (required)
- `--data`: Data directory for uploads and database (default: ./data)
- `--static`: Static files directory (default: ./server/static)
- `--templates`: Templates directory (default: ./server/templates)

### Data Storage

The registry uses SQLite to store module metadata and stores uploaded YAML files in the filesystem:

```
data/
  registry.db          # SQLite database
  uploads/             # Uploaded module files
    module-v1.0-123.yaml
    module-v2.0-456.yaml
```

## API Endpoints

### Public Endpoints

- `GET /` - Home page
- `GET /modules` - Browse all modules (HTML)
- `GET /modules/:id` - Download specific module (YAML)
- `GET /api/modules` - List all modules (JSON)
- `GET /api/modules/:id` - Get module details (JSON)

### Authenticated Endpoints

- `POST /login` - User login
- `GET /logout` - User logout
- `GET /upload` - Upload form page
- `POST /api/upload` - Upload module (multipart form)
- `GET /my-modules` - List user's uploaded modules

### API Response Format

```json
[
  {
    "id": 1,
    "name": "docker_install",
    "version": "1.0.0",
    "description": "Install Docker on Ubuntu/Debian",
    "author": "CLIPilot Team",
    "downloads": 42
  }
]
```

## Module Upload

### Requirements

Modules must be valid YAML files following the CLIPilot module specification:

```yaml
metadata:
  name: module_name
  version: "1.0.0"
  description: Brief description
  author: Your Name
  keywords:
    - keyword1
    - keyword2

flows:
  main:
    steps:
      - type: instruction
        name: step_name
        message: "Step message"
```

### Validation

The registry automatically validates:

- YAML syntax correctness
- Required metadata fields (name, version)
- Duplicate module names/versions
- File size (max 10MB)

## Using ChatGPT to Generate Modules

The upload page includes a detailed prompt you can use with ChatGPT to generate valid module YAML files:

1. Go to `/upload` (requires login)
2. Copy the ChatGPT prompt
3. Paste into ChatGPT with your task description
4. Upload the generated YAML

Example prompt usage:
> "Create a CLIPilot module YAML file for setting up a Python virtual environment with pip and installing common data science packages."

## Deployment

### Production Considerations

1. **HTTPS**: Always use HTTPS in production. Configure with a reverse proxy (nginx, Caddy)
2. **Database Backups**: Regularly backup `data/registry.db`
3. **Password Security**: Use strong admin passwords
4. **File Storage**: Monitor `data/uploads/` directory size
5. **Session Security**: Enable secure cookies when using HTTPS

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o registry ./cmd/registry

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/registry .
COPY --from=builder /app/server ./server
EXPOSE 8080
CMD ["./registry", "--password=$ADMIN_PASSWORD"]
```

### systemd Service

```ini
[Unit]
Description=CLIPilot Registry
After=network.target

[Service]
Type=simple
User=clipilot
WorkingDirectory=/opt/clipilot-registry
ExecStart=/opt/clipilot-registry/registry --password=your_password
Restart=always

[Install]
WantedBy=multi-user.target
```

## Security Notes

- Default authentication is basic (username/password)
- Sessions stored in memory (cleared on restart)
- No rate limiting by default
- Consider adding:
  - OAuth/OIDC for production
  - Rate limiting for API endpoints
  - File scanning for uploaded content
  - User registration system

## Database Schema

```sql
CREATE TABLE modules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    author TEXT,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    uploaded_by TEXT NOT NULL,
    file_path TEXT NOT NULL,
    original_filename TEXT,
    downloads INTEGER DEFAULT 0,
    UNIQUE(name, version)
);
```

## Development

### Running Locally

```bash
# Install dependencies
go mod download

# Run in development mode
go run ./cmd/registry --password=dev123

# Run tests (when implemented)
go test ./server/...
```

### File Structure

```
cmd/registry/
  main.go              # Entry point
server/
  handlers/
    handlers.go        # HTTP handlers
    migration.sql      # Database schema
  auth/
    auth.go           # Authentication logic
  static/
    style.css         # CSS styling
  templates/
    *.html            # HTML templates
```

## Troubleshooting

### Port Already in Use

```bash
# Use a different port
./registry --port=8081 --password=pass123
```

### Template Not Found

Ensure templates directory exists and contains HTML files:

```bash
ls -la server/templates/
# Should show: base.html, home.html, upload.html, etc.
```

### Database Locked

SQLite database may be locked if multiple processes access it:

```bash
# Stop all registry instances
pkill -f registry

# Check for stale lock files
rm -f data/registry.db-wal data/registry.db-shm
```

## License

Same as CLIPilot main project (see LICENSE file).
