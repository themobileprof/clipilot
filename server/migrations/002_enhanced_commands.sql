-- Enhanced command descriptions table
-- Server maintains high-quality, AI-enhanced descriptions
CREATE TABLE IF NOT EXISTS enhanced_commands (
  name TEXT PRIMARY KEY,
  description TEXT NOT NULL,
  enhanced_description TEXT, -- AI-enhanced with keywords
  keywords TEXT, -- Comma-separated searchable keywords
  category TEXT,
  use_cases TEXT, -- Comma-separated common use cases
  source TEXT DEFAULT 'community', -- community, ai, manual
  version INTEGER DEFAULT 1,
  last_enhanced INTEGER, -- Unix timestamp
  enhancement_model TEXT, -- e.g., 'gemini-1.5-flash'
  created_at INTEGER DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Command submissions from users (for discovery)
CREATE TABLE IF NOT EXISTS command_submissions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  command_name TEXT NOT NULL,
  user_description TEXT, -- From user's man page
  submitted_by TEXT, -- User identifier
  submitted_at INTEGER DEFAULT (strftime('%s', 'now')),
  processed BOOLEAN DEFAULT 0,
  UNIQUE(command_name, user_description)
);

-- Index for quick lookups
CREATE INDEX IF NOT EXISTS idx_enhanced_commands_name ON enhanced_commands(name);
CREATE INDEX IF NOT EXISTS idx_command_submissions_processed ON command_submissions(processed);
