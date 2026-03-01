package handlers

import (
	"database/sql"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// EnsureAdminUser creates the initial admin user if no users exist in the database
func EnsureAdminUser(db *sql.DB, username, password string) error {
	// Check if any users exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	// If users exist, skip bootstrap
	if count > 0 {
		log.Printf("Bootstrap: Users table already has %d user(s), skipping admin creation", count)
		return nil
	}

	log.Printf("Bootstrap: No users found, creating initial admin user '%s'", username)

	// Hash the password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Insert admin user into users table
	_, err = db.Exec(`
		INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, 'admin', ?, ?)
	`, username, username+"@admin.local", string(hashedPassword), time.Now(), time.Now())

	if err != nil {
		return err
	}

	log.Printf("Bootstrap: Successfully created admin user '%s'", username)
	return nil
}
