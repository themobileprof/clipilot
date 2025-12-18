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

# Check if registry is reachable
echo "Testing registry connectivity..."
if ! curl -f -s -m 5 "$REGISTRY_URL/api/modules" >/dev/null 2>&1; then
    echo "❌ Cannot reach registry at $REGISTRY_URL"
    echo "Please check:"
    echo "  - Is the URL correct?"
    echo "  - Is the registry server running?"
    echo "  - Do you have network connectivity?"
    exit 1
fi
echo "✓ Registry is reachable"
echo ""

# Login and get session cookie
echo "Logging in..."
COOKIE_JAR=$(mktemp)
echo "DEBUG: Cookie jar: $COOKIE_JAR"

LOGIN_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -c "$COOKIE_JAR" -X POST \
    -d "username=$USERNAME&password=$PASSWORD" \
    "$REGISTRY_URL/login")

HTTP_CODE=$(echo "$LOGIN_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
echo "DEBUG: Login HTTP code: $HTTP_CODE"
echo "DEBUG: Login response: $(echo "$LOGIN_RESPONSE" | grep -v "HTTP_CODE:")"

if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "302" ] && [ "$HTTP_CODE" != "303" ]; then
    echo "❌ Login failed. HTTP code: $HTTP_CODE"
    rm "$COOKIE_JAR"
    exit 1
fi

# Check if login was successful by trying to access a protected page
UPLOAD_PAGE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -b "$COOKIE_JAR" "$REGISTRY_URL/upload")
UPLOAD_HTTP_CODE=$(echo "$UPLOAD_PAGE" | grep "HTTP_CODE:" | cut -d: -f2)
echo "DEBUG: Upload page HTTP code: $UPLOAD_HTTP_CODE"

if [ "$UPLOAD_HTTP_CODE" != "200" ]; then
    echo "❌ Login failed. Cannot access protected page. HTTP code: $UPLOAD_HTTP_CODE"
    echo "Please check your credentials."
    rm "$COOKIE_JAR"
    exit 1
fi

if ! echo "$UPLOAD_PAGE" | grep -q "Upload"; then
    echo "❌ Login failed. Page doesn't contain expected content."
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
    
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -b "$COOKIE_JAR" -X POST \
        -F "module=@$module" \
        "$REGISTRY_URL/api/upload")
    
    UPLOAD_HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    RESPONSE_BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")
    
    if [ "$UPLOAD_HTTP_CODE" = "200" ] || [ "$UPLOAD_HTTP_CODE" = "201" ]; then
        if echo "$RESPONSE_BODY" | grep -q '"success":true'; then
            echo "✓"
            SUCCESS=$((SUCCESS + 1))
        else
            echo "✗"
            echo "  Response: $RESPONSE_BODY"
            FAILED=$((FAILED + 1))
        fi
    else
        echo "✗ (HTTP $UPLOAD_HTTP_CODE)"
        echo "  Response: $RESPONSE_BODY"
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
