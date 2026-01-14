package commands

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Indexer handles command discovery and indexing
type Indexer struct {
	db *sql.DB
}

// NewIndexer creates a new command indexer
func NewIndexer(db *sql.DB) *Indexer {
	return &Indexer{db: db}
}

// RefreshCommandIndex discovers all available commands and indexes them with descriptions
func (idx *Indexer) RefreshCommandIndex() error {
	fmt.Println("🔍 Discovering available commands...")

	// Get all commands using compgen -c
	commands, err := idx.discoverCommands()
	if err != nil {
		return fmt.Errorf("failed to discover commands: %w", err)
	}

	if len(commands) == 0 {
		return fmt.Errorf("no commands found")
	}

	fmt.Printf("📦 Found %d commands, indexing descriptions...\n", len(commands))

	// Begin transaction for batch insert
	tx, err := idx.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statement for batch inserts
	stmt, err := tx.Prepare(`
		INSERT INTO commands (name, description, has_man, last_seen)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			description = excluded.description,
			has_man = excluded.has_man,
			last_seen = excluded.last_seen
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Index commands with progress feedback
	indexed := 0
	timestamp := time.Now().Unix()
	lastProgress := 0

	for _, cmdName := range commands {
		// Get description from whatis
		description, hasMan := idx.getCommandDescription(cmdName)

		// Insert into database
		_, err := stmt.Exec(cmdName, description, hasMan, timestamp)
		if err != nil {
			// Log but continue on individual command errors
			continue
		}

		indexed++

		// Show progress every 10%
		progress := (indexed * 100) / len(commands)
		if progress >= lastProgress+10 {
			fmt.Printf("  %d%% (%d/%d)...\n", progress, indexed, len(commands))
			lastProgress = progress
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update settings
	_, err = idx.db.Exec(`
		INSERT INTO settings (key, value, value_type, description)
		VALUES ('commands_indexed', 'true', 'boolean', 'Whether system commands have been indexed')
		ON CONFLICT(key) DO UPDATE SET value = 'true', updated_at = strftime('%s', 'now')
	`)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	fmt.Printf("✅ Indexed %d commands successfully!\n", indexed)
	return nil
}

// discoverCommands returns all executable commands using compgen -c
func (idx *Indexer) discoverCommands() ([]string, error) {
	// Use bash to run compgen -c (bash builtin)
	cmd := exec.Command("bash", "-c", "compgen -c | sort -u")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("compgen failed: %w", err)
	}

	// Parse output into slice
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Filter out empty lines and aliases/functions
	commands := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip bash builtins that are not useful (like '[', ':', etc.)
		if len(line) == 1 && !isAlphaNumeric(line) {
			continue
		}
		commands = append(commands, line)
	}

	return commands, nil
}

// getCommandDescription fetches description from whatis
func (idx *Indexer) getCommandDescription(cmdName string) (string, bool) {
	// Try whatis first
	cmd := exec.Command("whatis", cmdName)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		// whatis format: "command (section) - description"
		// Extract just the description part
		description := strings.TrimSpace(string(output))

		// Parse whatis output
		if parts := strings.SplitN(description, " - ", 2); len(parts) == 2 {
			return strings.TrimSpace(parts[1]), true
		}

		// If no " - " separator, return as-is
		return description, true
	}

	// No man page available
	return "", false
}

// IsIndexed checks if commands have been indexed
func (idx *Indexer) IsIndexed() bool {
	var value string
	err := idx.db.QueryRow(`
		SELECT value FROM settings WHERE key = 'commands_indexed'
	`).Scan(&value)

	if err != nil {
		return false
	}

	return value == "true"
}

// GetCommandCount returns the number of indexed commands
func (idx *Indexer) GetCommandCount() int {
	var count int
	err := idx.db.QueryRow("SELECT COUNT(*) FROM commands").Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

// SearchCommands searches for commands by name or description
func (idx *Indexer) SearchCommands(query string, limit int) ([]CommandInfo, error) {
	query = strings.ToLower(query)

	rows, err := idx.db.Query(`
		SELECT name, description, has_man
		FROM commands
		WHERE name LIKE ? OR description LIKE ?
		ORDER BY
			CASE
				WHEN name = ? THEN 1
				WHEN name LIKE ? THEN 2
				ELSE 3
			END,
			name
		LIMIT ?
	`, "%"+query+"%", "%"+query+"%", query, query+"%", limit)

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer rows.Close()

	results := []CommandInfo{}
	for rows.Next() {
		var cmd CommandInfo
		var description sql.NullString

		if err := rows.Scan(&cmd.Name, &description, &cmd.HasMan); err != nil {
			continue
		}

		if description.Valid {
			cmd.Description = description.String
		}

		results = append(results, cmd)
	}

	return results, nil
}

// CommandInfo represents a command entry
type CommandInfo struct {
	Name        string
	Description string
	HasMan      bool
	HasHelp     *bool // NULL = unknown, false = no, true = yes
}

// isAlphaNumeric checks if string contains only alphanumeric characters
func isAlphaNumeric(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}
