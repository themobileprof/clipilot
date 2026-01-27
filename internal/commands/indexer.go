package commands

import (
	"database/sql"
	"fmt"
	"os"
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
	defer func() {
		_ = tx.Rollback() // Ignore error - might be committed
	}()

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
			fmt.Printf("Error inserting %s: %v\n", cmdName, err)
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

// discoverCommands returns all executable commands by scanning PATH
func (idx *Indexer) discoverCommands() ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		// Fallback paths if PATH is empty
		pathEnv = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	}

	paths := strings.Split(pathEnv, ":")
	uniqueCmds := make(map[string]bool)

	for _, dir := range paths {
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			// Skip unreadable directories
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Check if executable (any execute bit set)
			if info.Mode()&0111 == 0 {
				continue
			}

			name := entry.Name()
			// Filter out odd filenames or temporary files
			if len(name) == 0 || name[0] == '.' {
				continue
			}

			uniqueCmds[name] = true
		}
	}

	if len(uniqueCmds) == 0 {
		return nil, fmt.Errorf("no commands found in PATH")
	}

	commands := make([]string, 0, len(uniqueCmds))
	for cmd := range uniqueCmds {
		commands = append(commands, cmd)
	}

	return commands, nil
}

// getCommandDescription fetches the short description of a command
// Uses whatis and apropos to find the best description
func (idx *Indexer) getCommandDescription(name string) (string, bool) {
	// 1. Try whatis (standard and fast)
	cmd := exec.Command("whatis", name)
	output, err := cmd.Output()
	if err == nil {
		desc := idx.parseManOutput(string(output), name)
		if desc != "" {
			return desc, true
		}
	}

	// 2. Try apropos (broader search)
	cmd = exec.Command("apropos", name)
	output, err = cmd.Output()
	if err == nil {
		desc := idx.parseManOutput(string(output), name)
		if desc != "" {
			return desc, true
		}
	}

	return "", false
}

// parseManOutput extracts description from "name (sec) - description" format
func (idx *Indexer) parseManOutput(output, name string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) != 2 {
			continue
		}

		left := parts[0]
		desc := parts[1]

		// Check if 'name' is in the comma-separated list of commands
		// e.g. "bzcmp, bzdiff (1)"
		cmds := strings.Split(left, ",")
		found := false
		for _, c := range cmds {
			c = strings.TrimSpace(c)
			// Remove section info like " (1)" or "(1)"
			if idx := strings.Index(c, "("); idx != -1 {
				c = c[:idx]
			}
			c = strings.TrimSpace(c)

			if c == name {
				found = true
				break
			}
		}

		if found {
			return strings.TrimSpace(desc)
		}
	}
	return ""
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

// SearchCommands searches for commands by name or description using FTS
func (idx *Indexer) SearchCommands(query string, limit int) ([]CommandInfo, error) {
	// If the FTS table doesn't exist (migrations not run?), fall back to LIKE
	// But we assume migrations are run.
	
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return []CommandInfo{}, nil
	}

	// We join with the main table to get metadata like has_man
	// FTS5 rank is lower is better (usually), but depends on configuration. 
	
	rows, err := idx.db.Query(`
		SELECT c.name, c.description, c.has_man
		FROM commands c
		JOIN commands_fts f ON c.name = f.name
		WHERE commands_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit)

    // Helper to handle error if FTS table is missing (fallback)
	if err != nil && strings.Contains(err.Error(), "no such table") {
        return idx.searchCommandsFallback(query, limit)
    }
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

// searchCommandsFallback uses simple LIKE search
func (idx *Indexer) searchCommandsFallback(query string, limit int) ([]CommandInfo, error) {
    query = strings.ToLower(query)
	rows, err := idx.db.Query(`
		SELECT name, description, has_man
		FROM commands
		WHERE name LIKE ? OR description LIKE ?
		ORDER BY length(name)
		LIMIT ?
	`, "%"+query+"%", "%"+query+"%", limit)
    if err != nil {
        return nil, err
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

// buildFTSQuery constructs a search query for FTS5
// It tokenizes input and creates a query like: "token1"* OR "token2"*
func buildFTSQuery(input string) string {
    // Remove special FTS characters to prevent syntax errors
    cleaner := strings.NewReplacer("\"", "", "'", "", ":", "", "*", "", "(", "", ")", "")
    input = cleaner.Replace(input)
    input = strings.ToLower(input)
    
    terms := strings.Fields(input)
    if len(terms) == 0 {
        return ""
    }
    
    // Common stopwords to filter out for better keyword matching
    stopWords := map[string]bool{
        "how": true, "to": true, "do": true, "i": true, "a": true, "an": true, 
        "the": true, "for": true, "with": true, "my": true, "in": true, 
        "what": true, "is": true, "where": true, "me": true, "see": true,
        "show": true, "tell": true, "want": true, "need": true, "check": true,
        "files": true, "file": true, "output": true,
        "delete": false, "remove": false, "folder": false, "directory": false,
    }

    var queryParts []string
    for _, term := range terms {
        if stopWords[term] {
            continue
        }
        // Use prefix matching for each term
        queryParts = append(queryParts, fmt.Sprintf("\"%s\"*", term))
    }
    
    // If all terms were stopwords, revert to using original terms
    if len(queryParts) == 0 {
        for _, term := range terms {
            queryParts = append(queryParts, fmt.Sprintf("\"%s\"*", term))
        }
    }
    
    // OR semantics allow finding "list files" -> "list" works, even if "files" doesn't match
    return strings.Join(queryParts, " OR ")
}

// CommandInfo represents a command entry
type CommandInfo struct {
	Name        string
	Description string
	HasMan      bool
	HasHelp     *bool // NULL = unknown, false = no, true = yes
}
