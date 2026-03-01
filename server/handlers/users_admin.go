package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user record with additional info
type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	Password     string // Only used for display after creation
	Role         string
	GitHubID     sql.NullString
	AvatarURL    sql.NullString
	CreatedAt    time.Time
	ModuleCount  int
}

// AdminUsersPage shows the user management page
func (h *Handlers) AdminUsersPage(w http.ResponseWriter, r *http.Request) {
	// Check admin authentication
	if !h.auth.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get all users with module counts
	rows, err := h.db.Query(`
		SELECT 
			u.id,
			u.username,
			u.email,
			u.role,
			COALESCE(u.github_id, ''),
			COALESCE(u.avatar_url, ''),
			u.created_at,
			COUNT(m.id) as module_count
		FROM users u
		LEFT JOIN modules m ON m.author = u.username
		GROUP BY u.id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var githubID, avatarURL string
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Role,
			&githubID,
			&avatarURL,
			&user.CreatedAt,
			&user.ModuleCount,
		)
		if err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}

		if githubID != "" {
			user.GitHubID = sql.NullString{String: githubID, Valid: true}
		}
		if avatarURL != "" {
			user.AvatarURL = sql.NullString{String: avatarURL, Valid: true}
		}

		users = append(users, user)
	}

	// Get session and prepare data
	session := h.auth.GetSession(r)
	data := map[string]interface{}{
		"Title":    "User Management",
		"LoggedIn": true,
		"Session":  session,
		"Users":    users,
	}

	// Check for success/new user messages
	if r.URL.Query().Get("success") != "" {
		data["Success"] = r.URL.Query().Get("success")
	}

	// Check for new user data (from session/cookie)
	if username := r.URL.Query().Get("new_username"); username != "" {
		newUser := map[string]string{
			"Username": username,
			"Email":    r.URL.Query().Get("new_email"),
			"Password": r.URL.Query().Get("new_password"),
			"Role":     r.URL.Query().Get("new_role"),
		}
		data["NewUser"] = newUser
		data["Success"] = "User created successfully! Share the credentials below with the new user."
	}

	tmpl, err := template.ParseFiles("server/templates/users-admin.html")
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// CreateUser creates a new user account
func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
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

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	role := r.FormValue("role")

	// Validate required fields
	if username == "" || email == "" || password == "" || role == "" {
		http.Redirect(w, r, "/admin/users?error=All+fields+are+required", http.StatusSeeOther)
		return
	}

	// Validate role
	if role != "user" && role != "admin" {
		http.Redirect(w, r, "/admin/users?error=Invalid+role", http.StatusSeeOther)
		return
	}

	// Check if username already exists
	var existingID int64
	err := h.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&existingID)
	if err != sql.ErrNoRows {
		http.Redirect(w, r, "/admin/users?error=Username+already+exists", http.StatusSeeOther)
		return
	}

	// Check if email already exists
	err = h.db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&existingID)
	if err != sql.ErrNoRows {
		http.Redirect(w, r, "/admin/users?error=Email+already+exists", http.StatusSeeOther)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Insert user
	result, err := h.db.Exec(`
		INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, username, email, string(hashedPassword), role)

	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Redirect(w, r, "/admin/users?error=Failed+to+create+user", http.StatusSeeOther)
		return
	}

	userID, _ := result.LastInsertId()
	log.Printf("User created: %s (ID: %d, Role: %s) by admin", username, userID, role)

	// Redirect with new user credentials (shown once)
	redirectURL := "/admin/users?new_username=" + username +
		"&new_email=" + email +
		"&new_password=" + password +
		"&new_role=" + role
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// DeleteUser deletes a user account
func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
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

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if user is admin (prevent deleting admin users)
	var role string
	var username string
	err = h.db.QueryRow("SELECT username, role FROM users WHERE id = ?", userID).Scan(&username, &role)
	if err != nil {
		log.Printf("Error fetching user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if role == "admin" {
		http.Redirect(w, r, "/admin/users?error=Cannot+delete+admin+users", http.StatusSeeOther)
		return
	}

	// Start transaction to delete user and their modules/API keys
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Delete user's API keys
	_, err = tx.Exec("DELETE FROM api_keys WHERE user_id = ?", userID)
	if err != nil {
		log.Printf("Error deleting API keys: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// Delete user's modules (optional - you might want to keep them)
	_, err = tx.Exec("DELETE FROM modules WHERE author = ?", username)
	if err != nil {
		log.Printf("Error deleting modules: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// Delete user
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		log.Printf("Error deleting user: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	log.Printf("User deleted: %s (ID: %d) with all associated data", username, userID)
	http.Redirect(w, r, "/admin/users?success=User+deleted+successfully", http.StatusSeeOther)
}
