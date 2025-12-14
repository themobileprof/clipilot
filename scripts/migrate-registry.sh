#!/bin/bash
# Migration script to add registry support to existing clipilot database

DB_PATH="${1:-$HOME/.clipilot/clipilot.db}"

echo "Migrating CLIPilot database: $DB_PATH"
echo "==========================================="

sqlite3 "$DB_PATH" << 'EOF'
-- Check if columns already exist
.mode column

-- Add new columns to modules table (will error if already exists, that's OK)
ALTER TABLE modules ADD COLUMN registry_id INTEGER;
ALTER TABLE modules ADD COLUMN download_url TEXT;
ALTER TABLE modules ADD COLUMN author TEXT;
ALTER TABLE modules ADD COLUMN last_synced INTEGER;

-- Create registry_cache table if it doesn't exist
CREATE TABLE IF NOT EXISTS registry_cache (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  registry_url TEXT,
  last_sync INTEGER,
  total_modules INTEGER DEFAULT 0,
  cached_modules INTEGER DEFAULT 0,
  sync_status TEXT DEFAULT 'never',
  sync_error TEXT,
  created_at INTEGER DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Insert default registry settings
INSERT OR IGNORE INTO settings (key, value, value_type, description) VALUES
  ('registry_url', 'http://localhost:8080', 'string', 'Module registry server URL'),
  ('auto_sync', 'false', 'boolean', 'Auto-sync registry on startup'),
  ('sync_interval', '86400', 'integer', 'Registry sync interval in seconds (24h)');

-- Initialize registry cache
INSERT OR IGNORE INTO registry_cache (id, registry_url, sync_status) VALUES
  (1, 'http://localhost:8080', 'never');

-- Update db_version
UPDATE settings SET value = '2' WHERE key = 'db_version';

SELECT '✓ Migration completed successfully!';
EOF

echo ""
echo "✓ Database migration complete!"
echo ""
echo "Next steps:"
echo "  1. Run 'clipilot' to start the interactive assistant"
echo "  2. Run 'sync' to fetch available modules from registry"
echo "  3. Run 'modules list --all' to see all modules"
