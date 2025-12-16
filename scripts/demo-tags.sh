#!/bin/bash
# Demonstration of Tag-Based Module Queries and Downloads
# Shows the power of the enhanced tagging system
# Set REGISTRY_URL environment variable or it will fail

REGISTRY_URL="${REGISTRY_URL:-}"
COOKIES="${COOKIES:-/tmp/cookies.txt}"

if [ -z "$REGISTRY_URL" ]; then
    echo "Error: REGISTRY_URL environment variable is required"
    echo "Example: export REGISTRY_URL=http://localhost:8082"
    echo "         export REGISTRY_URL=https://registry.yourdomain.com"
    exit 1
fi

echo "================================================"
echo "CLIPilot Module Tag System Demonstration"
echo "================================================"
echo ""

# 1. Show all atomic file operations
echo "1. Atomic File Operations (building blocks)"
echo "   Tags: file-ops + atomic"
echo "   ---"
./scripts/query-modules.sh query file-ops atomic | jq -r '"\(.name) v\(.version) - \(.description)"' | head -10
echo "   ... ($(./scripts/query-modules.sh query file-ops atomic | jq -s 'length') total)"
echo ""

# 2. Show all DevOps operations
echo "2. DevOps Operations (for SREs and DevOps engineers)"
echo "   Tags: devops-ops"
echo "   ---"
./scripts/query-modules.sh devops-ops | jq -r '"\(.name) v\(.version) - \(.description)"'
echo ""

# 3. Show installer modules
echo "3. Installation & Setup Modules"
echo "   Tags: installer"
echo "   ---"
./scripts/query-modules.sh installer | jq -r '"\(.name) v\(.version) - \(.description)"'
echo ""

# 4. Show security-related modules
echo "4. Security-Related Modules"
echo "   Tags: security"
echo "   ---"
./scripts/query-modules.sh security | jq -r '"\(.name) v\(.version) - \(.description)"'
echo ""

# 5. Show monitoring modules
echo "5. Monitoring & Diagnostics Modules"
echo "   Tags: monitoring"
echo "   ---"
./scripts/query-modules.sh monitoring | jq -r '"\(.name) v\(.version) - \(.description)"'
echo ""

# 6. Show backup-related modules
echo "6. Backup & Archive Modules"
echo "   Tags: backup"
echo "   ---"
./scripts/query-modules.sh backup | jq -r '"\(.name) v\(.version) - \(.description)"'
echo ""

# 7. Show text processing modules
echo "7. Text Processing Modules"
echo "   Tags: text-ops"
echo "   ---"
./scripts/query-modules.sh text-ops | jq -r '"\(.name) v\(.version) - \(.description)"'
echo ""

echo "================================================"
echo "Tag Combinations (Advanced Queries)"
echo "================================================"
echo ""

echo "8. Composite DevOps Monitoring Tools"
echo "   Tags: devops-ops + composite + monitoring"
echo "   ---"
./scripts/query-modules.sh query devops-ops composite monitoring | jq -r '"\(.name) v\(.version) - \(.description)"'
echo ""

echo "9. Atomic System Operations"
echo "   Tags: system-ops + atomic"
echo "   ---"
./scripts/query-modules.sh query system-ops atomic | jq -r '"\(.name) v\(.version)"'
echo ""

echo "10. Development Environment Installers"
echo "    Tags: installer + development"
echo "    ---"
./scripts/query-modules.sh query installer development | jq -r '"\(.name) v\(.version) - \(.description)"'
echo ""

echo "================================================"
echo "Summary Statistics"
echo "================================================"
./scripts/query-modules.sh summary
echo ""
echo "✓ Tag system enables efficient module discovery"
echo "✓ Download related modules together instead of one-by-one"
echo "✓ Create custom bundles for specific use cases"
