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
   
   # GitHub OAuth (optional - for community contributions)
   GITHUB_CLIENT_ID=your_github_oauth_client_id
   GITHUB_CLIENT_SECRET=your_github_oauth_client_secret
   
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

## Setting up GitHub OAuth (Optional)

To allow community members to contribute modules via GitHub login:

1. Go to: https://github.com/settings/developers
2. Click: **New OAuth App**
3. Fill in:
   - **Application name**: CLIPilot Registry
   - **Homepage URL**: https://clipilot.themobileprof.com
   - **Authorization callback URL**: https://clipilot.themobileprof.com/auth/github/callback
4. Click: **Register application**
5. Copy the **Client ID** and generate a **Client Secret**
6. Add to your `.env.production`:
   ```env
   GITHUB_CLIENT_ID=your_client_id_here
   GITHUB_CLIENT_SECRET=your_client_secret_here
   ```

**Benefits:**
- Community members can login with GitHub (no need to create accounts)
- Contributions are automatically linked to GitHub profiles
- Admin account still works for administrative access

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

## Reverse Proxy Setup (Optional)

The registry is now accessible directly on port 8082. You can optionally use Nginx for HTTPS and cleaner URLs.

### Option 1: Direct Access (Simplest)

Access directly via port 8082:
```
http://your-server-ip:8082
```

For HTTPS, use Nginx below.

### Option 2: Nginx with HTTPS (Recommended for Production)

```nginx
# /etc/nginx/sites-available/clipilot.themobileprof.com.conf
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
        # Proxy to localhost:8082 (where Docker exposes the container)
        proxy_pass http://localhost:8082;
        
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
sudo ln -s /etc/nginx/sites-available/clipilot.themobileprof.com.conf /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# Get SSL certificate
sudo certbot --nginx -d clipilot.themobileprof.com
```

### Option 3: Caddy (Automatic HTTPS)

```caddyfile
# /etc/caddy/Caddyfile
clipilot.themobileprof.com {
    reverse_proxy localhost:8082
}
```

**That's it!** Caddy automatically handles:
- HTTPS certificates (Let's Encrypt)
- HTTP → HTTPS redirect
- Certificate renewal

```bash
sudo systemctl reload caddy
```

### Option 4: Traefik (Docker-Native)

If you prefer Docker-native routing:

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
    ports:
      - "8082:8080"
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.registry.rule=Host(`clipilot.themobileprof.com`)"
      - "traefik.http.routers.registry.entrypoints=websecure"
      - "traefik.http.routers.registry.tls.certresolver=le"
      - "traefik.http.services.registry.loadbalancer.server.port=8082"
    volumes:
      - registry-data:/app/data
    env_file:
      - .env.production
```

## Why Use a Reverse Proxy?

✅ **HTTPS Support** - Free SSL certificates with Let's Encrypt  
✅ **Clean URLs** - Use domain name instead of port numbers  
✅ **Single HTTPS endpoint** - Port 443 for all services  
✅ **Better security** - Hide internal ports from public  
✅ **Easy routing** - Use subdomains/paths for multiple services  

**Without Nginx:** `http://your-server-ip:8082`  
**With Nginx:** `https://clipilot.themobileprof.com`

### Example: Multiple Services on One Server

Each service exposed on its own port, proxied via Nginx with HTTPS:

```nginx
# clipilot.themobileprof.com → localhost:8082 (registry)
server {
    listen 443 ssl http2;
    server_name clipilot.themobileprof.com;
    location / {
        proxy_pass http://localhost:8082;
    }
}

# api.themobileprof.com → localhost:8000 (your backend)
server {
    listen 443 ssl http2;
    server_name api.themobileprof.com;
    location / {
        proxy_pass http://localhost:8000;
    }
}

# app.themobileprof.com → localhost:8081 (your frontend)
server {
    listen 443 ssl http2;
    server_name app.themobileprof.com;
    location / {
        proxy_pass http://localhost:8081;
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
