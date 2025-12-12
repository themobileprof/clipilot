package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migration.sql
var migrationSQL string

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
	path string
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Open database connection
	// Note: SQLite is embedded in the binary via CGO, no system installation needed
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w\nNote: SQLite is embedded in CLIPilot, but the database file may be corrupted or inaccessible", err)
	}

	// Set connection pool settings for better performance
	conn.SetMaxOpenConns(1) // SQLite works best with single connection
	conn.SetMaxIdleConns(1)

	db := &DB{
		conn: conn,
		path: dbPath,
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	_, err := db.conn.Exec(migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Conn returns the underlying database connection for advanced operations
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// GetSetting retrieves a setting value
func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting %s: %w", key, err)
	}
	return value, nil
}

// SetSetting updates or inserts a setting
func (db *DB) SetSetting(key, value string) error {
	_, err := db.conn.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = strftime('%s', 'now')
	`, key, value, value)
	if err != nil {
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}
	return nil
}

// LogQuery logs an intent query
func (db *DB) LogQuery(sessionID, input, resolvedModule, method string, confidence float64) (int64, error) {
	result, err := db.conn.Exec(`
		INSERT INTO logs (session_id, input, resolved_module, confidence, method, status)
		VALUES (?, ?, ?, ?, ?, 'started')
	`, sessionID, input, resolvedModule, confidence, method)
	if err != nil {
		return 0, fmt.Errorf("failed to log query: %w", err)
	}
	return result.LastInsertId()
}

// UpdateLogStatus updates the status of a log entry
func (db *DB) UpdateLogStatus(logID int64, status, errorMsg string, durationMs int64) error {
	_, err := db.conn.Exec(`
		UPDATE logs SET status = ?, error_message = ?, duration_ms = ?
		WHERE id = ?
	`, status, errorMsg, durationMs, logID)
	if err != nil {
		return fmt.Errorf("failed to update log status: %w", err)
	}
	return nil
}

// SetState saves a state value
func (db *DB) SetState(sessionID, key, value string) error {
	_, err := db.conn.Exec(`
		INSERT INTO state (key, value, session_id) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, session_id = ?, updated_at = strftime('%s', 'now')
	`, key, value, sessionID, value, sessionID)
	if err != nil {
		return fmt.Errorf("failed to set state %s: %w", key, err)
	}
	return nil
}

// GetState retrieves a state value
func (db *DB) GetState(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM state WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get state %s: %w", key, err)
	}
	return value, nil
}

// ClearSession removes all state for a session
func (db *DB) ClearSession(sessionID string) error {
	_, err := db.conn.Exec("DELETE FROM state WHERE session_id = ?", sessionID)
	if err != nil {
		return fmt.Errorf("failed to clear session %s: %w", sessionID, err)
	}
	return nil
}

// Begin starts a transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.conn.Begin()
}
