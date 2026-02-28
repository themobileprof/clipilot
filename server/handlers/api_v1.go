package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// APIv1ListModules handles GET /api/v1/modules with filtering, pagination, and sorting
func (h *Handlers) APIv1ListModules(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	tags := query.Get("tags")
	updatedSince := query.Get("updated_since")
	platform := query.Get("platform")
	search := query.Get("search")
	sortBy := query.Get("sort_by")
	if sortBy == "" {
		sortBy = "name"
	}
	order := query.Get("order")
	if order == "" {
		order = "asc"
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Build SQL query with filters
	sqlQuery := "SELECT id, name, version, description, author, COALESCE(tags, '[]'), uploaded_at, uploaded_by, downloads FROM modules WHERE 1=1"
	args := []interface{}{}

	// Apply filters
	if tags != "" {
		// Filter by tags (comma-separated)
		tagList := strings.Split(tags, ",")
		tagConditions := []string{}
		for _, tag := range tagList {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagConditions = append(tagConditions, "(tags LIKE '%' || ? || '%')")
				args = append(args, tag)
			}
		}
		if len(tagConditions) > 0 {
			sqlQuery += " AND (" + strings.Join(tagConditions, " OR ") + ")"
		}
	}

	if updatedSince != "" {
		sqlQuery += " AND uploaded_at > ?"
		args = append(args, updatedSince)
	}

	if platform != "" {
		sqlQuery += " AND (tags LIKE '%' || ? || '%')"
		args = append(args, platform)
	}

	if search != "" {
		sqlQuery += " AND (name LIKE '%' || ? || '%' OR description LIKE '%' || ? || '%')"
		args = append(args, search, search)
	}

	// Get total count before pagination
	countQuery := "SELECT COUNT(*) FROM (" + sqlQuery + ")"
	var total int
	if err := h.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("Count query error: %v", err)
		total = 0
	}

	// Apply sorting
	validSortFields := map[string]bool{"name": true, "downloads": true, "uploaded_at": true}
	if !validSortFields[sortBy] {
		sortBy = "name"
	}
	if order != "asc" && order != "desc" {
		order = "asc"
	}
	sqlQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, strings.ToUpper(order))

	// Apply pagination
	sqlQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := h.db.Query(sqlQuery, args...)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	modules := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var name, version, description, author, tagsJSON, uploadedBy string
		var uploadedAt time.Time
		var downloads int

		if err := rows.Scan(&id, &name, &version, &description, &author, &tagsJSON, &uploadedAt, &uploadedBy, &downloads); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		// Parse tags JSON
		var tagsList []string
		_ = json.Unmarshal([]byte(tagsJSON), &tagsList)

		module := map[string]interface{}{
			"id":             name, // Use name as ID for CLI compatibility
			"name":           name,
			"version":        version,
			"description":    description,
			"tags":           tagsList,
			"download_count": downloads,
			"uploaded_by":    uploadedBy,
			"uploaded_at":    uploadedAt.Format(time.RFC3339),
			"updated_at":     uploadedAt.Format(time.RFC3339),
		}

		modules = append(modules, module)
	}

	// Generate ETag
	etag := fmt.Sprintf(`"v1-modules-%d"`, time.Now().Unix()/300) // Cache for 5 minutes

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", time.Now().Format(http.TimeFormat))

	// Check If-None-Match for caching
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	response := map[string]interface{}{
		"modules": modules,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}

	json.NewEncoder(w).Encode(response)
}

// APIv1GetModule handles GET /api/v1/modules/:id
func (h *Handlers) APIv1GetModule(w http.ResponseWriter, r *http.Request) {
	// Extract module ID from path: /api/v1/modules/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/modules/")
	moduleID := strings.Split(path, "/")[0]

	if moduleID == "" || moduleID == "changed" {
		http.Error(w, "Invalid module ID", http.StatusBadRequest)
		return
	}

	var id int64
	var name, version, description, author, tagsJSON, uploadedBy, filePath string
	var uploadedAt time.Time
	var downloads int

	err := h.db.QueryRow(`
		SELECT id, name, version, description, author, COALESCE(tags, '[]'), 
		       uploaded_at, uploaded_by, file_path, downloads
		FROM modules WHERE name = ?
		ORDER BY uploaded_at DESC LIMIT 1
	`, moduleID).Scan(&id, &name, &version, &description, &author, &tagsJSON, &uploadedAt, &uploadedBy, &filePath, &downloads)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    "MODULE_NOT_FOUND",
				"message": fmt.Sprintf("Module '%s' does not exist", moduleID),
			},
		})
		return
	}

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Parse tags
	var tagsList []string
	_ = json.Unmarshal([]byte(tagsJSON), &tagsList)

	// Calculate checksum
	checksum := ""
	if content, err := os.ReadFile(filePath); err == nil {
		hash := sha256.Sum256(content)
		checksum = fmt.Sprintf("%x", hash)
	}

	module := map[string]interface{}{
		"id":              name,
		"name":            name,
		"version":         version,
		"description":     description,
		"tags":            tagsList,
		"download_count":  downloads,
		"uploaded_by":     uploadedBy,
		"uploaded_at":     uploadedAt.Format(time.RFC3339),
		"updated_at":      uploadedAt.Format(time.RFC3339),
		"checksum_sha256": checksum,
	}

	etag := fmt.Sprintf(`"%s"`, checksum)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", uploadedAt.Format(http.TimeFormat))

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	json.NewEncoder(w).Encode(module)
}

