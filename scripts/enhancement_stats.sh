#!/bin/bash
# View enhancement statistics

DB_PATH="${1:-data/registry.db}"

echo "=== Enhancement Statistics ==="
echo ""

# Total commands
TOTAL=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM enhanced_commands")
echo "Total Enhanced Commands: $TOTAL"

# By category
echo ""
echo "By Category:"
sqlite3 "$DB_PATH" "SELECT category, COUNT(*) as count FROM enhanced_commands WHERE category IS NOT NULL GROUP BY category ORDER BY count DESC" | while IFS='|' read -r category count; do
  echo "  $category: $count"
done

# Recently enhanced
echo ""
echo "Recently Enhanced (last 10):"
sqlite3 "$DB_PATH" "SELECT name, datetime(last_enhanced, 'unixepoch'), version FROM enhanced_commands ORDER BY last_enhanced DESC LIMIT 10" | while IFS='|' read -r name timestamp version; do
  echo "  $name (v$version) - $timestamp"
done

# Unprocessed submissions
UNPROCESSED=$(sqlite3 "$DB_PATH" "SELECT COUNT(DISTINCT command_name) FROM command_submissions cs LEFT JOIN enhanced_commands ec ON cs.command_name = ec.name WHERE cs.processed = 0 AND ec.name IS NULL")
echo ""
echo "Unprocessed Submissions (needs enhancement): $UNPROCESSED"

# Top unprocessed by submission count
if [ "$UNPROCESSED" -gt 0 ]; then
  echo ""
  echo "Top Unprocessed Commands (by submission count):"
  sqlite3 "$DB_PATH" "SELECT cs.command_name, COUNT(*) as submissions FROM command_submissions cs LEFT JOIN enhanced_commands ec ON cs.command_name = ec.name WHERE cs.processed = 0 AND ec.name IS NULL GROUP BY cs.command_name ORDER BY submissions DESC LIMIT 10" | while IFS='|' read -r name count; do
    echo "  $name: $count submissions"
  done
fi

echo ""
