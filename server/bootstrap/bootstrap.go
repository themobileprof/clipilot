package bootstrap

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DiscoverAndSubmitCommands discovers commands on the server and submits them for enhancement
func DiscoverAndSubmitCommands(db *sql.DB, minCommands int) error {
	// Check current count of enhanced commands
	var enhancedCount int
	err := db.QueryRow("SELECT COUNT(*) FROM enhanced_commands").Scan(&enhancedCount)
	if err != nil {
		return fmt.Errorf("failed to count enhanced commands: %w", err)
	}

	log.Printf("Current enhanced commands: %d", enhancedCount)

	// If we have enough commands, skip bootstrap
	if enhancedCount >= minCommands {
		log.Printf("✓ Sufficient commands (%d >= %d), skipping bootstrap", enhancedCount, minCommands)
		return nil
	}

	log.Printf("⚠️  Low command count (%d < %d), bootstrapping server commands...", enhancedCount, minCommands)

	// Discover commands on this server
	commands, err := discoverServerCommands()
	if err != nil {
		return fmt.Errorf("failed to discover commands: %w", err)
	}

	if len(commands) == 0 {
		return fmt.Errorf("no commands found on server")
	}

	log.Printf("🔍 Discovered %d commands on server", len(commands))

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Prepare submission statement
	stmt, err := tx.Prepare(`
		INSERT INTO command_submissions (command_name, user_description, submitted_by, submitted_at, processed)
		VALUES (?, ?, 'bootstrap', ?, 0)
		ON CONFLICT(command_name, submitted_by) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	submitted := 0
	timestamp := time.Now().Unix()

	for cmdName, description := range commands {
		// Check if already enhanced (don't submit if already enhanced)
		var exists bool
		err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM enhanced_commands WHERE name = ?)", cmdName).Scan(&exists)
		if err == nil && exists {
			continue
		}

		// Submit for enhancement
		_, err = stmt.Exec(cmdName, description, timestamp)
		if err == nil {
			submitted++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	log.Printf("✓ Submitted %d server commands for enhancement", submitted)
	log.Printf("💡 Run enhancement tool to process: ./bin/enhance --auto --limit=100")

	return nil
}

// discoverServerCommands discovers commands available on the server
func discoverServerCommands() (map[string]string, error) {
	commands := make(map[string]string)

	// Get all commands from PATH
	cmdList, err := getCommandsFromPATH()
	if err != nil {
		return nil, err
	}

	// Get descriptions using whatis (if available)
	for _, cmdName := range cmdList {
		description := getCommandDescription(cmdName)
		commands[cmdName] = description
	}

	return commands, nil
}

// getCommandsFromPATH gets all executable commands from PATH
func getCommandsFromPATH() ([]string, error) {
	// Use compgen -c to list all commands (bash built-in)
	cmd := exec.Command("bash", "-c", "compgen -c | sort -u | head -500")
	output, err := cmd.Output()
	if err != nil {
		// Fallback: manually scan PATH directories (limited)
		return scanPATHDirectories(), nil
	}

	lines := strings.Split(string(output), "\n")
	commands := []string{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "/") {
			continue
		}
		commands = append(commands, line)
	}

	return commands, nil
}

// scanPATHDirectories manually scans PATH directories for executables
func scanPATHDirectories() []string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		pathEnv = "/usr/local/bin:/usr/bin:/bin"
	}

	dirs := strings.Split(pathEnv, ":")
	commandsMap := make(map[string]bool)
	count := 0
	maxCommands := 300

	for _, dir := range dirs {
		if count >= maxCommands {
			break
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if count >= maxCommands {
				break
			}

			if entry.IsDir() {
				continue
			}

			// Check if executable
			info, err := entry.Info()
			if err != nil {
				continue
			}

			mode := info.Mode()
			if mode&0111 != 0 { // Has execute bit
				commandsMap[entry.Name()] = true
				count++
			}
		}
	}

	commands := []string{}
	for cmd := range commandsMap {
		commands = append(commands, cmd)
	}

	return commands
}

// getCommandDescription gets description from whatis
func getCommandDescription(cmdName string) string {
	// Try whatis first (with timeout)
	cmd := exec.Command("whatis", cmdName)
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			// Parse "command (section) - description" format
			line := strings.TrimSpace(lines[0])
			if idx := strings.Index(line, " - "); idx > 0 {
				return strings.TrimSpace(line[idx+3:])
			}
		}
	}

	// Simple fallback - just mark as available
	if _, err := exec.LookPath(cmdName); err == nil {
		return "Command line utility"
	}

	return "No description available"
}

// GetBootstrapStatus returns information about bootstrap status
func GetBootstrapStatus(db *sql.DB) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Count enhanced commands
	var enhancedCount int
	err := db.QueryRow("SELECT COUNT(*) FROM enhanced_commands").Scan(&enhancedCount)
	if err != nil {
		return nil, err
	}
	status["enhanced_count"] = enhancedCount

	// Count unprocessed submissions
	var unprocessedCount int
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT cs.command_name)
		FROM command_submissions cs
		LEFT JOIN enhanced_commands ec ON cs.command_name = ec.name
		WHERE cs.processed = 0 AND ec.name IS NULL
	`).Scan(&unprocessedCount)
	if err != nil {
		return nil, err
	}
	status["unprocessed_count"] = unprocessedCount

	// Count total submissions (including bootstrap)
	var totalSubmissions int
	err = db.QueryRow("SELECT COUNT(*) FROM command_submissions WHERE submitted_by = 'bootstrap'").Scan(&totalSubmissions)
	if err == nil {
		status["bootstrap_submissions"] = totalSubmissions
	}

	// Check if bootstrap has run
	status["bootstrap_ran"] = totalSubmissions > 0

	return status, nil
}