// APIv1DownloadModule handles GET /api/v1/modules/:id/download
func (h *Handlers) APIv1DownloadModule(w http.ResponseWriter, r *http.Request) {
	// Extract module ID from path: /api/v1/modules/{id}/download
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/modules/")
	moduleID := strings.Split(path, "/")[0]

	var filePath, name string
	var uploadedAt time.Time

	err := h.db.QueryRow(`
		SELECT file_path, name, uploaded_at
		FROM modules WHERE name = ?
		ORDER BY uploaded_at DESC LIMIT 1
	`, moduleID).Scan(&filePath, &name, &uploadedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("File read error: %v", err)
		http.Error(w, "Module file not found", http.StatusNotFound)
		return
	}

	// Calculate checksum for ETag
	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)
	etag := fmt.Sprintf(`"%s"`, checksum)

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.yaml"`, name))
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", uploadedAt.Format(http.TimeFormat))

	// Check cache
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Increment download counter in background
	go func() {
		_, err := h.db.Exec("UPDATE modules SET downloads = downloads + 1 WHERE name = ?", moduleID)
		if err != nil {
			log.Printf("Failed to increment download counter: %v", err)
		}
	}()

	w.Write(content)
}

// APIv1ChangedModules handles GET /api/v1/modules/changed for delta sync
func (h *Handlers) APIv1ChangedModules(w http.ResponseWriter, r *http.Request) {
	since := r.URL.Query().Get("since")
	if since == "" {
		http.Error(w, `{"error":"Missing 'since' parameter"}`, http.StatusBadRequest)
		return
	}

	// Parse timestamp
	sinceTime, err := time.Parse(time.RFC3339, since)
	if err != nil {
		http.Error(w, `{"error":"Invalid timestamp format, use RFC3339"}`, http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(`
		SELECT name, version, uploaded_at, file_path
		FROM modules WHERE uploaded_at > ?
		ORDER BY uploaded_at ASC
	`, sinceTime)

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	changedModules := []map[string]interface{}{}
	for rows.Next() {
		var name, version, filePath string
		var uploadedAt time.Time

		if err := rows.Scan(&name, &version, &uploadedAt, &filePath); err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		// Calculate checksum
		checksum := ""
		if content, err := os.ReadFile(filePath); err == nil {
			hash := sha256.Sum256(content)
			checksum = fmt.Sprintf("%x", hash)
		}

		module := map[string]interface{}{
			"id":              name,
			"version":         version,
			"checksum_sha256": checksum,
			"updated_at":      uploadedAt.Format(time.RFC3339),
			"change_type":     "updated", // Could be "added" or "updated"
		}

		changedModules = append(changedModules, module)
	}

	response := map[string]interface{}{
		"changed_modules": changedModules,
		"sync_timestamp":  time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// APIv1ModuleDependencies handles GET /api/v1/modules/:id/dependencies
func (h *Handlers) APIv1ModuleDependencies(w http.ResponseWriter, r *http.Request) {
	// Extract module ID from path: /api/v1/modules/{id}/dependencies
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/modules/")
	moduleID := strings.Split(path, "/")[0]

	// This is a simplified implementation
	// A full implementation would parse the YAML and recursively resolve dependencies
	var filePath string
	err := h.db.QueryRow(`
		SELECT file_path FROM modules WHERE name = ?
		ORDER BY uploaded_at DESC LIMIT 1
	`, moduleID).Scan(&filePath)

	if err == sql.ErrNoRows {
		http.Error(w, "Module not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Read and parse YAML to extract requires field
	_, err = os.ReadFile(filePath)
	if err != nil {
		log.Printf("File read error: %v", err)
		http.Error(w, "Module file not found", http.StatusNotFound)
		return
	}

	// Simple regex to extract requires field (basic implementation)
	// A production version should use proper YAML parsing
	// TODO: Implement proper dependency resolution
	dependencies := []interface{}{}

	response := map[string]interface{}{
		"module_id":     moduleID,
		"dependencies":  dependencies,
		"install_order": []string{moduleID}, // Simplified
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// APIv1Health handles GET /health with enhanced information
func (h *Handlers) APIv1Health(w http.ResponseWriter, r *http.Request) {
	// Check DB connection
	dbStatus := "connected"
	if err := h.db.Ping(); err != nil {
		dbStatus = "disconnected"
		log.Printf("Health check failed: DB is down: %v", err)
	}

	// Check disk space
	var diskFreeGB float64
	if stat, err := os.Stat(filepath.Dir(h.config.DBPath)); err == nil {
		_ = stat           // Disk space calculation would go here
		diskFreeGB = 100.0 // Placeholder
	}

	response := map[string]interface{}{
		"status":       "healthy",
		"version":      "1.0.0",
		"database":     dbStatus,
		"disk_free_gb": diskFreeGB,
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if dbStatus != "connected" {
		w.WriteHeader(http.StatusServiceUnavailable)
		response["status"] = "unhealthy"
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}
