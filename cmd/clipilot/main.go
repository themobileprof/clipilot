package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/themobileprof/clipilot/internal/db"
	"github.com/themobileprof/clipilot/internal/modules"
	"github.com/themobileprof/clipilot/internal/ui"
	"github.com/themobileprof/clipilot/internal/config"
)

var (
	version     = "1.0.0"
	configPath  string
	dbPath      string
	dryRun      bool
	initDB      bool
	resetDB     bool
	loadDir     string
	showVersion bool
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	defaultDBPath := filepath.Join(homeDir, ".clipilot", "clipilot.db")
	defaultConfigPath := filepath.Join(homeDir, ".clipilot", "config.yaml")

	flag.StringVar(&configPath, "config", defaultConfigPath, "Path to configuration file")
	flag.StringVar(&dbPath, "db", defaultDBPath, "Path to SQLite database")
	flag.BoolVar(&dryRun, "dry-run", false, "Show commands without executing")
	flag.BoolVar(&initDB, "init", false, "Initialize database and load modules")
	flag.BoolVar(&resetDB, "reset", false, "Reset database (delete and reinitialize)")
	flag.StringVar(&loadDir, "load", "", "Load modules from directory")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("CLIPilot v%s\n", version)
		fmt.Println("Offline CLI Assistant for Linux")
		return
	}

	// Load configuration (creates with defaults if doesn't exist)
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	// Use config for database path if not overridden by flag
	if dbPath == "" || dbPath == filepath.Join(os.Getenv("HOME"), ".clipilot", "clipilot.db") {
		dbPath = cfg.DBPath
	}

	// Reset if requested
	if resetDB {
		if err := resetDatabase(dbPath, loadDir); err != nil {
			log.Fatalf("Reset failed: %v", err)
		}
		return
	}

	// Open database
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Initialize if requested
	if initDB {
		if err := initializeDB(database, loadDir); err != nil {
			log.Fatalf("Initialization failed: %v", err)
		}
		return
	}

	// Load modules from directory if specified
	if loadDir != "" {
		if err := loadModulesFromDir(database, loadDir); err != nil {
			log.Fatalf("Failed to load modules: %v", err)
		}
		return
	}

	// Create REPL
	repl := ui.NewREPL(database.Conn())

	// Auto-sync registry if needed (only in interactive mode)
	args := flag.Args()
	if len(args) == 0 {
		// Interactive mode - check for auto-sync
		if err := repl.AutoSyncIfNeeded(); err != nil {
			log.Printf("Warning: Auto-sync failed: %v", err)
		}
	}

	// Check for non-interactive command
	if len(args) > 0 {
		// Non-interactive mode - join all args as a single command
		command := strings.Join(args, " ")
		if err := repl.ExecuteNonInteractive(command); err != nil {
			log.Fatalf("Command failed: %v", err)
		}
		return
	}

	// Start interactive REPL
	if err := repl.Start(); err != nil {
		log.Fatalf("REPL error: %v", err)
	}
}

// initializeDB initializes the database and optionally loads modules
func initializeDB(database *db.DB, modulesDir string) error {
	fmt.Println("Initializing CLIPilot...")

	// Database is already initialized in db.New()
	fmt.Println("✓ Database initialized")

	// Load modules if directory specified
	if modulesDir != "" {
		if err := loadModulesFromDir(database, modulesDir); err != nil {
			return fmt.Errorf("failed to load modules: %w", err)
		}
	} else {
		// Try to load from default location
		defaultDir := "modules"
		if _, err := os.Stat(defaultDir); err == nil {
			if err := loadModulesFromDir(database, defaultDir); err != nil {
				fmt.Printf("Warning: failed to load default modules: %v\n", err)
			}
		}
	}

	fmt.Println("\n✓ CLIPilot initialized successfully!")
	fmt.Println("Run 'clipilot' to start the interactive assistant.")
	return nil
}

// resetDatabase deletes and reinitializes the database
func resetDatabase(dbPath string, modulesDir string) error {
	fmt.Println("Resetting CLIPilot database...")
	fmt.Printf("Database: %s\n", dbPath)

	// Check if database exists
	if _, err := os.Stat(dbPath); err == nil {
		// Prompt for confirmation
		fmt.Print("\n⚠️  This will delete all your data, modules, and settings. Continue? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" && response != "YES" {
			fmt.Println("Reset cancelled.")
			return nil
		}

		// Delete the database file
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("failed to delete database: %w", err)
		}
		fmt.Println("✓ Database deleted")
	} else {
		fmt.Println("Database doesn't exist, creating new one...")
	}

	// Create new database
	database, err := db.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer database.Close()

	fmt.Println("✓ Database recreated")

	// Determine default modules directory
	homeDir, _ := os.UserHomeDir()
	defaultDir := filepath.Join(homeDir, ".clipilot", "modules")

	// Check if we should clear local modules cache (preventing stale data)
	if modulesDir == "" {
		if _, err := os.Stat(defaultDir); err == nil {
			fmt.Print("⚠️  Clear local modules cache (recommended)? [Y/n]: ")
			var modResp string
			_, _ = fmt.Scanln(&modResp)
			if modResp == "" || strings.ToLower(modResp) == "y" || strings.ToLower(modResp) == "yes" {
				if err := os.RemoveAll(defaultDir); err == nil {
					if err := os.MkdirAll(defaultDir, 0755); err != nil {
						fmt.Printf("Warning: failed to recreate modules dir: %v\n", err)
					}
					fmt.Println("✓ Local modules cache cleared")
				} else {
					fmt.Printf("Warning: failed to clear modules: %v\n", err)
				}
			}
		}
	}

	// Load modules if directory specified or still exists
	if modulesDir != "" {
		if err := loadModulesFromDir(database, modulesDir); err != nil {
			return fmt.Errorf("failed to load modules: %w", err)
		}
	} else {
		// Try to load from default location (if not cleared)
		if matches, _ := filepath.Glob(filepath.Join(defaultDir, "*.yaml")); len(matches) > 0 {
			if err := loadModulesFromDir(database, defaultDir); err != nil {
				fmt.Printf("Warning: failed to load default modules: %v\n", err)
			}
		} else {
			fmt.Println("ℹ️  No local modules found (cache empty).")
		}
	}

	fmt.Println("\n✓ Database reset successfully!")
	fmt.Println("💡 Run 'clipilot sync' to get fresh modules from the registry.")
	return nil
}

// loadModulesFromDir loads all YAML modules from a directory
func loadModulesFromDir(database *db.DB, dir string) error {
	loader := modules.NewLoader(database.Conn())

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .yaml and .yml files
		name := entry.Name()
		if filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml" {
			continue
		}

		path := filepath.Join(dir, name)
		fmt.Printf("Loading %s... ", name)

		module, err := loader.LoadFromFile(path)
		if err != nil {
			fmt.Printf("✗ Error: %v\n", err)
			continue
		}

		if err := loader.ImportModule(module); err != nil {
			fmt.Printf("✗ Import failed: %v\n", err)
			continue
		}

		fmt.Printf("✓ %s (v%s)\n", module.Name, module.Version)
		loadedCount++
	}

	if loadedCount == 0 {
		fmt.Println("No modules loaded.")
	} else {
		fmt.Printf("\n✓ Loaded %d module(s)\n", loadedCount)
	}

	return nil
}
