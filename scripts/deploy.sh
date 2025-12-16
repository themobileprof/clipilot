#!/bin/bash
# Production Deployment Script for CLIPilot Registry
# This script deploys the registry to your production server

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CONTAINER_NAME="clipilot-registry"
IMAGE_NAME="themobileprof/clipilot-registry:latest"
VOLUME_NAME="clipilot-registry-data"

# Check if .env.production exists
if [ ! -f .env.production ]; then
    echo -e "${RED}Error: .env.production file not found!${NC}"
    echo "Please create .env.production with your production configuration."
    echo "See .env.example for reference."
    exit 1
fi

# Load environment variables from .env.production
set -a
source .env.production
set +a

# Validate required environment variables
if [ -z "$ADMIN_PASSWORD" ]; then
    echo -e "${RED}Error: ADMIN_PASSWORD not set in .env.production${NC}"
    exit 1
fi

if [ -z "$BASE_URL" ]; then
    echo -e "${RED}Error: BASE_URL not set in .env.production${NC}"
    exit 1
fi

echo -e "${GREEN}=== CLIPilot Registry Deployment ===${NC}"
echo ""
echo "Configuration:"
echo "  Container: $CONTAINER_NAME"
echo "  Image: $IMAGE_NAME"
echo "  Base URL: $BASE_URL"
echo "  Admin User: ${ADMIN_USER:-admin}"
echo ""

# Pull latest image
echo -e "${YELLOW}Pulling latest Docker image...${NC}"
docker pull $IMAGE_NAME

# Stop and remove existing container (if exists)
if docker ps -a | grep -q $CONTAINER_NAME; then
    echo -e "${YELLOW}Stopping existing container...${NC}"
    docker stop $CONTAINER_NAME 2>/dev/null || true
    echo -e "${YELLOW}Removing existing container...${NC}"
    docker rm $CONTAINER_NAME 2>/dev/null || true
fi

# Create volume if it doesn't exist
if ! docker volume ls | grep -q $VOLUME_NAME; then
    echo -e "${YELLOW}Creating data volume...${NC}"
    docker volume create $VOLUME_NAME
fi

# Deploy new container
echo -e "${YELLOW}Deploying new container...${NC}"
docker run -d \
  --name $CONTAINER_NAME \
  --restart unless-stopped \
  --network bridge \
  -v $VOLUME_NAME:/app/data \
  -e PORT="${PORT:-8080}" \
  -e ADMIN_USER="${ADMIN_USER:-admin}" \
  -e ADMIN_PASSWORD="$ADMIN_PASSWORD" \
  -e BASE_URL="$BASE_URL" \
  -e DATA_DIR="${DATA_DIR:-/app/data}" \
  -e STATIC_DIR="${STATIC_DIR:-/app/static}" \
  -e TEMPLATE_DIR="${TEMPLATE_DIR:-/app/templates}" \
  -e SESSION_SECRET="${SESSION_SECRET:-}" \
  -e SESSION_TIMEOUT="${SESSION_TIMEOUT:-86400}" \
  -e MAX_UPLOAD_SIZE="${MAX_UPLOAD_SIZE:-10485760}" \
  $IMAGE_NAME

# Wait for container to start
echo -e "${YELLOW}Waiting for container to start...${NC}"
sleep 3

# Check if container is running
if docker ps | grep -q $CONTAINER_NAME; then
    echo -e "${GREEN}✓ Container is running${NC}"
    
    # Get container IP for reverse proxy
    CONTAINER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $CONTAINER_NAME)
    echo ""
    echo -e "${GREEN}=== Deployment Successful! ===${NC}"
    echo ""
    echo "Container Details:"
    echo "  Name: $CONTAINER_NAME"
    echo "  IP: $CONTAINER_IP"
    echo "  Internal Port: ${PORT:-8080}"
    echo ""
    echo "Next Steps:"
    echo "  1. Configure your reverse proxy (Nginx/Caddy) to proxy to:"
    echo "     http://$CONTAINER_IP:${PORT:-8080}"
    echo "  2. Or access directly via container name:"
    echo "     http://$CONTAINER_NAME:${PORT:-8080}"
    echo ""
    echo "Nginx Configuration Example:"
    echo "  location / {"
    echo "      proxy_pass http://$CONTAINER_NAME:${PORT:-8080};"
    echo "      proxy_set_header Host \$host;"
    echo "      proxy_set_header X-Real-IP \$remote_addr;"
    echo "      proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;"
    echo "      proxy_set_header X-Forwarded-Proto \$scheme;"
    echo "  }"
    echo ""
    echo "View logs:"
    echo "  docker logs -f $CONTAINER_NAME"
    echo ""
    echo "Access URL: $BASE_URL"
    
else
    echo -e "${RED}✗ Container failed to start${NC}"
    echo ""
    echo "Checking logs..."
    docker logs $CONTAINER_NAME
    exit 1
fi
