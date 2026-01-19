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
	mux.HandleFunc("/health", h.HealthCheck) // Health check
	mux.HandleFunc("/modules", h.ListModules)
	mux.HandleFunc("/modules/", h.GetModule)
	mux.HandleFunc("/api/modules", h.APIListModules)
	mux.HandleFunc("/api/modules/", h.APIGetModule)

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
	fmt.Println("  - API: /api/modules")
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
