# CLIPilot Registry Server

The CLIPilot Registry is a web application for hosting and distributing CLIPilot modules. It's designed to run as a Docker container.

## Quick Start (Local Development)

### Using Docker Compose (Development Only)

⚠️ **Note:** Docker Compose is suitable for local development and testing, but not recommended for production. See [Production Deployment](#production-deployment) below.

1. **Create environment file**:
   ```bash
   cat > .env << EOF
   ADMIN_USER=admin
   ADMIN_PASS=your_secure_password_here
   EOF
   ```

2. **Start the registry**:
   ```bash
   docker-compose up -d
   ```

3. **Access the registry**:
   - Web UI: http://localhost:8080
   - API: http://localhost:8080/api/modules

4. **View logs**:
   ```bash
   docker-compose logs -f registry
   ```

5. **Stop the registry**:
   ```bash
   docker-compose down
   ```

### Using Docker Run (Development)

```bash
docker run -d \
  --name clipilot-registry \
  -p 8080:8080 \
  -v registry-data:/app/data \
  -e REGISTRY_ADMIN_USER=admin \
  -e REGISTRY_ADMIN_PASS=changeme \
  themobileprof/clipilot-registry:latest
```

## Building from Source

```bash
# Build the Docker image
docker build -f Dockerfile.registry -t clipilot-registry .

# Run the container
docker run -d -p 8080:8080 -v registry-data:/app/data clipilot-registry
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REGISTRY_PORT` | `8080` | Port the server listens on |
| `REGISTRY_ADMIN_USER` | `admin` | Admin username for uploads |
| `REGISTRY_ADMIN_PASS` | `changeme` | Admin password (change this!) |
| `REGISTRY_DB_PATH` | `/app/data/registry.db` | SQLite database path |
| `REGISTRY_UPLOAD_DIR` | `/app/data/uploads` | Module upload directory |

## Data Persistence

The registry stores data in `/app/data`:
- `registry.db` - SQLite database with module metadata
- `uploads/` - Uploaded module files

Mount this directory as a volume to persist data:
```bash
-v /path/on/host:/app/data
```

## Production Deployment

⚠️ **Docker Compose is NOT recommended for production.** Use one of these production-grade solutions:

### Option 1: Kubernetes (Recommended for Scale)

**Deployment manifest:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: clipilot-registry
spec:
  replicas: 2
  selector:
    matchLabels:
      app: clipilot-registry
  template:
    metadata:
      labels:
        app: clipilot-registry
    spec:
      containers:
      - name: registry
        image: themobileprof/clipilot-registry:latest
        ports:
        - containerPort: 8080
        env:
        - name: REGISTRY_ADMIN_USER
          valueFrom:
            secretKeyRef:
              name: registry-secrets
              key: admin-user
        - name: REGISTRY_ADMIN_PASS
          valueFrom:
            secretKeyRef:
              name: registry-secrets
              key: admin-pass
        volumeMounts:
        - name: data
          mountPath: /app/data
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: registry-data-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: clipilot-registry
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: clipilot-registry
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: clipilot-registry
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - registry.example.com
    secretName: registry-tls
  rules:
  - host: registry.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: clipilot-registry
            port:
              number: 80
```

### Option 2: Docker Swarm (Simpler Alternative)

```bash
# Initialize swarm (on manager node)
docker swarm init

# Create secrets
echo "admin" | docker secret create registry_user -
echo "secure_password_here" | docker secret create registry_pass -

# Deploy stack
docker stack deploy -c docker-stack.yml registry
```

**docker-stack.yml:**

```yaml
version: '3.8'

services:
  registry:
    image: themobileprof/clipilot-registry:latest
    ports:
      - "8080:8080"
    volumes:
      - registry-data:/app/data
    secrets:
      - registry_user
      - registry_pass
    environment:
      - REGISTRY_ADMIN_USER_FILE=/run/secrets/registry_user
      - REGISTRY_ADMIN_PASS_FILE=/run/secrets/registry_pass
    deploy:
      replicas: 2
      update_config:
        parallelism: 1
        delay: 10s
      restart_policy:
        condition: on-failure
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.1'
          memory: 128M
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3

  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs:/etc/nginx/certs:ro
    depends_on:
      - registry
    deploy:
      replicas: 1

volumes:
  registry-data:
    driver: local

secrets:
  registry_user:
    external: true
  registry_pass:
    external: true
```

### Option 3: Managed Container Services

**AWS ECS/Fargate:**
```bash
# Create task definition
aws ecs register-task-definition --cli-input-json file://task-definition.json

# Create service
aws ecs create-service \
  --cluster production \
  --service-name clipilot-registry \
  --task-definition clipilot-registry:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --load-balancers targetGroupArn=arn:aws:...,containerName=registry,containerPort=8080
```

**Google Cloud Run:**
```bash
gcloud run deploy clipilot-registry \
  --image themobileprof/clipilot-registry:latest \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars REGISTRY_ADMIN_USER=admin \
  --set-secrets REGISTRY_ADMIN_PASS=registry-password:latest
```

**Azure Container Instances:**
```bash
az container create \
  --resource-group myResourceGroup \
  --name clipilot-registry \
  --image themobileprof/clipilot-registry:latest \
  --dns-name-label clipilot-registry \
  --ports 8080 \
  --environment-variables REGISTRY_ADMIN_USER=admin \
  --secure-environment-variables REGISTRY_ADMIN_PASS=secure_password
```

### Production Best Practices

1. **Orchestration**: Use Kubernetes, Docker Swarm, or managed container services
2. **HTTPS/TLS**: Always use reverse proxy (nginx, Traefik, cloud load balancer) with SSL
3. **Secrets Management**: Use orchestrator secrets, not environment variables
4. **High Availability**: Run multiple replicas with load balancing
5. **Persistent Storage**: Use network volumes (NFS, EBS, Cloud Filestore)
6. **Monitoring**: Implement health checks, logging, and metrics
7. **Backups**: Automate regular backups of `/app/data`
8. **Updates**: Use rolling updates with zero downtime
9. **Resource Limits**: Set CPU/memory limits to prevent resource exhaustion
10. **Security**: Run as non-root, scan images, keep updated

## API Endpoints

- `GET /` - Web UI home page
- `GET /api/modules` - List all modules (JSON)
- `GET /api/modules/:id` - Get module details
- `GET /api/modules/:id/download` - Download module file
- `POST /api/modules` - Upload new module (requires auth)
- `GET /health` - Health check endpoint

## Client Configuration

To use a custom registry with CLIPilot:

```bash
clipilot settings set registry_url "http://your-registry:8080"
```

## Troubleshooting

### Container won't start
```bash
# Check logs
docker logs clipilot-registry

# Verify permissions on data volume
docker exec clipilot-registry ls -la /app/data
```

### Cannot upload modules
- Verify admin credentials are correct
- Check disk space: `docker exec clipilot-registry df -h /app/data`
- Review logs for authentication errors

### Performance issues
- Increase container resources if needed
- Check SQLite database size and consider vacuuming
- Monitor with: `docker stats clipilot-registry`

## Development

For local development:

```bash
# Build and run locally
go run ./cmd/registry

# Or with hot reload (requires air)
air -c .air.toml
```

The registry will be available at http://localhost:8080
