package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"

	"github.com/themobileprof/clipilot/server/auth"
)

// GitHubLogin redirects to GitHub OAuth
func (h *Handlers) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	if h.githubOAuth == nil {
		http.Error(w, "GitHub OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Generate random state for CSRF protection
	state := generateState()

	// Store state in session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   600, // 10 minutes
	})

	// Redirect to GitHub
	url := h.githubOAuth.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GitHubCallback handles the OAuth callback from GitHub
func (h *Handlers) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	if h.githubOAuth == nil {
		http.Error(w, "GitHub OAuth not configured", http.StatusServiceUnavailable)
		return
	}

	// Verify state for CSRF protection
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	stateParam := r.URL.Query().Get("state")
	if stateParam != stateCookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	token, err := h.githubOAuth.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("OAuth exchange error: %v", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Get GitHub user info
	ghUser, err := auth.GetGitHubUser(context.Background(), token, h.githubOAuth)
	if err != nil {
		log.Printf("Failed to get GitHub user: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Create session for GitHub user
	h.auth.SetGitHubSession(w, ghUser)

	log.Printf("GitHub user logged in: %s (%s)", ghUser.Login, ghUser.Name)

	// Redirect to home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// generateState creates a random state string for OAuth CSRF protection
func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
