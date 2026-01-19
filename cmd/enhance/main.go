package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

// Admin tool to enhance command descriptions using Gemini
// Usage: go run cmd/enhance/main.go --db=registry.db --api-key=... --command=ss

var (
	dbPath         = flag.String("db", "data/registry.db", "Path to registry database")
	apiKey         = flag.String("api-key", "", "Gemini API key")
	command        = flag.String("command", "", "Command name to enhance")
	batchFile      = flag.String("batch", "", "JSON file with commands to enhance")
	dryRun         = flag.Bool("dry-run", false, "Show what would be done without saving")
	forceReenhance = flag.Bool("force", false, "Re-enhance even if already enhanced")
)

type CommandToEnhance struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func main() {
	flag.Parse()

	if *apiKey == "" {
		fmt.Println("Error: --api-key required")
		fmt.Println("\nGet your Gemini API key from: https://makersuite.google.com/app/apikey")
		os.Exit(1)
	}

	// Open database
	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if *command != "" {
		// Enhance single command
		enhanceSingleCommand(db, *command)
	} else if *batchFile != "" {
		// Enhance batch from file
		enhanceBatchCommands(db, *batchFile)
	} else if flag.NArg() > 0 {
		// Enhance unprocessed submissions
		enhanceUnprocessedSubmissions(db, flag.Arg(0))
	} else {
		fmt.Println("Usage:")
		fmt.Println("  --command=ss                     # Enhance single command")
		fmt.Println("  --batch=commands.json            # Enhance batch from file")
		fmt.Println("  --api-key=YOUR_KEY               # Required: Gemini API key")
		fmt.Println("  --dry-run                        # Test without saving")
		fmt.Println("\nOr: enhance_commands <count>     # Process N unprocessed submissions")
	}
}

func enhanceSingleCommand(db *sql.DB, cmdName string) {
	// Check if already enhanced
	if !*forceReenhance {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM enhanced_commands WHERE name = ?)", cmdName).Scan(&exists)
		if err == nil && exists {
			fmt.Printf("⏭️  Command '%s' already enhanced (use --force to re-enhance)\n", cmdName)
			return
		}
	}

	// Get command from submissions
	var description string
	err := db.QueryRow(`
		SELECT user_description 
		FROM command_submissions 
		WHERE command_name = ? AND processed = 0
		LIMIT 1
	`, cmdName).Scan(&description)

	if err != nil {
		// Try enhanced_commands table
		err = db.QueryRow("SELECT description FROM enhanced_commands WHERE name = ?", cmdName).Scan(&description)
		if err != nil {
			log.Fatalf("Command not found: %s", cmdName)
		}
	}

	fmt.Printf("=== Enhancing: %s ===\n", cmdName)
	fmt.Printf("Original: %s\n\n", description)

	// Call enhancement (placeholder - implement actual Gemini call)
	enhanced := enhanceCommandDescription(*apiKey, cmdName, description)

	fmt.Printf("Enhanced Description: %s\n", enhanced.EnhancedDescription)
	fmt.Printf("Keywords: %v\n", enhanced.Keywords)
	fmt.Printf("Category: %s\n", enhanced.Category)
	fmt.Printf("Use Cases: %v\n", enhanced.UseCases)

	if !*dryRun {
		// Save to database
		saveEnhancement(db, cmdName, description, enhanced)
		fmt.Println("\n✓ Saved to database")
	} else {
		fmt.Println("\n[DRY RUN - not saved]")
	}
}

func enhanceBatchCommands(db *sql.DB, filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Failed to read batch file: %v", err)
	}

	var commands []CommandToEnhance
	if err := json.Unmarshal(data, &commands); err != nil {
		log.Fatalf("Invalid JSON: %v", err)
	}

	// Filter out already enhanced commands
	toProcess := []CommandToEnhance{}
	for _, cmd := range commands {
		if *forceReenhance {
			toProcess = append(toProcess, cmd)
		} else {
			var exists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM enhanced_commands WHERE name = ?)", cmd.Name).Scan(&exists)
			if err == nil && !exists {
				toProcess = append(toProcess, cmd)
			}
		}
	}

	skipped := len(commands) - len(toProcess)
	if skipped > 0 {
		fmt.Printf("⏭️  Skipping %d already enhanced commands (use --force to re-enhance)\n", skipped)
	}

	if len(toProcess) == 0 {
		fmt.Println("✓ All commands already enhanced!")
		return
	}

	fmt.Printf("Processing %d commands...\n\n", len(toProcess))

	for i, cmd := range toProcess {
		fmt.Printf("[%d/%d] Enhancing: %s\n", i+1, len(toProcess), cmd.Name)

		enhanced := enhanceCommandDescription(*apiKey, cmd.Name, cmd.Description)

		if !*dryRun {
			saveEnhancement(db, cmd.Name, cmd.Description, enhanced)
			fmt.Println("  ✓ Saved")
		}
		fmt.Println()
	}

	fmt.Printf("✓ Completed %d commands (%d skipped)\n", len(toProcess), skipped)
}

