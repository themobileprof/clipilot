# Admin User Creation

This directory contains scripts for creating admin users in the CLIPilot Registry.

## Local Development

```bash
./scripts/create-admin.sh
```

The script will prompt for:
- Admin password (twice for confirmation)
- Database location (defaults to `./data/registry.db`)

## Production

Set admin credentials in the GitHub `ENV_FILE` secret (written to `~/clipilot-registry/env` on deploy):

```bash
ADMIN_USER=admin
ADMIN_PASSWORD=your-secure-password-here
BASE_URL=https://your-domain.com
```

The registry creates the admin user automatically on startup using these credentials.

To create or update an admin manually:

```bash
DB_PATH=~/clipilot-data/registry.db ./scripts/create-admin.sh
```

## API Key Generation

After creating the admin user, the script will generate an API key automatically.

**Save this API key securely** — you'll need it for:
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
systemctl --user status clipilot-registry
journalctl --user -u clipilot-registry -n 50
```

### Permission Denied

Ensure your SSH user owns the data directory:

```bash
chown -R "$(whoami):$(whoami)" ~/clipilot-data ~/clipilot-registry
```

### User Already Exists

The script will prompt you to update the password if the user already exists.
