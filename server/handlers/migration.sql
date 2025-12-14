-- Registry database schema

CREATE TABLE IF NOT EXISTS modules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    author TEXT,
    tags TEXT, -- JSON array of tags
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    uploaded_by TEXT NOT NULL,
    file_path TEXT NOT NULL,
    original_filename TEXT,
    downloads INTEGER DEFAULT 0,
    UNIQUE(name, version)
);

CREATE INDEX IF NOT EXISTS idx_modules_name ON modules(name);
CREATE INDEX IF NOT EXISTS idx_modules_uploaded_by ON modules(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_modules_uploaded_at ON modules(uploaded_at DESC);
