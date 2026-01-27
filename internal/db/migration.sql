-- CLIPilot Database Schema
-- SQLite3 migration for modular CLI assistant

-- Enable standard journal mode for maximum compatibility (prevents SIGSYS on Termux)
PRAGMA journal_mode=DELETE;
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

-- Available commands on this system (populated via compgen -c and whatis)
CREATE TABLE IF NOT EXISTS commands (
  name TEXT PRIMARY KEY,
  description TEXT,
  has_man BOOLEAN DEFAULT 0,
  has_help BOOLEAN,  -- NULL = unknown, 0 = no, 1 = yes (detected lazily)
  last_seen INTEGER DEFAULT (strftime('%s', 'now')),
  created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Common commands catalog (reference for suggesting installations)
CREATE TABLE IF NOT EXISTS common_commands (
  name TEXT PRIMARY KEY,
  description TEXT NOT NULL,
  category TEXT NOT NULL,  -- development, networking, file-management, database, etc.
  keywords TEXT,  -- comma-separated search keywords
  apt_package TEXT,  -- Debian/Ubuntu package name
  pkg_package TEXT,  -- Termux package name
  dnf_package TEXT,  -- Fedora/RHEL package name
  brew_package TEXT,  -- macOS Homebrew package name
  arch_package TEXT,  -- Arch Linux package name
  alternative_to TEXT,  -- comma-separated list of similar commands
  homepage TEXT,
  priority INTEGER DEFAULT 50  -- Higher = more commonly needed (0-100)
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
CREATE INDEX IF NOT EXISTS idx_commands_name ON commands(name);
CREATE INDEX IF NOT EXISTS idx_common_commands_name ON common_commands(name);
CREATE INDEX IF NOT EXISTS idx_common_commands_category ON common_commands(category);
CREATE INDEX IF NOT EXISTS idx_common_commands_priority ON common_commands(priority DESC);

-- FTS5 Virtual Tables for Enhanced Search
-- We use independent FTS tables populated via triggers for maximum compatibility and performance

-- FTS for system commands
CREATE VIRTUAL TABLE IF NOT EXISTS commands_fts USING fts5(
    name, 
    description, 
    tokenize='porter'
);

-- Triggers to keep commands_fts in sync with commands table
CREATE TRIGGER IF NOT EXISTS commands_ai AFTER INSERT ON commands BEGIN
  INSERT INTO commands_fts(name, description) VALUES (new.name, new.description);
END;

CREATE TRIGGER IF NOT EXISTS commands_ad AFTER DELETE ON commands BEGIN
  INSERT INTO commands_fts(commands_fts, rowid, name, description) VALUES('delete', old.rowid, old.name, old.description);
END;

CREATE TRIGGER IF NOT EXISTS commands_au AFTER UPDATE ON commands BEGIN
  INSERT INTO commands_fts(commands_fts, rowid, name, description) VALUES('delete', old.rowid, old.name, old.description);
  INSERT INTO commands_fts(name, description) VALUES (new.name, new.description);
END;

-- FTS for common commands catalog
CREATE VIRTUAL TABLE IF NOT EXISTS common_commands_fts USING fts5(
    name, 
    description, 
    keywords, 
    category, 
    tokenize='porter'
);

-- Triggers to keep common_commands_fts in sync with common_commands table
CREATE TRIGGER IF NOT EXISTS common_commands_ai AFTER INSERT ON common_commands BEGIN
  INSERT INTO common_commands_fts(name, description, keywords, category) 
  VALUES (new.name, new.description, new.keywords, new.category);
END;

CREATE TRIGGER IF NOT EXISTS common_commands_ad AFTER DELETE ON common_commands BEGIN
  INSERT INTO common_commands_fts(common_commands_fts, rowid, name, description, keywords, category) 
  VALUES('delete', old.rowid, old.name, old.description, old.keywords, old.category);
END;

CREATE TRIGGER IF NOT EXISTS common_commands_au AFTER UPDATE ON common_commands BEGIN
  INSERT INTO common_commands_fts(common_commands_fts, rowid, name, description, keywords, category) 
  VALUES('delete', old.rowid, old.name, old.description, old.keywords, old.category);
  INSERT INTO common_commands_fts(name, description, keywords, category) 
  VALUES (new.name, new.description, new.keywords, new.category);
END;

-- Embedding tables for semantic search (Layer 2)
-- Pre-computed embeddings are cached here for fast startup

-- Module embeddings for semantic intent matching
CREATE TABLE IF NOT EXISTS module_embeddings (
  module_id TEXT PRIMARY KEY,
  embedding BLOB NOT NULL,  -- JSON-encoded float32 array (384 dimensions)
  text_hash TEXT,  -- Hash of text used to generate embedding (for invalidation)
  updated_at INTEGER DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (module_id) REFERENCES modules(id) ON DELETE CASCADE
);

-- Command embeddings for semantic command search
CREATE TABLE IF NOT EXISTS command_embeddings (
  command_name TEXT PRIMARY KEY,
  embedding BLOB NOT NULL,  -- JSON-encoded float32 array (384 dimensions)
  text_hash TEXT,  -- Hash of text used to generate embedding (for invalidation)
  updated_at INTEGER DEFAULT (strftime('%s', 'now')),
  FOREIGN KEY (command_name) REFERENCES commands(name) ON DELETE CASCADE
);

-- Semantic model metadata
CREATE TABLE IF NOT EXISTS semantic_model (
  id INTEGER PRIMARY KEY CHECK (id = 1),  -- Single row table
  model_name TEXT NOT NULL DEFAULT 'all-MiniLM-L6-v2',
  model_version TEXT,
  embedding_dim INTEGER DEFAULT 384,
  last_loaded INTEGER,
  embeddings_computed INTEGER DEFAULT 0,
  total_modules INTEGER DEFAULT 0,
  total_commands INTEGER DEFAULT 0,
  created_at INTEGER DEFAULT (strftime('%s', 'now')),
  updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Initialize semantic model metadata
INSERT OR IGNORE INTO semantic_model (id, model_name, embedding_dim) VALUES
  (1, 'all-MiniLM-L6-v2', 384);

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
  ('registry_url', 'https://clipilot.themobileprof.com', 'string', 'Module registry server URL'),
  ('auto_sync', 'false', 'boolean', 'Auto-sync registry on startup'),
  ('sync_interval', '86400', 'integer', 'Registry sync interval in seconds (24h)'),
  ('commands_indexed', 'false', 'boolean', 'Whether system commands have been indexed');

-- Initialize registry cache with default production URL
INSERT OR IGNORE INTO registry_cache (id, registry_url, sync_status) VALUES
  (1, 'https://clipilot.themobileprof.com', 'never');

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
