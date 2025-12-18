#!/bin/bash
# Upload all modules to the registry
# Usage: ./scripts/upload-modules.sh [registry-url] [username] [password]

set -e

REGISTRY_URL="${1:-https://clipilot.themobileprof.com}"
USERNAME="${2:-admin}"
PASSWORD="${3}"

if [ -z "$PASSWORD" ]; then
    echo "Usage: $0 [registry-url] [username] [password]"
    echo "Example: $0 https://clipilot.themobileprof.com admin mypassword"
    exit 1
fi

echo "Uploading modules to $REGISTRY_URL"
echo "Username: $USERNAME"
echo ""

# Login and get session cookie
echo "Logging in..."
COOKIE_JAR=$(mktemp)
LOGIN_RESPONSE=$(curl -s -c "$COOKIE_JAR" -X POST \
    -d "username=$USERNAME&password=$PASSWORD" \
    "$REGISTRY_URL/login")

# Check if login was successful by trying to access a protected page
if ! curl -s -b "$COOKIE_JAR" "$REGISTRY_URL/upload" | grep -q "Upload Module"; then
    echo "❌ Login failed. Please check your credentials."
    rm "$COOKIE_JAR"
    exit 1
fi

echo "✓ Logged in successfully"
echo ""

# Count total modules
TOTAL=$(ls -1 modules/*.yaml 2>/dev/null | wc -l)
CURRENT=0
SUCCESS=0
FAILED=0

echo "Found $TOTAL modules to upload"
echo ""

# Upload each module
for module in modules/*.yaml; do
    CURRENT=$((CURRENT + 1))
    MODULE_NAME=$(basename "$module")
    
    printf "[%d/%d] Uploading %s... " "$CURRENT" "$TOTAL" "$MODULE_NAME"
    
    RESPONSE=$(curl -s -b "$COOKIE_JAR" -X POST \
        -F "file=@$module" \
        "$REGISTRY_URL/api/upload")
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        echo "✓"
        SUCCESS=$((SUCCESS + 1))
    else
        echo "✗"
        echo "  Error: $RESPONSE"
        FAILED=$((FAILED + 1))
    fi
done

# Cleanup
rm "$COOKIE_JAR"

echo ""
echo "========================================"
echo "Upload Summary:"
echo "  Total: $TOTAL"
echo "  Success: $SUCCESS"
echo "  Failed: $FAILED"
echo "========================================"

if [ $FAILED -eq 0 ]; then
    echo "✓ All modules uploaded successfully!"
    exit 0
else
    echo "⚠️  Some modules failed to upload"
    exit 1
fi
