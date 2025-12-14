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
	"regexp"
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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

// validateModule performs comprehensive validation on a module
func validateModule(module *models.Module) error {
	// Validate required top-level fields
	if module.Name == "" {
		return fmt.Errorf("name is required")
	}
	if module.Version == "" {
		return fmt.Errorf("version is required")
	}
	if module.Description == "" {
		return fmt.Errorf("description is required")
	}

	// Validate name format (lowercase, alphanumeric, underscores only)
	nameRegex := regexp.MustCompile(`^[a-z0-9_]+$`)
	if !nameRegex.MatchString(module.Name) {
		return fmt.Errorf("name must be lowercase alphanumeric with underscores only (got: %s)", module.Name)
	}

	// Validate version format (semantic versioning)
	versionRegex := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !versionRegex.MatchString(module.Version) {
		return fmt.Errorf("version must follow semantic versioning (e.g., 1.0.0, got: %s)", module.Version)
	}

	// Validate tags (at least one required for search/discovery)
	if len(module.Tags) == 0 {
		return fmt.Errorf("tags are required (at least one tag for module discovery)")
	}

	// Validate flows
	if len(module.Flows) == 0 {
		return fmt.Errorf("flows section is required")
	}

	// Check if at least one flow exists (doesn't have to be named "main" in this structure)
	hasValidFlow := false
	for flowName, flow := range module.Flows {
		if flow != nil && len(flow.Steps) > 0 {
			hasValidFlow = true
			// Validate flow has a start step
			if flow.Start == "" {
				return fmt.Errorf("flow '%s': start field is required", flowName)
			}
			// Validate start step exists
			if _, exists := flow.Steps[flow.Start]; !exists {
				return fmt.Errorf("flow '%s': start step '%s' not found in steps", flowName, flow.Start)
			}
		}
	}
	if !hasValidFlow {
		return fmt.Errorf("at least one flow with steps is required")
	}

	// Validate each step in each flow
	validTypes := map[string]bool{
		"instruction": true,
		"action":      true,
		"branch":      true,
		"terminal":    true,
	}

	for flowName, flow := range module.Flows {
		for stepKey, step := range flow.Steps {
			if step.Type == "" {
				return fmt.Errorf("flow '%s', step '%s': type is required", flowName, stepKey)
			}
			if !validTypes[step.Type] {
				return fmt.Errorf("flow '%s', step '%s': invalid type '%s' (must be: instruction, action, branch, or terminal)", flowName, stepKey, step.Type)
			}
			if step.Type == "action" && step.Command == "" {
				return fmt.Errorf("flow '%s', step '%s': command is required for action steps", flowName, stepKey)
			}
			if step.Type == "branch" && step.BasedOn == "" {
				return fmt.Errorf("flow '%s', step '%s': based_on is required for branch steps", flowName, stepKey)
			}
		}
	}

	// Validate file size constraints
	if len(module.Description) > 500 {
		return fmt.Errorf("description too long (max 500 characters)")
	}
	if len(module.Tags) > 20 {
		return fmt.Errorf("too many tags (max 20)")
	}
	for i, tag := range module.Tags {
		if len(tag) > 50 {
			return fmt.Errorf("tag %d too long (max 50 characters)", i)
		}
	}
	if module.Metadata.Author != "" && len(module.Metadata.Author) > 100 {
		return fmt.Errorf("author name too long (max 100 characters)")
	}

	return nil
}

