package handlers

import (
	"database/sql"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"

	"github.com/themobileprof/clipilot/pkg/models"
	"github.com/themobileprof/clipilot/server/auth"
)

//go:embed migration.sql
var migrationSQL string

type Config struct {
	UploadsDir  string
	DBPath      string
	StaticDir   string
	TemplateDir string
	AdminUser   string
	AdminPass   string
}

type Handlers struct {
	config    Config
	db        *sql.DB
	templates *template.Template
	auth      *auth.Manager
}

type ModuleRecord struct {
	ID          int64
	Name        string
	Version     string
	Description string
	Author      string
	UploadedAt  time.Time
	UploadedBy  string
	FilePath    string
	Downloads   int
}

func New(cfg Config) *Handlers {
	// Initialize database
	db, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Run migrations
	if _, err := db.Exec(migrationSQL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Load templates
	tmplPattern := filepath.Join(cfg.TemplateDir, "*.html")
	templates, err := template.ParseGlob(tmplPattern)
	if err != nil {
		log.Printf("Warning: Failed to load templates from %s: %v", tmplPattern, err)
		templates = template.New("empty")
	}

	// Initialize auth manager
	authMgr := auth.NewManager(cfg.AdminUser, cfg.AdminPass)

	return &Handlers{
		config:    cfg,
		db:        db,
		templates: templates,
		auth:      authMgr,
	}
}

// Home page
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title":       "CLIPilot Registry",
		"Description": "Community module registry for CLIPilot",
		"LoggedIn":    h.auth.IsAuthenticated(r),
	}

	if err := h.templates.ExecuteTemplate(w, "home.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ListModules displays all modules
func (h *Handlers) ListModules(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, version, description, author, uploaded_at, uploaded_by, downloads
		FROM modules
		ORDER BY uploaded_at DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var modules []ModuleRecord
	for rows.Next() {
		var m ModuleRecord
		if err := rows.Scan(&m.ID, &m.Name, &m.Version, &m.Description, &m.Author, &m.UploadedAt, &m.UploadedBy, &m.Downloads); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		modules = append(modules, m)
	}

	data := map[string]interface{}{
		"Title":    "Browse Modules",
		"Modules":  modules,
		"LoggedIn": h.auth.IsAuthenticated(r),
	}

	if err := h.templates.ExecuteTemplate(w, "modules.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetModule serves a specific module for download
func (h *Handlers) GetModule(w http.ResponseWriter, r *http.Request) {
	// Extract module ID from URL (e.g., /modules/123)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	moduleID := parts[1]
	var m ModuleRecord
	err := h.db.QueryRow(`
		SELECT id, name, version, file_path, downloads
		FROM modules
		WHERE id = ?
	`, moduleID).Scan(&m.ID, &m.Name, &m.Version, &m.FilePath, &m.Downloads)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Increment download counter
	_, _ = h.db.Exec("UPDATE modules SET downloads = downloads + 1 WHERE id = ?", m.ID)

	// Serve file
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.yaml", m.Name, m.Version))
	http.ServeFile(w, r, m.FilePath)
}

// UploadPage shows the upload form (authenticated users only)
func (h *Handlers) UploadPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Upload Module",
		"LoggedIn": true,
		"Username": h.auth.GetUsername(r),
	}

	if err := h.templates.ExecuteTemplate(w, "upload.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// APIUpload handles module file uploads
func (h *Handlers) APIUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("module")
	if err != nil {
		http.Error(w, "Missing module file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate YAML
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	var module models.Module
	if err := yaml.Unmarshal(data, &module); err != nil {
		http.Error(w, fmt.Sprintf("Invalid YAML: %v", err), http.StatusBadRequest)
		return
	}

	// Basic validation
	if module.Name == "" || module.Version == "" {
		http.Error(w, "Module must have name and version", http.StatusBadRequest)
		return
	}

	// Check for duplicates
	var exists int
	err = h.db.QueryRow("SELECT COUNT(*) FROM modules WHERE name = ? AND version = ?",
		module.Name, module.Version).Scan(&exists)
	if err == nil && exists > 0 {
		http.Error(w, "Module with this name and version already exists", http.StatusConflict)
		return
	}

	// Save file
	filename := fmt.Sprintf("%s-%s-%d.yaml", module.Name, module.Version, time.Now().Unix())
	filepath := filepath.Join(h.config.UploadsDir, filename)

	outFile, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	if _, err := outFile.Write(data); err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	// Insert into database
	username := h.auth.GetUsername(r)
	_, err = h.db.Exec(`
		INSERT INTO modules (name, version, description, author, uploaded_by, file_path, original_filename)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, module.Name, module.Version, module.Description,
		module.Metadata.Author, username, filepath, header.Filename)

	if err != nil {
		log.Printf("Database insert error: %v", err)
		http.Error(w, "Failed to save module", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"success": true, "message": "Module uploaded successfully"}`)
}

// MyModules shows modules uploaded by the current user
func (h *Handlers) MyModules(w http.ResponseWriter, r *http.Request) {
	username := h.auth.GetUsername(r)

	rows, err := h.db.Query(`
		SELECT id, name, version, description, uploaded_at, downloads
		FROM modules
		WHERE uploaded_by = ?
		ORDER BY uploaded_at DESC
	`, username)

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var modules []ModuleRecord
	for rows.Next() {
		var m ModuleRecord
		if err := rows.Scan(&m.ID, &m.Name, &m.Version, &m.Description, &m.UploadedAt, &m.Downloads); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		modules = append(modules, m)
	}

	data := map[string]interface{}{
		"Title":    "My Modules",
		"Modules":  modules,
		"LoggedIn": true,
		"Username": username,
	}

	if err := h.templates.ExecuteTemplate(w, "my-modules.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Login handles authentication
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := map[string]interface{}{
			"Title": "Login",
		}
		if err := h.templates.ExecuteTemplate(w, "login.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if h.auth.Authenticate(username, password) {
			h.auth.SetSession(w, username)
			http.Redirect(w, r, "/upload", http.StatusSeeOther)
			return
		}

		data := map[string]interface{}{
			"Title": "Login",
			"Error": "Invalid credentials",
		}
		if err := h.templates.ExecuteTemplate(w, "login.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Logout clears session
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	h.auth.ClearSession(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// RequireAuth middleware
func (h *Handlers) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.auth.IsAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// API endpoints for CLI access
func (h *Handlers) APIListModules(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT id, name, version, description, author, downloads
		FROM modules
		ORDER BY uploaded_at DESC
	`)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("["))

	first := true
	for rows.Next() {
		var m ModuleRecord
		if err := rows.Scan(&m.ID, &m.Name, &m.Version, &m.Description, &m.Author, &m.Downloads); err != nil {
			continue
		}

		if !first {
			w.Write([]byte(","))
		}
		first = false

		fmt.Fprintf(w, `{"id":%d,"name":"%s","version":"%s","description":"%s","author":"%s","downloads":%d}`,
			m.ID, m.Name, m.Version, m.Description, m.Author, m.Downloads)
	}

	w.Write([]byte("]"))
}

func (h *Handlers) APIGetModule(w http.ResponseWriter, r *http.Request) {
	// Same as GetModule but with JSON response option
	h.GetModule(w, r)
}
