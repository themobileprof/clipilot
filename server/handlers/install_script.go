package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadInstallScript handles POST /api/install-script/upload
// Requires authentication (session or API key with admin role)
// Accepts multipart/form-data with fields:
// - file: the install script (install.sh)
// - version: semantic version (e.g., "v1.0.0")
func (h *Handlers) UploadInstallScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication - support both session and API key
	authorized := false
	var username string

	// Try session-based auth first
	if h.auth.IsAdmin(r) {
		authorized = true
		username = h.auth.GetUsername(r)
	}

	// Try Bearer token auth (for CI/CD)
	if !authorized {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			apiKey := strings.TrimPrefix(authHeader, "Bearer ")
			// Validate API key and check admin role
			var userID int64
			var role string
			err := h.db.QueryRow(`
				SELECT u.id, u.username, u.role
				FROM api_keys ak
				JOIN users u ON ak.user_id = u.id
				WHERE ak.key_hash = ? 
				  AND ak.revoked = 0 
				  AND (ak.expires_at IS NULL OR ak.expires_at > CURRENT_TIMESTAMP)
			`, hashAPIKey(apiKey)).Scan(&userID, &username, &role)

			if err == nil && role == "admin" {
				authorized = true
			}
		}
	}

	if !authorized {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error": "Unauthorized. Admin access required.",
			"hint":  "Use session authentication or provide a valid API key with admin role.",
		}); err != nil {
			log.Printf("Failed to encode unauthorized response: %v", err)
		}
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		log.Printf("Failed to parse multipart form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Get version parameter
	version := r.FormValue("version")
	if version == "" {
		http.Error(w, "Missing 'version' parameter", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Failed to get file from form: %v", err)
		http.Error(w, "Missing 'file' parameter", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate filename
	if !strings.HasSuffix(header.Filename, ".sh") {
		http.Error(w, "File must be a shell script (.sh)", http.StatusBadRequest)
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Failed to read file: %v", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Validate script content (basic checks)
	scriptContent := string(content)
	if !strings.Contains(scriptContent, "#!/") {
		http.Error(w, "Invalid script: missing shebang", http.StatusBadRequest)
		return
	}

	// Calculate checksum
	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)

	// Get file size
	fileSize := int64(len(content))

	// Save file to uploads directory
	uploadDir := filepath.Join(h.config.UploadsDir, "install_scripts")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Failed to create upload directory: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Filename format: install-{version}-{checksum[:8]}.sh
	filename := fmt.Sprintf("install-%s-%s.sh", version, checksum[:8])
	filePath := filepath.Join(uploadDir, filename)

	// Write file
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		log.Printf("Failed to write file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Get uploader user ID
	var uploaderID int64
	err = h.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&uploaderID)
	if err != nil {
		// If user doesn't exist in new users table, use NULL
		uploaderID = 0
	}

	// Deactivate previous version if exists
	_, err = h.db.Exec("UPDATE install_scripts SET is_active = 0 WHERE is_active = 1")
	if err != nil {
		log.Printf("Warning: failed to deactivate old scripts: %v", err)
	}

	// Insert into database
	var uploaderIDPtr *int64
	if uploaderID > 0 {
		uploaderIDPtr = &uploaderID
	}

	_, err = h.db.Exec(`
		INSERT INTO install_scripts (version, file_path, checksum_sha256, size_bytes, uploaded_by, is_active, uploaded_at)
		VALUES (?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP)
	`, version, filePath, checksum, fileSize, uploaderIDPtr)

	if err != nil {
		log.Printf("Failed to insert install script record: %v", err)
		// Clean up file
		os.Remove(filePath)
		http.Error(w, "Failed to save script metadata", http.StatusInternalServerError)
		return
	}

	log.Printf("Install script uploaded: version=%s, user=%s, checksum=%s", version, username, checksum[:8])

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"version":  version,
		"checksum": checksum,
		"filename": filename,
		"message":  "Install script uploaded successfully",
	}); err != nil {
		log.Printf("Failed to encode upload response: %v", err)
	}
}

// GetInstallScript serves the latest active install script at GET /clio
// Public endpoint - no authentication required
func (h *Handlers) GetInstallScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Query for latest active install script
	var filePath, version, checksum string
	var uploadedAt time.Time

	err := h.db.QueryRow(`
		SELECT file_path, version, checksum_sha256, uploaded_at
		FROM install_scripts
		WHERE is_active = 1
		ORDER BY uploaded_at DESC
		LIMIT 1
	`).Scan(&filePath, &version, &checksum, &uploadedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Install script not available. Please check back later.", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Failed to query install script: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Install script file missing: %s", filePath)
		http.Error(w, "Install script file not found", http.StatusNotFound)
		return
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Failed to read install script: %v", err)
		http.Error(w, "Failed to read install script", http.StatusInternalServerError)
		return
	}

	// Verify checksum
	hash := sha256.Sum256(content)
	actualChecksum := fmt.Sprintf("%x", hash)
	if actualChecksum != checksum {
		log.Printf("Checksum mismatch for install script: expected=%s, actual=%s", checksum, actualChecksum)
		http.Error(w, "Install script integrity check failed", http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "text/x-shellscript")
	w.Header().Set("Content-Disposition", "inline; filename=\"install.sh\"")
	w.Header().Set("X-Script-Version", version)
	w.Header().Set("X-Script-Checksum", checksum)
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Check ETag for caching
	etag := fmt.Sprintf(`"%s"`, checksum[:16])
	w.Header().Set("ETag", etag)

	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Serve content
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(content); err != nil {
		log.Printf("Failed to write install script: %v", err)
		return
	}

	log.Printf("Served install script: version=%s, size=%d bytes", version, len(content))
}