// APIUpload handles module file uploads
func (h *Handlers) APIUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File too large (max 10MB)", http.StatusBadRequest)
		return
	}

	// Check for overwrite flag
	overwrite := r.FormValue("overwrite") == "true"

	file, header, err := r.FormFile("module")
	if err != nil {
		http.Error(w, "Missing module file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".yaml") &&
		!strings.HasSuffix(strings.ToLower(header.Filename), ".yml") {
		http.Error(w, "File must be a YAML file (.yaml or .yml)", http.StatusBadRequest)
		return
	}

	// Read and validate YAML
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Check file size
	if len(data) > 1024*1024 { // 1MB
		http.Error(w, "YAML file too large (max 1MB)", http.StatusBadRequest)
		return
	}

	// Parse YAML
	var module models.Module
	if err := yaml.Unmarshal(data, &module); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"success": false, "error": "Invalid YAML syntax: %s"}`,
			strings.ReplaceAll(err.Error(), `"`, `\"`))
		return
	}

	// Comprehensive validation
	if err := validateModule(&module); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"success": false, "error": "Validation failed: %s"}`,
			strings.ReplaceAll(err.Error(), `"`, `\"`))
		return
	}

	// Check for duplicates
	var existingID int
	var existingFilePath string
	err = h.db.QueryRow("SELECT id, file_path FROM modules WHERE name = ? AND version = ?",
		module.Name, module.Version).Scan(&existingID, &existingFilePath)

	moduleExists := err == nil

	if moduleExists && !overwrite {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, `{"success": false, "error": "Module '%s' version %s already exists. Use overwrite=true to update."}`,
			module.Name, module.Version)
		return
	}

	// Save file
	filename := fmt.Sprintf("%s-%s-%d.yaml", module.Name, module.Version, time.Now().Unix())
	savePath := filepath.Join(h.config.UploadsDir, filename)

	outFile, err := os.Create(savePath)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"success": false, "error": "Failed to save file"}`)
		return
	}
	defer outFile.Close()

	if _, err := outFile.Write(data); err != nil {
		log.Printf("Failed to write file: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"success": false, "error": "Failed to write file"}`)
		return
	}

	// Insert or update database
	username := h.auth.GetUsername(r)

	// Marshal tags to JSON
	tagsJSON := "[]"
	if len(module.Tags) > 0 {
		tagsList := make([]string, len(module.Tags))
		for i, tag := range module.Tags {
			tagsList[i] = fmt.Sprintf("%q", tag)
		}
		tagsJSON = "[" + strings.Join(tagsList, ",") + "]"
	}

	if moduleExists {
		// Update existing module
		_, err = h.db.Exec(`
			UPDATE modules 
			SET description = ?, author = ?, tags = ?, uploaded_by = ?, file_path = ?, original_filename = ?, uploaded_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, module.Description, module.Metadata.Author, tagsJSON, username, savePath, header.Filename, existingID)

		if err != nil {
			log.Printf("Database update error: %v", err)
			os.Remove(savePath) // Clean up new file on DB error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"success": false, "error": "Failed to update module metadata"}`)
			return
		}

		// Delete old file after successful DB update
		if existingFilePath != "" && existingFilePath != savePath {
			if err := os.Remove(existingFilePath); err != nil {
				log.Printf("Warning: Failed to remove old file %s: %v", existingFilePath, err)
			}
		}

		log.Printf("Module updated successfully: %s v%s by %s", module.Name, module.Version, username)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"success": true, "message": "Module '%s' v%s updated successfully"}`,
			module.Name, module.Version)
	} else {
		// Insert new module
		_, err = h.db.Exec(`
			INSERT INTO modules (name, version, description, author, tags, uploaded_by, file_path, original_filename)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, module.Name, module.Version, module.Description,
			module.Metadata.Author, tagsJSON, username, savePath, header.Filename)

		if err != nil {
			log.Printf("Database insert error: %v", err)
			os.Remove(savePath) // Clean up file on DB error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"success": false, "error": "Failed to save module metadata"}`)
			return
		}

		log.Printf("Module uploaded successfully: %s v%s by %s", module.Name, module.Version, username)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"success": true, "message": "Module '%s' v%s uploaded successfully"}`,
			module.Name, module.Version)
	}
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

		// Validate input
		if username == "" || password == "" {
			data := map[string]interface{}{
				"Title": "Login",
				"Error": "Username and password are required",
			}
			if err := h.templates.ExecuteTemplate(w, "login.html", data); err != nil {
				log.Printf("Template error: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		if h.auth.Authenticate(username, password) {
			h.auth.SetSession(w, username)
			http.Redirect(w, r, "/upload", http.StatusSeeOther)
			return
		}

		log.Printf("Failed login attempt for user: %s", username)
		data := map[string]interface{}{
			"Title": "Login",
			"Error": "Invalid username or password. Please try again.",
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
		SELECT id, name, version, description, author, COALESCE(tags, '[]'), downloads
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
		var tagsJSON string
		if err := rows.Scan(&m.ID, &m.Name, &m.Version, &m.Description, &m.Author, &tagsJSON, &m.Downloads); err != nil {
			continue
		}

		if !first {
			w.Write([]byte(","))
		}
		first = false

		fmt.Fprintf(w, `{"id":%d,"name":"%s","version":"%s","description":"%s","author":"%s","tags":%s,"downloads":%d}`,
			m.ID, m.Name, m.Version, m.Description, m.Author, tagsJSON, m.Downloads)
	}

	w.Write([]byte("]"))
}

func (h *Handlers) APIGetModule(w http.ResponseWriter, r *http.Request) {
	// Same as GetModule but with JSON response option
	h.GetModule(w, r)
}
