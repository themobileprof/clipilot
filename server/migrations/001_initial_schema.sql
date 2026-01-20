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
    github_user TEXT, -- GitHub username if uploaded via GitHub OAuth
    file_path TEXT NOT NULL,
    original_filename TEXT,
    downloads INTEGER DEFAULT 0,
    UNIQUE(name, version)
);

CREATE INDEX IF NOT EXISTS idx_modules_name ON modules(name);
CREATE INDEX IF NOT EXISTS idx_modules_uploaded_by ON modules(uploaded_by);
CREATE INDEX IF NOT EXISTS idx_modules_github_user ON modules(github_user);
CREATE INDEX IF NOT EXISTS idx_modules_uploaded_at ON modules(uploaded_at DESC);

-- Module requests from users (when no matching module exists)
CREATE TABLE IF NOT EXISTS module_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    query TEXT NOT NULL,
    user_context TEXT, -- Optional: OS, device info, etc.
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'pending', -- pending, in_progress, completed, duplicate
    duplicate_of INTEGER, -- ID of the original request if this is a duplicate
    notes TEXT, -- Admin notes about the request
    fulfilled_by_module TEXT, -- Module name that fulfills this request
    FOREIGN KEY (duplicate_of) REFERENCES module_requests(id)
);

CREATE INDEX IF NOT EXISTS idx_module_requests_created_at ON module_requests(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_module_requests_status ON module_requests(status);
CREATE INDEX IF NOT EXISTS idx_module_requests_query ON module_requests(query);
