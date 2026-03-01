package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/themobileprof/clipilot/server/handlers"
	"github.com/themobileprof/clipilot/server/middleware"
)

var (
	version = "1.0.0"
)

func main() {
	// Load .env file if it exists
	loadEnvFile(".env")

	// Configuration with environment variable support
	port := getEnv("PORT", "8080")
	dataDir := getEnv("DATA_DIR", "./data")
	staticDir := getEnv("STATIC_DIR", "./server/static")
	tmplDir := getEnv("TEMPLATE_DIR", "./server/templates")
	adminUser := getEnv("ADMIN_USER", "admin")
	adminPass := getEnv("ADMIN_PASSWORD", "")
	githubClientID := getEnv("GITHUB_CLIENT_ID", "")
	githubClientSecret := getEnv("GITHUB_CLIENT_SECRET", "")
	baseURL := getEnv("BASE_URL", "")

	// Allow command-line flags to override environment variables
	flag.StringVar(&port, "port", port, "Server port")
	flag.StringVar(&dataDir, "data", dataDir, "Data directory")
	flag.StringVar(&staticDir, "static", staticDir, "Static files directory")
	flag.StringVar(&tmplDir, "templates", tmplDir, "Templates directory")
	flag.StringVar(&adminUser, "admin", adminUser, "Admin username")
	flag.StringVar(&adminPass, "password", adminPass, "Admin password (required)")
	flag.Parse()

	if adminPass == "" {
		log.Fatal("Error: Admin password is required. Set ADMIN_PASSWORD env var or use --password flag")
	}

	// Create data directories
	uploadsDir := filepath.Join(dataDir, "uploads")
	dbPath := filepath.Join(dataDir, "registry.db")

	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	fmt.Printf("CLIPilot Registry v%s\n", version)
	fmt.Printf("Starting server on port %s\n", port)
	fmt.Printf("Data directory: %s\n", dataDir)
	fmt.Printf("Admin user: %s\n", adminUser)
	fmt.Println()

	// Initialize handlers
	h := handlers.New(handlers.Config{
		UploadsDir:         uploadsDir,
		DBPath:             dbPath,
		StaticDir:          staticDir,
		TemplateDir:        tmplDir,
		AdminUser:          adminUser,
		AdminPass:          adminPass,
		GitHubClientID:     githubClientID,
		GitHubClientSecret: githubClientSecret,
		BaseURL:            baseURL,
	})

	// Setup routes
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", h.Home)
	mux.HandleFunc("/health", h.APIv1Health) // Enhanced health check
	mux.HandleFunc("/modules", h.ListModules)
	mux.HandleFunc("/modules/", h.GetModule)

	// Legacy API endpoints
	mux.HandleFunc("/api/modules", h.APIListModules)
	mux.HandleFunc("/api/modules/", h.APIGetModule)

	// New v1 API endpoints for Clio
	mux.HandleFunc("/api/v1/modules", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/modules" {
			h.APIv1ListModules(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/api/v1/modules/changed", h.APIv1ChangedModules)
	mux.HandleFunc("/api/v1/modules/", func(w http.ResponseWriter, r *http.Request) {
		// Route to appropriate handler based on path suffix
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/modules/")
		parts := strings.Split(path, "/")

		if path == "changed" {
			h.APIv1ChangedModules(w, r)
		} else if len(parts) >= 2 && parts[1] == "download" {
			h.APIv1DownloadModule(w, r)
		} else if len(parts) >= 2 && parts[1] == "dependencies" {
			h.APIv1ModuleDependencies(w, r)
		} else if len(parts) == 1 && parts[0] != "" {
			h.APIv1GetModule(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	// Auth routes
	mux.HandleFunc("/login", h.Login)
	mux.HandleFunc("/logout", h.Logout)
	mux.HandleFunc("/auth/github", h.GitHubLogin)
	mux.HandleFunc("/auth/github/callback", h.GitHubCallback)

	// Protected routes (require authentication)
	mux.HandleFunc("/upload", h.RequireAuth(h.UploadPage))
	mux.HandleFunc("/api/upload", h.RequireAuth(h.APIUpload))
	mux.HandleFunc("/my-modules", h.RequireAuth(h.MyModules))

	geminiAPIKey := getEnv("GEMINI_API_KEY", "")

	// Semantic search endpoint (public) - now cached
	mux.HandleFunc("/api/commands/search", h.HandleSemanticSearch(geminiAPIKey))

	// Module request tracking (public POST, admin-only view)
	mux.HandleFunc("/api/module-request", h.APIModuleRequest)
	mux.HandleFunc("/api/module-request/", h.APIUpdateModuleRequest)
	mux.HandleFunc("/module-requests", h.ModuleRequestsPage)

	// Install script endpoints (for Clio client installation)
	mux.HandleFunc("/clio", h.GetInstallScript)                         // Public - serves latest install script
	mux.HandleFunc("/api/install-script/upload", h.UploadInstallScript) // Admin only - upload new script
	mux.HandleFunc("/api/install-scripts", h.ListInstallScripts)        // Admin only - list all scripts
	mux.HandleFunc("/api/install-scripts/", h.ActivateInstallScript)    // Admin only - activate specific version

	// Admin API key management
	mux.HandleFunc("/admin/api-keys", h.AdminAPIKeysPageWithFlash) // Admin only - manage API keys
	mux.HandleFunc("/admin/api-keys/generate", h.GenerateAPIKey)   // Admin only - generate new key
	mux.HandleFunc("/admin/api-keys/revoke", h.RevokeAPIKey)       // Admin only - revoke key

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Initialize Rate Limiter: 60 requests per minute
	rateLimiter := middleware.NewRateLimiter(60, 1*time.Minute)

	// Start server
	addr := ":" + port
	if baseURL == "" {
		baseURL = "http://localhost" + addr
	}
	fmt.Printf("✓ Server ready at %s\n", baseURL)
	fmt.Println("  - Home: /")
	fmt.Println("  - Health: /health")
	fmt.Println("  - Modules: /modules")
	fmt.Println("  - Upload: /upload (requires login)")
	fmt.Println("  - API (legacy): /api/modules")
	fmt.Println("  - API v1: /api/v1/modules")
	fmt.Println("  - API v1 Delta Sync: /api/v1/modules/changed")
	fmt.Println("  - Clio Install: /clio (public)")
	fmt.Println("  - Clio Upload: /api/install-script/upload (admin)")
	fmt.Println("  - API Keys: /admin/api-keys (admin)")
	fmt.Println()

	// Wrap mux with rate limiter
	if err := http.ListenAndServe(addr, rateLimiter.Limit(mux)); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		// .env file is optional, silently continue if not found
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"'")

		// Set environment variable if not already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
