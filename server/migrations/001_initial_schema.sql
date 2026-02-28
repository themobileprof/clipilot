-- Registry database schema

CREATE TABLE IF NOT EXISTS modules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    description TEXT,
    author TEXT,
    tags TEXT, -- JSON array of tags
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    uploaded_by TEXT NOT NULL, -- Legacy: username string
    user_id INTEGER, -- New: foreign key to users table
    github_user TEXT, -- GitHub username if uploaded via GitHub OAuth
    file_path TEXT NOT NULL,
    original_filename TEXT,
    downloads INTEGER DEFAULT 0,
    UNIQUE(name, version),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
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

-- Users table for persistent authentication and RBAC
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT, -- bcrypt hash, NULL for OAuth-only users
    github_id TEXT UNIQUE, -- GitHub user ID for OAuth
    avatar_url TEXT,  
    role TEXT NOT NULL CHECK(role IN ('admin', 'contributor', 'user')) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- API keys for programmatic access (CI/CD, automation)
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    key_hash TEXT UNIQUE NOT NULL, -- bcrypt hash of the API key
    name TEXT NOT NULL, -- User-friendly name for the key
    scopes TEXT NOT NULL, -- JSON array of scopes: ["module:upload", "install:upload"]
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP, -- NULL for never expires
    last_used_at TIMESTAMP,
    revoked BOOLEAN DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_revoked ON api_keys(revoked);

-- Sessions for web authentication (database-backed for multi-instance deployment)
CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash TEXT UNIQUE NOT NULL, -- bcrypt hash of session token
    user_id INTEGER NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Install scripts for Clio (uploaded by Clio CI/CD)
CREATE TABLE IF NOT EXISTS install_scripts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version TEXT NOT NULL,
    file_path TEXT NOT NULL,
    checksum_sha256 TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    uploaded_by INTEGER NOT NULL, -- user_id
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1, -- Only one should be active at a time
    FOREIGN KEY (uploaded_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_install_scripts_is_active ON install_scripts(is_active);
CREATE INDEX IF NOT EXISTS idx_install_scripts_uploaded_at ON install_scripts(uploaded_at DESC);
