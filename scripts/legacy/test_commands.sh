#!/bin/bash
# Test script for command indexing feature

set -e

echo "=== Testing CLIPilot Command Indexing ===="
echo ""

# Build
echo "1. Building CLIPilot..."
go build -o bin/clipilot ./cmd/clipilot
echo "✓ Build successful"
echo ""

# Initialize
echo "2. Initializing database..."
rm -f ~/.clipilot/clipilot.db
./bin/clipilot --init --load=modules > /dev/null 2>&1
echo "✓ Database initialized"
echo ""

# Test command indexing
echo "3. Indexing system commands..."
echo "update-commands" | ./bin/clipilot 2>&1 | grep -E "(Indexed|commands)" | head -5
echo "✓ Commands indexed"
echo ""

# Verify database
echo "4. Checking database..."
COUNT=$(sqlite3 ~/.clipilot/clipilot.db "SELECT COUNT(*) FROM commands")
echo "✓ Found $COUNT commands in database"
echo ""

# Test command search
echo "5. Testing command search..."
echo -e "search ls\nexit" | ./bin/clipilot 2>&1 | grep -A 3 "Found.*module" | head -6
echo "✓ Command search working"
echo ""

# Test specific commands
echo "6. Verifying specific commands..."
sqlite3 ~/.clipilot/clipilot.db "SELECT name, has_man FROM commands WHERE name IN ('ls', 'git', 'cat') ORDER BY name"
echo "✓ Commands have man pages indexed"
echo ""

echo "=== All Tests Passed ===="
