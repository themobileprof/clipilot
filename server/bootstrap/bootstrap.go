package bootstrap

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/themobileprof/clipilot/internal/models"
	yaml "gopkg.in/yaml.v3"
)

// SeedBuiltinModules scans the modules directory and registers them in the database
func SeedBuiltinModules(db *sql.DB, modulesDir string) error {
	log.Println("🌱 Seeding builtin modules from", modulesDir)

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return fmt.Errorf("failed to read modules directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		path := filepath.Join(modulesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Warning: failed to read %s: %v", path, err)
			continue
		}

		var module models.Module
		if err := yaml.Unmarshal(data, &module); err != nil {
			log.Printf("Warning: failed to parse %s: %v", path, err)
			continue
		}

		// Insert or update (forcing file path to the builtin location)
		_, err = db.Exec(`
			INSERT INTO modules (
				name, version, description, author, 
				file_path, original_filename, uploaded_by, uploaded_at
			) VALUES (?, ?, ?, ?, ?, ?, 'system', CURRENT_TIMESTAMP)
			ON CONFLICT(name, version) DO UPDATE SET
				file_path = excluded.file_path,
				uploaded_by = 'system',
				description = excluded.description
		`, module.Name, module.Version, module.Description, module.Metadata.Author, path, entry.Name())

		if err != nil {
			log.Printf("Warning: failed to seed %s: %v", module.Name, err)
		} else {
			count++
		}
	}

	log.Printf("✓ Seeded %d builtin modules", count)
	return nil
}


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

	// 1. Start with essential commands (Robust Seeding)
	// This ensures we always have high-quality data even in minimal environments
	essential := getEssentialCommands()
	for name, desc := range essential {
		commands[name] = desc
	}

	// 2. Discover local commands from PATH
	cmdList, err := getCommandsFromPATH()
	if err == nil {
		// Get descriptions using whatis (if available)
		for _, cmdName := range cmdList {
			// Don't overwrite essential commands with potentially poorer descriptions
			if _, exists := commands[cmdName]; exists {
				continue
			}
			
			description := getCommandDescription(cmdName)
			commands[cmdName] = description
		}
	} else {
		log.Printf("Warning: PATH discovery failed, relying on essential commands: %v", err)
	}

	return commands, nil
}

// getEssentialCommands returns a hardcoded list of high-value commands
// Used for robust seeding when local discovery is limited
func getEssentialCommands() map[string]string {
	return map[string]string{
		"ls":      "list directory contents",
		"cp":      "copy files and directories",
		"mv":      "move (rename) files",
		"rm":      "remove files or directories",
		"mkdir":   "make directories",
		"rmdir":   "remove empty directories",
		"cd":      "change shell working directory",
		"pwd":     "print name of current/working directory",
		"cat":     "concatenate files and print on the standard output",
		"less":    "opposite of more - file viewer",
		"grep":    "print lines that match patterns",
		"find":    "search for files in a directory hierarchy",
		"tar":     "an archiving utility",
		"zip":     "package and compress (archive) files",
		"unzip":   "list, test and extract compressed files in a ZIP archive",
		"ssh":     "OpenSSH remote login client",
		"scp":     "OpenSSH secure file copy",
		"rsync":   "a fast, versatile, remote (and local) file-copying tool",
		"curl":    "transfer a URL",
		"wget":    "The non-interactive network downloader",
		"git":     "the stupid content tracker",
		"docker":  "Docker image and container command line interface",
		"ps":      "report a snapshot of the current processes",
		"top":     "display Linux processes",
		"htop":    "interactive process viewer",
		"kill":    "send a signal to a process",
		"killall": "kill processes by name",
		"chmod":   "change file mode bits",
		"chown":   "change file owner and group",
		"sudo":    "execute a command as another user",
		"df":      "report file system disk space usage",
		"du":      "estimate file space usage",
		"free":    "Display amount of free and used memory in the system",
		"ip":      "show / manipulate routing, network devices, interfaces and tunnels",
		"ping":    "send ICMP ECHO_REQUEST to network hosts",
		"netstat": "Print network connections, routing tables, interface statistics",
		"ss":      "another utility to investigate sockets",
		"nmap":    "Network exploration tool and security / port scanner",
		"man":     "an interface to the system reference manuals",
		"whatis":  "display one-line manual page descriptions",
		"history": "GNU History Library",
		"echo":    "display a line of text",
		"touch":   "change file timestamps",
		"head":    "output the first part of files",
		"tail":    "output the last part of files",
		"sed":     "stream editor for filtering and transforming text",
		"awk":     "pattern scanning and processing language",
	}
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
