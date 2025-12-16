-- CLIPilot Database Schema
-- SQLite3 migration for modular CLI assistant

-- Enable WAL mode for better concurrency and performance
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;

-- Modules metadata table
CREATE TABLE IF NOT EXISTS modules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  description TEXT,
  tags TEXT, -- comma-separated for simple queries
  provides TEXT, -- comma-separated list of capabilities
  requires TEXT, -- comma-separated list of dependencies
  size_kb INTEGER DEFAULT 0,
  installed BOOLEAN DEFAULT 0,
  registry_id INTEGER, -- ID from remote registry (NULL for local-only modules)
  download_url TEXT, -- URL to download module from registry
  author TEXT, -- Module author
  last_synced INTEGER, -- Unix timestamp of last registry sync
  json_content TEXT, -- compressed JSON/YAML of full module definition
  created_at INTEGER DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Normalized steps (cached flow steps for fast access)
CREATE TABLE IF NOT EXISTS steps (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  module_id TEXT NOT NULL,
  flow_name TEXT NOT NULL DEFAULT 'main',
  step_key TEXT NOT NULL,
  type TEXT NOT NULL, -- action, instruction, branch, terminal
  message TEXT,
  command TEXT,
  run_module TEXT, -- for action type
  order_num INTEGER DEFAULT 0,
  next_step TEXT,
  extra_json TEXT, -- conditions, validations, branch maps (JSON)
  FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE
);

-- Intent patterns for keyword-based search
CREATE TABLE IF NOT EXISTS intent_patterns (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  module_id TEXT NOT NULL,
  pattern TEXT NOT NULL, -- keyword or simple phrase
  weight REAL DEFAULT 1.0, -- importance weight for scoring
  pattern_type TEXT DEFAULT 'keyword', -- keyword, command, tag
  FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE
);

-- Module dependencies
CREATE TABLE IF NOT EXISTS dependencies (
  module_id TEXT NOT NULL,
  requires_module_id TEXT NOT NULL,
  required_version TEXT,
  PRIMARY KEY (module_id, requires_module_id),
  FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE
);

-- Runtime state for flow execution and persistence
CREATE TABLE IF NOT EXISTS state (
  key TEXT PRIMARY KEY,
  value TEXT,
  session_id TEXT,
  created_at INTEGER DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Execution logs for telemetry and debugging
CREATE TABLE IF NOT EXISTS logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts INTEGER DEFAULT (strftime('%s', 'now')),
  session_id TEXT,
  input TEXT,
  resolved_module TEXT,
  confidence REAL,
  method TEXT, -- keyword, llm_local, llm_online, manual
  status TEXT, -- started, completed, failed, cancelled
  error_message TEXT,
  duration_ms INTEGER
);

-- Validation results for quality tracking
CREATE TABLE IF NOT EXISTS validations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  log_id INTEGER,
  step_key TEXT,
  validation_type TEXT, -- check_command, parse_output, etc.
  passed BOOLEAN,
  message TEXT,
  ts INTEGER DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (log_id) REFERENCES logs(id) ON DELETE CASCADE
);

-- User settings and preferences
CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT,
  value_type TEXT DEFAULT 'string', -- string, boolean, integer, float
  description TEXT,
  updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Registry cache metadata
CREATE TABLE IF NOT EXISTS registry_cache (
  id INTEGER PRIMARY KEY CHECK (id = 1), -- Single row table
  registry_url TEXT,
  last_sync INTEGER, -- Unix timestamp
  total_modules INTEGER DEFAULT 0,
  cached_modules INTEGER DEFAULT 0,
  sync_status TEXT DEFAULT 'never', -- never, syncing, success, failed
  sync_error TEXT,
  created_at INTEGER DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_modules_installed ON modules(installed);
CREATE INDEX IF NOT EXISTS idx_modules_tags ON modules(tags);
CREATE INDEX IF NOT EXISTS idx_intent_pattern ON intent_patterns(pattern);
CREATE INDEX IF NOT EXISTS idx_intent_module ON intent_patterns(module_id);
CREATE INDEX IF NOT EXISTS idx_steps_module ON steps(module_id);
CREATE INDEX IF NOT EXISTS idx_steps_order ON steps(module_id, order_num);
CREATE INDEX IF NOT EXISTS idx_logs_ts ON logs(ts DESC);
CREATE INDEX IF NOT EXISTS idx_logs_module ON logs(resolved_module);
CREATE INDEX IF NOT EXISTS idx_state_session ON state(session_id);

-- Insert default settings
INSERT OR IGNORE INTO settings (key, value, value_type, description) VALUES
  ('online_mode', 'false', 'boolean', 'Enable online LLM fallback'),
  ('auto_confirm', 'false', 'boolean', 'Auto-confirm safe commands'),
  ('thresh_keyword', '0.6', 'float', 'Confidence threshold for keyword search'),
  ('thresh_llm', '0.6', 'float', 'Confidence threshold for local LLM'),
  ('telemetry_enabled', 'false', 'boolean', 'Anonymous usage statistics'),
  ('color_output', 'false', 'boolean', 'Enable colored terminal output'),
  ('max_history', '1000', 'integer', 'Maximum command history entries'),
  ('db_version', '2', 'integer', 'Database schema version'),
  ('registry_url', '', 'string', 'Module registry server URL (required for registry features)'),
  ('auto_sync', 'false', 'boolean', 'Auto-sync registry on startup'),
  ('sync_interval', '86400', 'integer', 'Registry sync interval in seconds (24h)');

-- Initialize registry cache (empty URL - must be configured)
INSERT OR IGNORE INTO registry_cache (id, registry_url, sync_status) VALUES
  (1, '', 'never');

-- Triggers for updated_at timestamps
CREATE TRIGGER IF NOT EXISTS update_modules_timestamp 
AFTER UPDATE ON modules
BEGIN
  UPDATE modules SET updated_at = strftime('%s', 'now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_state_timestamp 
AFTER UPDATE ON state
BEGIN
  UPDATE state SET updated_at = strftime('%s', 'now') WHERE key = NEW.key;
END;

CREATE TRIGGER IF NOT EXISTS update_settings_timestamp 
AFTER UPDATE ON settings
BEGIN
  UPDATE settings SET updated_at = strftime('%s', 'now') WHERE key = NEW.key;
END;
