# Admin User Creation for Docker Deployments

This directory contains scripts for creating admin users in the CLIPilot Registry.

## Local Development

For local development (non-Docker), use:

```bash
./scripts/create-admin.sh
```

The script will prompt for:
- Admin password (twice for confirmation)
- Database location (defaults to `./data/registry.db`)

## Docker Deployment

For Docker deployments, you have two options:

### Option 1: Use Environment Variables (Recommended)

Set admin credentials in your `.env` file or docker-compose.yml:

```bash
ADMIN_USER=admin
ADMIN_PASSWORD=your-secure-password-here
```

The registry will create the admin user automatically on startup using these credentials.

### Option 2: Manual Creation via Docker Exec

1. Start the registry container:
```bash
docker compose up -d
```

2. Install sqlite3 in the container (temporary):
```bash
docker compose exec registry sh -c 'apk add sqlite' 2>/dev/null || \
docker exec clipilot-registry sh -c 'apt-get update && apt-get install -y sqlite3'
```

Note: The distroless base image doesn't support this. Use a different base image if you need shell  access.

3. Access the database volume from host:

```bash
# Find the volume mount point
docker volume inspect clipilot_registry-data

# Copy the create-admin.sh script and run with the volume path
docker run --rm -v clipilot_registry-data:/data \
  -v $(pwd)/scripts:/scripts \
  alpine:latest sh /scripts/create-admin.sh
```

### Option 3: Direct Database Access (Advanced)

If you have direct access to the Docker volume on your host:

```bash
# Find the volume location
VOLUME_PATH=$(docker volume inspect clipilot_registry-data -f '{{.Mountpoint}}')

# Run the script with the correct DB path
DB_PATH="$VOLUME_PATH/registry.db" ./scripts/create-admin.sh
```

## API Key Generation

After creating the admin user, the script will generate an API key automatically.

**Save this API key securely** - you'll need it for:
- Clio CI/CD (add as `CLIPILOT_API_KEY` secret in GitHub)
- Automated module uploads
- Admin API operations

## Security Best Practices

1. **Strong Passwords**: Use at least 16 characters with mixed case, numbers, and symbols
2. **Rotate API Keys**: Periodically regenerate API keys and update CI/CD secrets
3. **Limit Access**: Only grant admin access to trusted users
4. **HTTPS Only**: Always use HTTPS in production to protect credentials

## Troubleshooting

### Database Not Found

Make sure the registry server has started at least once to initialize the database.

```bash
docker compose up -d
docker compose logs registry | grep "Server ready"
```

### Permission Denied

Ensure you have write access to the database file:

```bash
# For local development
chmod 644 ./data/registry.db

# For Docker
docker compose exec registry ls -la /app/data/
```

### User Already Exists

The script will prompt you to update the password if the user already exists.

## Manual SQL Approach

If the script doesn't work, you can create an admin manually:

```sql
-- Generate password hash (replace 'your-password' with your actual password)
-- In bash: echo -n "your-password" | sha256sum

-- Insert admin user
INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
VALUES (
  'admin',
  'admin@localhost',
  'YOUR_PASSWORD_HASH_HERE',
  'admin',
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP
);

-- Generate API key (replace with your own random hex string)
INSERT INTO api_keys (user_id, key_hash, name, scopes, expires_at, revoked, created_at)
VALUES (
  (SELECT id FROM users WHERE username = 'admin'),
  'YOUR_API_KEY_HASH_HERE',
  'ci-cd-key',
  '["upload", "admin"]',
  NULL,
  0,
  CURRENT_TIMESTAMP
);
```

Run with:
```bash
sqlite3 /path/to/registry.db < admin-setup.sql
```
