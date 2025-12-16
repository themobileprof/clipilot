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
   
   BASE_URL=https://clipilot.themobileprof.com
   REGISTRY_URL=https://clipilot.themobileprof.com
   
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

## Reverse Proxy Setup (Recommended)

**Let Nginx/Caddy handle external ports** - your app always runs on internal port 8080.

### Option 1: Nginx (Most Common)

```nginx
# /etc/nginx/sites-available/clipilot-registry
server {
    listen 80;
    server_name clipilot.themobileprof.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name clipilot.themobileprof.com;

    # SSL certificates (use certbot for Let's Encrypt)
    ssl_certificate /etc/letsencrypt/live/clipilot.themobileprof.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/clipilot.themobileprof.com/privkey.pem;
    
    # Security headers
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;

    location / {
        # Proxy to Docker container by name (Docker DNS)
        proxy_pass http://clipilot-registry:8080;
        
        # Or use container IP (get with: docker inspect clipilot-registry)
        # proxy_pass http://172.17.0.2:8080;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket support (if needed)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

**Enable the site:**
```bash
sudo ln -s /etc/nginx/sites-available/clipilot-registry /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Get SSL certificate
sudo certbot --nginx -d clipilot.themobileprof.com
```

### Option 2: Caddy (Automatic HTTPS)

```caddyfile
# /etc/caddy/Caddyfile
cligilot.themobileprof.com {
    reverse_proxy clipilot-registry:8080
}
```

**That's it!** Caddy automatically handles:
- HTTPS certificates (Let's Encrypt)
- HTTP → HTTPS redirect
- Certificate renewal

```bash
sudo systemctl reload caddy
```

### Option 3: Traefik (Docker-Native)

```yaml
# docker-compose.yml with Traefik
version: '3.8'

services:
  traefik:
    image: traefik:v2.10
    command:
      - "--api.insecure=true"
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.le.acme.email=you@example.com"
      - "--certificatesresolvers.le.acme.storage=/letsencrypt/acme.json"
      - "--certificatesresolvers.le.acme.httpchallenge.entrypoint=web"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./letsencrypt:/letsencrypt

  clipilot-registry:
    image: themobileprof/clipilot-registry:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.registry.rule=Host(`clipilot.themobileprof.com`)"
      - "traefik.http.routers.registry.entrypoints=websecure"
      - "traefik.http.routers.registry.tls.certresolver=le"
      - "traefik.http.services.registry.loadbalancer.server.port=8080"
    volumes:
      - registry-data:/app/data
    env_file:
      - .env.production
```

## Why Use a Reverse Proxy?

✅ **No port conflicts** - All apps run on their own internal ports  
✅ **Single HTTPS endpoint** - Port 443 for all services  
✅ **Automatic SSL** - Let's Encrypt integration  
✅ **Load balancing** - Run multiple instances  
✅ **Better security** - Apps don't need to be exposed  
✅ **Easy routing** - Use subdomains/paths for multiple services  

### Example: Multiple Services on One Server

```nginx
# clipilot.themobileprof.com → clipilot-registry:8080
server {
    listen 443 ssl http2;
    server_name clipilot.themobileprof.com;
    location / {
        proxy_pass http://clipilot-registry:8080;
    }
}

# api.themobileprof.com → your-backend:8000
server {
    listen 443 ssl http2;
    server_name api.themobileprof.com;
    location / {
        proxy_pass http://themobileprof-backend:8080;
    }
}

# app.themobileprof.com → your-frontend:8081
server {
    listen 443 ssl http2;
    server_name app.themobileprof.com;
    location / {
        proxy_pass http://tmp-react-frontend:8081;
    }
}
```

## Verification

After deployment, verify:

```bash
# Check container is running
curl https://clipilot.themobileprof.com/

# Check API
curl https://clipilot.themobileprof.com/api/modules

# Test authentication
curl -X POST https://clipilot.themobileprof.com/login \
  -d "username=admin&password=your_password" \
  -c cookies.txt

# Test upload (requires auth)
curl -b cookies.txt -X POST https://clipilot.themobileprof.com/api/upload \
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
