# Environment Variables Configuration

The CLIPilot Registry supports configuration through environment variables, making it ideal for containerized deployments and production environments.

## Configuration Priority

The registry uses the following priority order (highest to lowest):

1. **Command-line flags** - Override everything
2. **Environment variables** - Loaded from system or .env file
3. **Default values** - Built-in sensible defaults

## Using .env File

### 1. Create your .env file

```bash
# Copy the example
cp .env.example .env

# Edit with your values
nano .env
```

### 2. Example .env file

```bash
# Server Configuration
PORT=8080

# Data Storage
DATA_DIR=./data

# Static Assets
STATIC_DIR=./server/static
TEMPLATE_DIR=./server/templates

# Authentication
ADMIN_USER=admin
ADMIN_PASSWORD=your_secure_password_here
```

### 3. Run the registry

```bash
# Will automatically load .env file
./bin/registry

# Or specify a different env file location
ENV_FILE=prod.env ./bin/registry
```

## Environment Variables

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server listening port |
| `DATA_DIR` | `./data` | Directory for uploads and database |
| `STATIC_DIR` | `./server/static` | Static files directory |
| `TEMPLATE_DIR` | `./server/templates` | HTML templates directory |
| `ADMIN_USER` | `admin` | Admin username |
| `ADMIN_PASSWORD` | *(required)* | Admin password |

### Usage Examples

#### Development
```bash
export ADMIN_PASSWORD=dev123
./bin/registry
```

#### Production with .env
```bash
# .env file
PORT=443
DATA_DIR=/var/lib/clipilot/data
ADMIN_USER=administrator
ADMIN_PASSWORD=StrongP@ssw0rd!
```

#### Docker
```bash
docker run -d \
  -e PORT=8080 \
  -e DATA_DIR=/data \
  -e ADMIN_PASSWORD=secure123 \
  -v /host/data:/data \
  -p 8080:8080 \
  clipilot/registry
```

#### Kubernetes
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: registry-config
data:
  PORT: "8080"
  DATA_DIR: "/data"
---
apiVersion: v1
kind: Secret
metadata:
  name: registry-secrets
type: Opaque
stringData:
  ADMIN_PASSWORD: "your-secret-password"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clipilot-registry
spec:
  template:
    spec:
      containers:
      - name: registry
        image: clipilot/registry:latest
        envFrom:
        - configMapRef:
            name: registry-config
        - secretRef:
            name: registry-secrets
```

#### systemd Service
```ini
[Unit]
Description=CLIPilot Registry
After=network.target

[Service]
Type=simple
User=clipilot
WorkingDirectory=/opt/clipilot-registry
Environment="PORT=8080"
Environment="DATA_DIR=/var/lib/clipilot"
Environment="ADMIN_USER=admin"
EnvironmentFile=/etc/clipilot/registry.env
ExecStart=/opt/clipilot-registry/registry
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
# /etc/clipilot/registry.env
ADMIN_PASSWORD=production_password
```

## Command-Line Flags (Override Environment)

Even with environment variables set, you can override them with flags:

```bash
# Use .env for most settings, override port
./bin/registry --port=9000

# Override everything
./bin/registry \
  --port=8080 \
  --data=/custom/data \
  --admin=superadmin \
  --password=SecurePass123
```

## Security Best Practices

### 1. Never Commit .env Files

The `.gitignore` file already excludes `.env` files:

```bash
# Already in .gitignore
.env
data/
*.db
```

### 2. Use Strong Passwords

```bash
# Generate a secure password
openssl rand -base64 32

# Or use pwgen
pwgen -s 32 1
```

### 3. Restrict File Permissions

```bash
# Protect your .env file
chmod 600 .env

# Only owner can read
ls -la .env
# -rw------- 1 user user 256 Dec 14 10:00 .env
```

### 4. Use Secrets Management in Production

For production deployments, use proper secrets management:

- **Docker Secrets** for Swarm
- **Kubernetes Secrets** for K8s
- **AWS Secrets Manager** for AWS
- **HashiCorp Vault** for multi-cloud
- **Azure Key Vault** for Azure
- **Google Secret Manager** for GCP

### 5. Environment-Specific Files

```bash
# Different files for different environments
.env.development
.env.staging
.env.production

# Load specific environment
ln -sf .env.production .env
./bin/registry
```

## Troubleshooting

### Password Not Set Error

```
Error: Admin password is required. Set ADMIN_PASSWORD env var or use --password flag
```

**Solution:**
```bash
# Set environment variable
export ADMIN_PASSWORD=mypassword
./bin/registry

# Or use flag
./bin/registry --password=mypassword

# Or create .env file
echo "ADMIN_PASSWORD=mypassword" > .env
./bin/registry
```

### .env File Not Loading

The `.env` file must be in the current working directory when you run the binary:

```bash
# Wrong - .env file not found
cd /tmp
/opt/registry/bin/registry

# Right - run from directory with .env
cd /opt/registry
./bin/registry

# Or use absolute path for .env
cd /tmp
ENV_FILE=/opt/registry/.env /opt/registry/bin/registry
```

### Debugging Environment Variables

```bash
# Check what's set
printenv | grep -E '(PORT|DATA_DIR|ADMIN)'

# Test with specific values
PORT=9090 ADMIN_PASSWORD=test ./bin/registry
```

## Migration from Flags to Environment Variables

If you're currently using command-line flags, migration is easy:

**Before:**
```bash
./bin/registry \
  --port=8080 \
  --data=/var/data \
  --admin=admin \
  --password=secret123
```

**After (.env file):**
```bash
# .env
PORT=8080
DATA_DIR=/var/data
ADMIN_USER=admin
ADMIN_PASSWORD=secret123
```

```bash
# Run
./bin/registry
```

Both methods work simultaneously, so you can migrate gradually!

## Summary

✅ **Environment variables** - Production-ready configuration  
✅ **.env file support** - Easy local development  
✅ **Command-line flags** - Quick overrides  
✅ **Sensible defaults** - Works out of the box  
✅ **Security-focused** - .env files excluded from git  
✅ **Container-friendly** - Perfect for Docker/K8s  

The registry now follows the [12-factor app](https://12factor.net/) methodology for configuration management!