// ListInstallScripts returns all install scripts (for admin dashboard)
// GET /api/install-scripts
func (h *Handlers) ListInstallScripts(w http.ResponseWriter, r *http.Request) {
	// Require admin authentication
	if !h.auth.IsAdmin(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Admin access required"}); err != nil {
			log.Printf("Failed to encode unauthorized response: %v", err)
		}
		return
	}

	rows, err := h.db.Query(`
		SELECT 
			id, version, checksum_sha256, is_active, uploaded_at,
			COALESCE((SELECT username FROM users WHERE id = uploaded_by), 'unknown') as uploader
		FROM install_scripts
		ORDER BY uploaded_at DESC
	`)
	if err != nil {
		log.Printf("Failed to query install scripts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type InstallScript struct {
		ID         int64  `json:"id"`
		Version    string `json:"version"`
		Checksum   string `json:"checksum_sha256"`
		IsActive   bool   `json:"is_active"`
		Uploader   string `json:"uploaded_by"`
		UploadedAt string `json:"uploaded_at"`
	}

	var scripts []InstallScript
	for rows.Next() {
		var s InstallScript
		var uploadedAt time.Time
		if err := rows.Scan(&s.ID, &s.Version, &s.Checksum, &s.IsActive, &uploadedAt, &s.Uploader); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		s.UploadedAt = uploadedAt.Format(time.RFC3339)
		scripts = append(scripts, s)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"scripts": scripts,
		"count":   len(scripts),
	}); err != nil {
		log.Printf("Failed to encode scripts list: %v", err)
	}
}

// ActivateInstallScript sets a specific version as active
// POST /api/install-scripts/:id/activate
func (h *Handlers) ActivateInstallScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Require admin authentication
	if !h.auth.IsAdmin(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "Admin access required"}); err != nil {
			log.Printf("Failed to encode unauthorized response: %v", err)
		}
		return
	}

	// Extract script ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/install-scripts/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "activate" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	scriptID := parts[0]

	// Deactivate all scripts
	_, err := h.db.Exec("UPDATE install_scripts SET is_active = 0")
	if err != nil {
		log.Printf("Failed to deactivate scripts: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Activate the specified script
	result, err := h.db.Exec("UPDATE install_scripts SET is_active = 1 WHERE id = ?", scriptID)
	if err != nil {
		log.Printf("Failed to activate script: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Install script not found", http.StatusNotFound)
		return
	}

	log.Printf("Activated install script ID: %s", scriptID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Install script activated",
	}); err != nil {
		log.Printf("Failed to encode activation response: %v", err)
	}
}

// hashAPIKey creates a SHA256 hash of an API key for storage/comparison
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}
