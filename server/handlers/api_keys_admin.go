package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// APIKey represents an API key record
type APIKey struct {
	ID         int64
	UserID     int64
	KeyHash    string
	KeyPreview string // First 16 chars for display
	Name       string
	Scopes     string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	Revoked    bool
}

// AdminAPIKeysPage shows the API key management page
func (h *Handlers) AdminAPIKeysPage(w http.ResponseWriter, r *http.Request) {
	// Check admin authentication
	if !h.auth.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get all API keys for admin users
	rows, err := h.db.Query(`
		SELECT 
			ak.id, 
			ak.user_id, 
			ak.key_hash, 
			ak.name, 
			ak.scopes, 
			ak.created_at, 
			COALESCE(ak.expires_at, ''),
			ak.revoked
		FROM api_keys ak
		ORDER BY ak.created_at DESC
	`)
	if err != nil {
		log.Printf("Error fetching API keys: %v", err)
		http.Error(w, "Failed to load API keys", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apiKeys []APIKey
	for rows.Next() {
		var key APIKey
		var expiresAtStr string
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.KeyHash,
			&key.Name,
			&key.Scopes,
			&key.CreatedAt,
			&expiresAtStr,
			&key.Revoked,
		)
		if err != nil {
			log.Printf("Error scanning API key: %v", err)
			continue
		}

		// Parse expires_at if present
		if expiresAtStr != "" {
			key.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAtStr)
		}

		// Create preview (first 16 chars of hash)
		if len(key.KeyHash) > 16 {
			key.KeyPreview = key.KeyHash[:16] + "..."
		} else {
			key.KeyPreview = key.KeyHash
		}

		apiKeys = append(apiKeys, key)
	}

	// Get any flash messages from session
	session := h.auth.GetSession(r)
	data := map[string]interface{}{
		"Title":    "API Keys",
		"LoggedIn": true,
		"Session":  session,
		"APIKeys":  apiKeys,
	}

	if err := h.templates.ExecuteTemplate(w, "api-keys.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

// GenerateAPIKey creates a new API key
func (h *Handlers) GenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check admin authentication
	if !h.auth.IsAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	scopes := r.FormValue("scopes")
	expiresDaysStr := r.FormValue("expires_days")

	if name == "" || scopes == "" {
		http.Error(w, "Name and scopes are required", http.StatusBadRequest)
		return
	}

	// Validate scopes is valid JSON
	var scopesArray []string
	if err := json.Unmarshal([]byte(scopes), &scopesArray); err != nil {
		http.Error(w, "Invalid scopes format", http.StatusBadRequest)
		return
	}

	// Generate random API key (32 bytes = 64 hex chars)
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		log.Printf("Error generating random key: %v", err)
		http.Error(w, "Failed to generate key", http.StatusInternalServerError)
		return
	}
	apiKey := "clipilot_" + base64.RawURLEncoding.EncodeToString(keyBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := fmt.Sprintf("%x", hash)

	// Calculate expiration if provided
	var expiresAt sql.NullTime
	if expiresDaysStr != "" {
		days, err := strconv.Atoi(expiresDaysStr)
		if err == nil && days > 0 {
			expiresAt.Valid = true
			expiresAt.Time = time.Now().AddDate(0, 0, days)
		}
	}

	// Get admin user ID
	session := h.auth.GetSession(r)
	var userID int64
	err := h.db.QueryRow("SELECT id FROM users WHERE username = ? AND role = 'admin' LIMIT 1",
		session.Username).Scan(&userID)
	if err != nil {
		log.Printf("Error finding admin user: %v", err)
		http.Error(w, "Failed to find user", http.StatusInternalServerError)
		return
	}

	// Insert into database
	_, err = h.db.Exec(`
		INSERT INTO api_keys (user_id, key_hash, name, scopes, expires_at, revoked, created_at)
		VALUES (?, ?, ?, ?, ?, 0, CURRENT_TIMESTAMP)
	`, userID, keyHash, name, scopes, expiresAt)

	if err != nil {
		log.Printf("Error inserting API key: %v", err)
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Redirect back with the new key displayed
	// We'll pass it as a query parameter (only shown once)
	http.Redirect(w, r, "/admin/api-keys?new_key="+apiKey, http.StatusSeeOther)
}

// RevokeAPIKey revokes an existing API key
func (h *Handlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check admin authentication
	if !h.auth.IsAdmin(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	keyIDStr := r.FormValue("key_id")
	keyID, err := strconv.ParseInt(keyIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid key ID", http.StatusBadRequest)
		return
	}

	// Revoke the key
	_, err = h.db.Exec("UPDATE api_keys SET revoked = 1 WHERE id = ?", keyID)
	if err != nil {
		log.Printf("Error revoking API key: %v", err)
		http.Error(w, "Failed to revoke key", http.StatusInternalServerError)
		return
	}

	log.Printf("API key %d revoked by admin", keyID)
	http.Redirect(w, r, "/admin/api-keys?success=Key+revoked+successfully", http.StatusSeeOther)
}

// Updated AdminAPIKeysPage to handle flash messages
func (h *Handlers) AdminAPIKeysPageWithFlash(w http.ResponseWriter, r *http.Request) {
	// Check admin authentication
	if !h.auth.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get all API keys for admin users
	rows, err := h.db.Query(`
		SELECT 
			ak.id, 
			ak.user_id, 
			ak.key_hash, 
			ak.name, 
			ak.scopes, 
			ak.created_at, 
			COALESCE(ak.expires_at, ''),
			ak.revoked
		FROM api_keys ak
		JOIN users u ON ak.user_id = u.id
		WHERE u.role = 'admin'
		ORDER BY ak.created_at DESC
	`)
	if err != nil {
		log.Printf("Error fetching API keys: %v", err)
		http.Error(w, "Failed to load API keys", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apiKeys []APIKey
	for rows.Next() {
		var key APIKey
		var expiresAtStr string
		err := rows.Scan(
			&key.ID,
			&key.UserID,
			&key.KeyHash,
			&key.Name,
			&key.Scopes,
			&key.CreatedAt,
			&expiresAtStr,
			&key.Revoked,
		)
		if err != nil {
			log.Printf("Error scanning API key: %v", err)
			continue
		}

		// Parse expires_at if present
		if expiresAtStr != "" {
			key.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expiresAtStr)
		}

		// Create preview (first 16 chars of hash)
		if len(key.KeyHash) > 16 {
			key.KeyPreview = key.KeyHash[:16] + "..."
		} else {
			key.KeyPreview = key.KeyHash
		}

		apiKeys = append(apiKeys, key)
	}

	// Get session and flash messages
	session := h.auth.GetSession(r)
	data := map[string]interface{}{
		"Title":    "API Keys",
		"LoggedIn": true,
		"Session":  session,
		"APIKeys":  apiKeys,
	}

	// Check for new key in query params
	if newKey := r.URL.Query().Get("new_key"); newKey != "" {
		data["NewAPIKey"] = newKey
		data["Success"] = "API key generated successfully! Copy it now - it won't be shown again."
	}

	// Check for success message
	if success := r.URL.Query().Get("success"); success != "" {
		data["Success"] = success
	}

	if err := h.templates.ExecuteTemplate(w, "api-keys.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}
