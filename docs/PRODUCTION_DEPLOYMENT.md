# Production Deployment Guide

## GitHub Secrets Setup

You need to configure these secrets in your GitHub repository:

### 1. Docker Hub Credentials
- `DOCKER_USERNAME` - Your Docker Hub username
- `DOCKERHUB_TOKEN` - Docker Hub access token (Settings → Security → New Access Token)

### 2. SSH Access to Production Server
- `SSH_HOST` - Your server's IP or domain
- `SSH_USERNAME` - SSH user (e.g., `root` or `deploy`)
- `SSH_PRIVATE_KEY` - Your SSH private key (contents of `~/.ssh/id_rsa`)
- `SSH_PORT` - SSH port (optional, defaults to 22)

### 3. Environment Configuration
- `ENV_FILE` - Complete .env file content for production

## Creating the ENV_FILE Secret

1. Copy `.env.production` and customize it:
   ```bash
   cp .env.production .env.prod.custom
   nano .env.prod.custom
   ```

2. Update these values:
   ```env
   PORT=8080
   ADMIN_USER=admin
   ADMIN_PASSWORD=CHANGE_THIS_TO_SECURE_PASSWORD
   
   BASE_URL=https://registry.yourdomain.com
   REGISTRY_URL=https://registry.yourdomain.com
   
   DATA_DIR=/app/data
   
   SESSION_SECRET=GENERATE_RANDOM_64_CHAR_STRING
   SESSION_TIMEOUT=86400
   MAX_UPLOAD_SIZE=10485760
   ```

3. Add as GitHub Secret:
   - Go to: `Settings → Secrets and variables → Actions`
   - Click: `New repository secret`
   - Name: `ENV_FILE`
   - Value: Paste entire contents of `.env.prod.custom`
   - Click: `Add secret`

## Generating Secure Values

```bash
# Generate secure admin password
openssl rand -base64 32

# Generate session secret
openssl rand -hex 64
```

## Deploying

### Manual Deployment via GitHub Actions

1. Go to: `Actions → Deploy Registry to Production`
2. Click: `Run workflow`
3. Select: `production` or `staging`
4. Click: `Run workflow`

The workflow will:
- Build multi-arch Docker image
- Push to Docker Hub
- SSH to your server
- Create .env file from secret
- Deploy new container on port 8082
- Clean up temporary files

### Manual Deployment via SSH

If you prefer manual deployment:

```bash
# SSH to your server
ssh user@your-server.com

# Create .env file (paste your ENV_FILE content)
nano /opt/clipilot-registry/.env

# Run the container
docker run -d \
  --name clipilot-registry \
  --restart unless-stopped \
  -p 8082:8080 \
  -v clipilot-registry-data:/app/data \
  --env-file /opt/clipilot-registry/.env \
  themobileprof/clipilot-registry:latest

# Check it's running
docker ps | grep clipilot
docker logs clipilot-registry
```

## Nginx Reverse Proxy Setup

If using Nginx to serve on standard HTTPS port:

```nginx
server {
    listen 80;
    server_name registry.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name registry.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/registry.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/registry.yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Verification

After deployment, verify:

```bash
# Check container is running
curl https://registry.yourdomain.com/

# Check API
curl https://registry.yourdomain.com/api/modules

# Test authentication
curl -X POST https://registry.yourdomain.com/login \
  -d "username=admin&password=your_password" \
  -c cookies.txt

# Test upload (requires auth)
curl -b cookies.txt -X POST https://registry.yourdomain.com/api/upload \
  -F "module=@test_module.yaml"
```

## Updating Production

To update the production deployment:

1. **Update code**: Push changes to main branch
2. **Create release**: Tag with new version (`git tag v0.1.7`)
3. **Run workflow**: Manually trigger "Deploy Registry to Production"

The workflow automatically:
- Pulls latest code
- Builds new image
- Deploys to server
- Maintains data volume (no data loss)

## Rollback

If something goes wrong:

```bash
# SSH to server
ssh user@your-server.com

# Stop current container
docker stop clipilot-registry
docker rm clipilot-registry

# Run previous version
docker run -d \
  --name clipilot-registry \
  --restart unless-stopped \
  -p 8082:8080 \
  -v clipilot-registry-data:/app/data \
  --env-file /opt/clipilot-registry/.env \
  themobileprof/clipilot-registry:v0.1.6
```

## Security Best Practices

1. ✅ **Never commit .env files** - Use GitHub Secrets
2. ✅ **Use strong passwords** - Generate with `openssl rand`
3. ✅ **Enable HTTPS** - Use Let's Encrypt/Certbot
4. ✅ **Restrict SSH access** - Use key-based auth only
5. ✅ **Regular backups** - Backup `/var/lib/docker/volumes/clipilot-registry-data`
6. ✅ **Monitor logs** - `docker logs -f clipilot-registry`
7. ✅ **Update regularly** - Keep Docker images up to date

## Troubleshooting

**Container won't start:**
```bash
docker logs clipilot-registry
```

**Port already in use:**
```bash
# Check what's using port 8082
sudo lsof -i :8082
# Use different port in deployment
```

**Can't connect to server:**
```bash
# Check firewall
sudo ufw status
sudo ufw allow 8082/tcp
```

**Environment variables not loading:**
```bash
# Verify env file
docker exec clipilot-registry env | grep -E 'PORT|ADMIN|BASE_URL'
```
