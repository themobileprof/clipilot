#!/bin/bash
# Installs a git pre-commit hook to verify tests pass before allowing a commit.

HOOK_DIR=".git/hooks"
HOOK_FILE="$HOOK_DIR/pre-commit"

if [ ! -d ".git" ]; then
    echo "Error: .git directory not found. Are you in the root of the repository?"
    exit 1
fi

mkdir -p "$HOOK_DIR"

echo "Installing pre-commit hook to $HOOK_FILE..."

cat > "$HOOK_FILE" << 'EOF'
#!/bin/bash
# Pre-commit hook to run tests

echo "========================================================"
echo "🤖 CLIPilot Pre-Commit Check"
echo "========================================================"

# Check if tests script exists
if [ -f "./scripts/test.sh" ]; then
    echo "Running test suite..."
    if ! ./scripts/test.sh; then
        echo "❌ Tests failed! Commit aborted."
        echo "Please fix the errors and try again."
        exit 1
    fi
else
    echo "⚠️  scripts/test.sh not found. Skipping tests."
fi

echo "✅ Tests passed. Proceeding with commit."
EOF

chmod +x "$HOOK_FILE"

echo "✓ Hook installed successfully."
echo "Tests will now strictly run before every commit."
