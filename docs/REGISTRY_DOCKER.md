# CLIPilot Registry Server

The CLIPilot Registry is a web application for hosting and distributing CLIPilot modules. It runs as a Docker container.

## Quick Start

### Pull and Run the Image

```bash
# Pull the image from Docker Hub
docker pull themobileprof/clipilot-registry:latest

# Run the container
docker run -d \
  --name clipilot-registry \
  -p 8080:8080 \
  -v registry-data:/app/data \
  -e REGISTRY_ADMIN_USER=admin \
  -e REGISTRY_ADMIN_PASS=changeme \
  themobileprof/clipilot-registry:latest

# Access the registry
# Web UI: http://localhost:8080
# API: http://localhost:8080/api/modules
```

**Basic Docker commands:**
```bash
# View logs
docker logs -f clipilot-registry

# Stop the container
docker stop clipilot-registry

# Start it again
docker start clipilot-registry

# Remove the container
docker rm clipilot-registry

# Remove the image
docker rmi themobileprof/clipilot-registry:latest
```

## Build, Push, and Deploy Your Own

### 1. Build the Image

```bash
# Build from source
docker build -f Dockerfile.registry -t your-username/clipilot-registry:latest .

# Build for multiple architectures (requires buildx)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f Dockerfile.registry \
  -t your-username/clipilot-registry:latest \
  .
```

### 2. Push to Docker Hub

```bash
# Login to Docker Hub
docker login

# Push the image
docker push your-username/clipilot-registry:latest

# Push with version tag
docker tag your-username/clipilot-registry:latest your-username/clipilot-registry:1.0.0
docker push your-username/clipilot-registry:1.0.0
```

### 3. Pull and Run on Any Server

```bash
# On your production server
docker pull your-username/clipilot-registry:latest

# Run it
docker run -d \
  --name clipilot-registry \
  --restart unless-stopped \
  -p 8080:8080 \
  -v /path/to/data:/app/data \
  -e REGISTRY_ADMIN_USER=admin \
  -e REGISTRY_ADMIN_PASS=your_secure_password \
  your-username/clipilot-registry:latest
```

## Using Docker Compose (Optional)

For easier local development:

```bash
# Create .env file
cat > .env << EOF
ADMIN_USER=admin
ADMIN_PASS=your_password
EOF

# Start with compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `REGISTRY_PORT` | `8080` | Port the server listens on |
| `REGISTRY_ADMIN_USER` | `admin` | Admin username for uploads |
| `REGISTRY_ADMIN_PASS` | `changeme` | Admin password (change this!) |
| `REGISTRY_DB_PATH` | `/app/data/registry.db` | SQLite database path |
| `REGISTRY_UPLOAD_DIR` | `/app/data/uploads` | Module upload directory |

### Data Persistence

The registry stores data in `/app/data`:
- `registry.db` - SQLite database with module metadata
- `uploads/` - Uploaded module files

Mount this directory as a volume to persist data:
```bash
-v /path/on/host:/app/data
```

## Production Deployment

For production, consider:

### 1. Run with HTTPS (Nginx Reverse Proxy)

```bash
# Run registry on internal port
docker run -d \
  --name clipilot-registry \
  --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -v /data/registry:/app/data \
  -e REGISTRY_ADMIN_USER=admin \
  -e REGISTRY_ADMIN_PASS=secure_password \
  themobileprof/clipilot-registry:latest

# Setup nginx with Let's Encrypt
# (Use certbot or your preferred SSL tool)
```

### 2. Run Multiple Instances (Load Balancing)

If you need high availability, run multiple containers behind a load balancer (nginx, HAProxy, or cloud load balancer).

### 3. Use Container Orchestration

For larger deployments, consider:

#### Kubernetes

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

#### Cloud Managed Services

- **AWS ECS**: Run as Fargate tasks
- **Google Cloud Run**: Fully managed, auto-scaling
- **Azure Container Instances**: Simple container hosting

See cloud provider docs for specifics.

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
