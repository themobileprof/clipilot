package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ModuleRequest represents a user request for a missing module
type ModuleRequest struct {
	ID                int64     `json:"id"`
	Query             string    `json:"query"`
	UserContext       string    `json:"user_context,omitempty"`
	IPAddress         string    `json:"ip_address,omitempty"`
	UserAgent         string    `json:"user_agent,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	Status            string    `json:"status"`
	DuplicateOf       *int64    `json:"duplicate_of,omitempty"`
	Notes             string    `json:"notes,omitempty"`
	FulfilledByModule string    `json:"fulfilled_by_module,omitempty"`
}

// APIModuleRequest handles POST /api/module-request
// This is called by CLIPilot clients when no matching module is found
func (h *Handlers) APIModuleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query       string `json:"query"`
		UserContext string `json:"user_context,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate query
	query := strings.TrimSpace(req.Query)
	if query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Get client info
	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()

	// Insert request into database
	result, err := h.db.Exec(`
		INSERT INTO module_requests (query, user_context, ip_address, user_agent)
		VALUES (?, ?, ?, ?)
	`, query, req.UserContext, ipAddress, userAgent)

	if err != nil {
		log.Printf("Failed to insert module request: %v", err)
		http.Error(w, "Failed to save request", http.StatusInternalServerError)
		return
	}

	requestID, _ := result.LastInsertId()

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"message":    "Thank you! Your request has been received. Our community is working to expand the module library, and your feedback helps us prioritize what to build next.",
		"request_id": requestID,
		"status":     "pending",
		"note":       "You can check https://clipilot.themobileprof.com for new modules, or contribute by logging in with GitHub.",
	}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// ModuleRequestsPage shows admin view of all module requests
func (h *Handlers) ModuleRequestsPage(w http.ResponseWriter, r *http.Request) {
	session := h.auth.GetSession(r)

	// Only admins can view requests
	if session == nil || !session.IsAdmin {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get status filter from query params
	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "pending"
	}

	// Query requests
	var rows *sql.Rows
	var err error

	if statusFilter == "all" {
		rows, err = h.db.Query(`
			SELECT id, query, user_context, ip_address, user_agent, created_at, 
			       status, duplicate_of, notes, fulfilled_by_module
			FROM module_requests
			ORDER BY created_at DESC
			LIMIT 500
		`)
	} else {
		rows, err = h.db.Query(`
			SELECT id, query, user_context, ip_address, user_agent, created_at, 
			       status, duplicate_of, notes, fulfilled_by_module
			FROM module_requests
			WHERE status = ?
			ORDER BY created_at DESC
			LIMIT 500
		`, statusFilter)
	}

	if err != nil {
		http.Error(w, "Failed to fetch requests", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var requests []ModuleRequest
	for rows.Next() {
		var req ModuleRequest
		var duplicateOf sql.NullInt64
		var notes, fulfilled sql.NullString

		err := rows.Scan(
			&req.ID, &req.Query, &req.UserContext, &req.IPAddress, &req.UserAgent,
			&req.CreatedAt, &req.Status, &duplicateOf, &notes, &fulfilled,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		if duplicateOf.Valid {
			req.DuplicateOf = &duplicateOf.Int64
		}
		if notes.Valid {
			req.Notes = notes.String
		}
		if fulfilled.Valid {
			req.FulfilledByModule = fulfilled.String
		}

		requests = append(requests, req)
	}

	// Get counts by status
	counts := make(map[string]int)
	countRows, err := h.db.Query(`
		SELECT status, COUNT(*) as count
		FROM module_requests
		GROUP BY status
	`)
	if err == nil {
		defer countRows.Close()
		for countRows.Next() {
			var status string
			var count int
			if err := countRows.Scan(&status, &count); err == nil {
				counts[status] = count
			}
		}
	}

	data := map[string]interface{}{
		"Title":              "Module Requests",
		"Session":            session,
		"Requests":           requests,
		"StatusFilter":       statusFilter,
		"StatusCounts":       counts,
		"GitHubOAuthEnabled": h.githubOAuth != nil,
	}

	if err := h.templates.ExecuteTemplate(w, "module_requests.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// APIUpdateModuleRequest handles PUT/PATCH /api/module-request/:id
// Allows admins to update status, notes, etc.
func (h *Handlers) APIUpdateModuleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session := h.auth.GetSession(r)
	if session == nil || !session.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract ID from URL
	path := strings.TrimPrefix(r.URL.Path, "/api/module-request/")
	var requestID int64
	if _, err := fmt.Sscanf(path, "%d", &requestID); err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	var update struct {
		Status            *string `json:"status,omitempty"`
		Notes             *string `json:"notes,omitempty"`
		FulfilledByModule *string `json:"fulfilled_by_module,omitempty"`
		DuplicateOf       *int64  `json:"duplicate_of,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build update query dynamically
	var updates []string
	var args []interface{}

	if update.Status != nil {
		updates = append(updates, "status = ?")
		args = append(args, *update.Status)
	}
	if update.Notes != nil {
		updates = append(updates, "notes = ?")
		args = append(args, *update.Notes)
	}
	if update.FulfilledByModule != nil {
		updates = append(updates, "fulfilled_by_module = ?")
		args = append(args, *update.FulfilledByModule)
	}
	if update.DuplicateOf != nil {
		updates = append(updates, "duplicate_of = ?")
		args = append(args, *update.DuplicateOf)
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	args = append(args, requestID)
	query := fmt.Sprintf("UPDATE module_requests SET %s WHERE id = ?", strings.Join(updates, ", "))

	if _, err := h.db.Exec(query, args...); err != nil {
		log.Printf("Failed to update request: %v", err)
		http.Error(w, "Failed to update request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Request updated successfully",
	}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header (set by some proxies)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