func enhanceUnprocessedSubmissions(db *sql.DB, countStr string) {
	var count int
	fmt.Sscanf(countStr, "%d", &count)
	if count <= 0 {
		count = 10
	}

	// Only get unprocessed submissions that aren't already enhanced
	rows, err := db.Query(`
		SELECT DISTINCT cs.command_name, cs.user_description
		FROM command_submissions cs
		LEFT JOIN enhanced_commands ec ON cs.command_name = ec.name
		WHERE cs.processed = 0 AND ec.name IS NULL
		LIMIT ?
	`, count)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	commands := []CommandToEnhance{}
	for rows.Next() {
		var cmd CommandToEnhance
		if err := rows.Scan(&cmd.Name, &cmd.Description); err == nil {
			commands = append(commands, cmd)
		}
	}

	fmt.Printf("Processing %d unprocessed submissions...\n\n", len(commands))

	for i, cmd := range commands {
		fmt.Printf("[%d/%d] Enhancing: %s\n", i+1, len(commands), cmd.Name)

		enhanced := enhanceCommandDescription(*apiKey, cmd.Name, cmd.Description)

		if !*dryRun {
			saveEnhancement(db, cmd.Name, cmd.Description, enhanced)
			markProcessed(db, cmd.Name)
			fmt.Println("  ✓ Saved and marked processed")
		}
		fmt.Println()
	}
}

type Enhancement struct {
	EnhancedDescription string
	Keywords            []string
	Category            string
	UseCases            []string
}

func enhanceCommandDescription(apiKey, name, description string) *Enhancement {
	// TODO: Implement actual Gemini API call
	// For now, return placeholder

	fmt.Println("  [Calling Gemini API...]")

	return &Enhancement{
		EnhancedDescription: description + " (placeholder)",
		Keywords:            []string{"keyword1", "keyword2"},
		Category:            "general",
		UseCases:            []string{"use case 1"},
	}
}

func saveEnhancement(db *sql.DB, name, originalDesc string, enh *Enhancement) {
	keywordsStr := ""
	if len(enh.Keywords) > 0 {
		keywordsStr = enh.Keywords[0]
		for i := 1; i < len(enh.Keywords); i++ {
			keywordsStr += "," + enh.Keywords[i]
		}
	}

	// Track if this is a re-enhancement (version bump)
	var currentVersion int
	db.QueryRow("SELECT version FROM enhanced_commands WHERE name = ?", name).Scan(&currentVersion)
	newVersion := currentVersion + 1

	useCasesStr := ""
	if len(enh.UseCases) > 0 {
		useCasesStr = enh.UseCases[0]
		for i := 1; i < len(enh.UseCases); i++ {
			useCasesStr += "," + enh.UseCases[i]
		}
	}

	_, err := db.Exec(`
		INSERT INTO enhanced_commands 
		(name, description, enhanced_description, keywords, category, use_cases, source, last_enhanced, enhancement_model, version)
		VALUES (?, ?, ?, ?, ?, ?, 'ai', strftime('%s', 'now'), 'gemini-1.5-flash', ?)
		ON CONFLICT(name) DO UPDATE SET
			enhanced_description = excluded.enhanced_description,
			keywords = excluded.keywords,
			category = excluded.category,
			use_cases = excluded.use_cases,
			last_enhanced = excluded.last_enhanced,
			version = excluded.version,
			updated_at = strftime('%s', 'now')
	`, name, originalDesc, enh.EnhancedDescription, keywordsStr, enh.Category, useCasesStr, newVersion)

	if err != nil {
		log.Printf("  Error saving: %v", err)
	}
}

func markProcessed(db *sql.DB, cmdName string) {
	_, _ = db.Exec("UPDATE command_submissions SET processed = 1 WHERE command_name = ?", cmdName)
}
