#!/bin/bash
# CLIPilot Registry - Admin User Bootstrap Script
# Creates an admin user and generates an API key for CI/CD integration

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
DB_PATH="${DB_PATH:-./data/registry.db}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@localhost}"

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}  CLIPilot Registry - Admin Setup${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    echo -e "${RED}Error: Database not found at $DB_PATH${NC}"
    echo -e "${YELLOW}Make sure the registry server has been started at least once.${NC}"
    exit 1
fi

# Check if sqlite3 is installed
if ! command -v sqlite3 &> /dev/null; then
    echo -e "${RED}Error: sqlite3 command not found${NC}"
    echo -e "${YELLOW}Please install SQLite3:${NC}"
    echo "  Ubuntu/Debian: apt-get install sqlite3"
    echo "Alpine: apk add sqlite"
    echo "  macOS: brew install sqlite3"
    exit 1
fi

echo -e "${YELLOW}Database:${NC} $DB_PATH"
echo -e "${YELLOW}Username:${NC} $ADMIN_USERNAME"
echo -e "${YELLOW}Email:${NC} $ADMIN_EMAIL"
echo ""

# Prompt for password
read -sp "$(echo -e ${YELLOW}Enter admin password:${NC} )" ADMIN_PASSWORD
echo ""

if [ -z "$ADMIN_PASSWORD" ]; then
    echo -e "${RED}Error: Password cannot be empty${NC}"
    exit 1
fi

read -sp "$(echo -e ${YELLOW}Confirm password:${NC} )" ADMIN_PASSWORD_CONFIRM
echo ""

if [ "$ADMIN_PASSWORD" != "$ADMIN_PASSWORD_CONFIRM" ]; then
    echo -e "${RED}Error: Passwords do not match${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}Creating admin user...${NC}"

# Generate password hash (simple SHA-256 for now - in production use bcrypt)
# Note: The server's auth system expects SHA-256 for backward compatibility
PASSWORD_HASH=$(echo -n "$ADMIN_PASSWORD" | sha256sum | cut -d' ' -f1)

# Check if user already exists
EXISTING_USER=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM users WHERE username = '$ADMIN_USERNAME'" 2>/dev/null || echo "0")

if [ "$EXISTING_USER" != "0" ]; then
    echo -e "${YELLOW}Warning: User '$ADMIN_USERNAME' already exists${NC}"
    read -p "$(echo -e ${YELLOW}Update password? [y/N]:${NC} )" -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}Skipping user creation.${NC}"
    else
        # Update existing user
        sqlite3 "$DB_PATH" <<EOF
UPDATE users 
SET password_hash = '$PASSWORD_HASH', 
    email = '$ADMIN_EMAIL',
    role = 'admin',
    updated_at = CURRENT_TIMESTAMP
WHERE username = '$ADMIN_USERNAME';
EOF
        echo -e "${GREEN}✓ Admin user updated${NC}"
    fi
else
    # Insert new admin user
    sqlite3 "$DB_PATH" <<EOF
INSERT INTO users (username, email, password_hash, github_id, avatar_url, role, created_at, updated_at)
VALUES ('$ADMIN_USERNAME', '$ADMIN_EMAIL', '$PASSWORD_HASH', NULL, NULL, 'admin', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
EOF
    echo -e "${GREEN}✓ Admin user created${NC}"
fi

# Get user ID
USER_ID=$(sqlite3 "$DB_PATH" "SELECT id FROM users WHERE username = '$ADMIN_USERNAME'")

echo ""
echo -e "${GREEN}Generating API key for CI/CD...${NC}"

# Generate a random API key (32 bytes = 64 hex chars)
API_KEY="clipilot_$(openssl rand -hex 32)"

# Hash the API key for storage
API_KEY_HASH=$(echo -n "$API_KEY" | sha256sum | cut -d' ' -f1)

# Insert API key
API_KEY_NAME="ci-cd-key"
sqlite3 "$DB_PATH" <<EOF
INSERT INTO api_keys (user_id, key_hash, name, scopes, expires_at, revoked, created_at)
VALUES ($USER_ID, '$API_KEY_HASH', '$API_KEY_NAME', '["upload", "admin"]', NULL, 0, CURRENT_TIMESTAMP);
EOF

echo -e "${GREEN}✓ API key generated${NC}"

echo ""
echo -e "${BLUE}======================================${NC}"
echo -e "${GREEN}Admin User Created Successfully!${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo -e "${YELLOW}Username:${NC} $ADMIN_USERNAME"
echo -e "${YELLOW}Email:${NC} $ADMIN_EMAIL"
echo -e "${YELLOW}Password:${NC} [hidden]"
echo ""
echo -e "${YELLOW}API Key (for CI/CD):${NC}"
echo -e "${GREEN}$API_KEY${NC}"
echo ""
echo -e "${BLUE}Usage Examples:${NC}"
echo ""
echo -e "${YELLOW}1. Login to web UI:${NC}"
echo "   https://clipilot.themobileprof.com/login"
echo "   Username: $ADMIN_USERNAME"
echo "   Password: [your password]"
echo ""
echo -e "${YELLOW}2. Upload install script via API:${NC}"
echo "   curl -X POST https://clipilot.themobileprof.com/api/install-script/upload \\"
echo "     -H \"Authorization: Bearer $API_KEY\" \\"
echo "     -F \"file=@install.sh\" \\"
echo "     -F \"version=v1.0.0\""
echo ""
echo -e "${YELLOW}3. Store in GitHub Secrets (for Clio CI/CD):${NC}"
echo "   Name: CLIPILOT_API_KEY"
echo "   Value: $API_KEY"
echo ""
echo -e "${RED}⚠️  IMPORTANT: Save this API key securely!${NC}"
echo -e "${RED}   It will not be shown again.${NC}"
echo ""
