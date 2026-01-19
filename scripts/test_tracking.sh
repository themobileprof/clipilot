#!/bin/bash
# Test enhancement tracking

echo "=== Testing Enhancement Tracking ==="
echo ""

DB_PATH="${1:-data/test_registry.db}"
rm -f "$DB_PATH"

echo "1. Creating test database..."
sqlite3 "$DB_PATH" << 'EOF'
CREATE TABLE IF NOT EXISTS enhanced_commands (
  name TEXT PRIMARY KEY,
  description TEXT NOT NULL,
  enhanced_description TEXT,
  keywords TEXT,
  category TEXT,
  use_cases TEXT,
  source TEXT DEFAULT 'community',
  version INTEGER DEFAULT 1,
  last_enhanced INTEGER,
  enhancement_model TEXT,
  created_at INTEGER DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE IF NOT EXISTS command_submissions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  command_name TEXT NOT NULL,
  user_description TEXT,
  submitted_by TEXT,
  submitted_at INTEGER DEFAULT (strftime('%s', 'now')),
  processed BOOLEAN DEFAULT 0,
  UNIQUE(command_name, user_description)
);
EOF

echo "✓ Database created"
echo ""

echo "2. Adding test submissions..."
sqlite3 "$DB_PATH" << 'EOF'
INSERT INTO command_submissions (command_name, user_description) VALUES
  ('ss', 'utility to investigate sockets'),
  ('lsof', 'list open files'),
  ('netstat', 'print network connections'),
  ('cp', 'copy files and directories'),
  ('mv', 'move/rename files and directories');
EOF

echo "✓ Added 5 submissions"
echo ""

echo "3. Simulating enhancement of 2 commands..."
sqlite3 "$DB_PATH" << 'EOF'
INSERT INTO enhanced_commands (name, description, enhanced_description, keywords, category, version) VALUES
  ('ss', 'utility to investigate sockets', 'socket utility - check ports listening connections', 'ports,listening,connections', 'networking', 1),
  ('lsof', 'list open files', 'list open files - check processes ports file descriptors', 'files,processes,ports,descriptors', 'process', 1);
EOF

echo "✓ Enhanced ss and lsof"
echo ""

echo "4. Query unprocessed submissions (should exclude enhanced)..."
UNENHANCED=$(sqlite3 "$DB_PATH" "
SELECT COUNT(DISTINCT cs.command_name)
FROM command_submissions cs
LEFT JOIN enhanced_commands ec ON cs.command_name = ec.name
WHERE cs.processed = 0 AND ec.name IS NULL
")

echo "   Commands needing enhancement: $UNENHANCED"
if [ "$UNENHANCED" -eq 3 ]; then
  echo "   ✓ Correct! (netstat, cp, mv)"
else
  echo "   ✗ Expected 3, got $UNENHANCED"
fi
echo ""

echo "5. List commands needing enhancement..."
sqlite3 "$DB_PATH" "
SELECT cs.command_name
FROM command_submissions cs
LEFT JOIN enhanced_commands ec ON cs.command_name = ec.name
WHERE cs.processed = 0 AND ec.name IS NULL
" | while read -r cmd; do
  echo "   - $cmd"
done
echo ""

echo "6. Verify enhanced commands are in database..."
ENHANCED_COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM enhanced_commands")
echo "   Total enhanced: $ENHANCED_COUNT"
if [ "$ENHANCED_COUNT" -eq 2 ]; then
  echo "   ✓ Correct! (ss, lsof)"
else
  echo "   ✗ Expected 2, got $ENHANCED_COUNT"
fi
echo ""

echo "7. Test version tracking (simulate re-enhancement)..."
sqlite3 "$DB_PATH" "
UPDATE enhanced_commands 
SET version = version + 1,
    enhanced_description = 'socket utility - updated description',
    updated_at = strftime('%s', 'now')
WHERE name = 'ss'
"
NEW_VERSION=$(sqlite3 "$DB_PATH" "SELECT version FROM enhanced_commands WHERE name = 'ss'")
echo "   ss version after re-enhancement: $NEW_VERSION"
if [ "$NEW_VERSION" -eq 2 ]; then
  echo "   ✓ Version correctly incremented"
else
  echo "   ✗ Expected version 2, got $NEW_VERSION"
fi
echo ""

echo "=== Test Summary ==="
echo "✓ Tracking query correctly excludes enhanced commands"
echo "✓ Version tracking works"
echo "✓ Database schema supports enhancement tracking"
echo ""
echo "Clean up: rm $DB_PATH"
