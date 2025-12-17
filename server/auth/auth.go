package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Manager struct {
	adminUser string
	adminPass string
	sessions  map[string]*Session
	mu        sync.RWMutex
}

type Session struct {
	Username   string
	IsAdmin    bool
	GitHubUser *GitHubUserInfo
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

type GitHubUserInfo struct {
	Login     string
	AvatarURL string
	Name      string
}

const (
	sessionCookie = "clipilot_session"
	sessionTTL    = 24 * time.Hour
)

func NewManager(adminUser, adminPass string) *Manager {
	m := &Manager{
		adminUser: adminUser,
		adminPass: adminPass,
		sessions:  make(map[string]*Session),
	}

	// Start cleanup goroutine
	go m.cleanupExpiredSessions()

	return m
}

// Authenticate checks credentials
func (m *Manager) Authenticate(username, password string) bool {
	return username == m.adminUser && password == m.adminPass
}

// SetSession creates a new session for admin user
func (m *Manager) SetSession(w http.ResponseWriter, username string) {
	token := m.generateToken()

	session := &Session{
		Username:  username,
		IsAdmin:   true,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionTTL),
	}

	m.mu.Lock()
	m.sessions[token] = session
	m.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

// SetGitHubSession creates a new session for GitHub user
func (m *Manager) SetGitHubSession(w http.ResponseWriter, ghUser *GitHubUser) {
	token := m.generateToken()

	session := &Session{
		Username: ghUser.Login,
		IsAdmin:  false,
		GitHubUser: &GitHubUserInfo{
			Login:     ghUser.Login,
			AvatarURL: ghUser.AvatarURL,
			Name:      ghUser.Name,
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(sessionTTL),
	}

	m.mu.Lock()
	m.sessions[token] = session
	m.mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

// ClearSession removes a session
func (m *Manager) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// IsAuthenticated checks if request has valid session
func (m *Manager) IsAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}

	m.mu.RLock()
	session, exists := m.sessions[cookie.Value]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(session.ExpiresAt) {
		m.mu.Lock()
		delete(m.sessions, cookie.Value)
		m.mu.Unlock()
		return false
	}

	return true
}

// GetUsername returns username from session
func (m *Manager) GetUsername(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return ""
	}

	m.mu.RLock()
	session, exists := m.sessions[cookie.Value]
	m.mu.RUnlock()

	if !exists {
		return ""
	}

	return session.Username
}

// GetSession returns the full session
func (m *Manager) GetSession(r *http.Request) *Session {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}

	m.mu.RLock()
	session, exists := m.sessions[cookie.Value]
	m.mu.RUnlock()

	if !exists || time.Now().After(session.ExpiresAt) {
		return nil
	}

	return session
}

// IsAdmin checks if the current session is admin
func (m *Manager) IsAdmin(r *http.Request) bool {
	session := m.GetSession(r)
	return session != nil && session.IsAdmin
}

// generateToken creates a random session token
func (m *Manager) generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// This should never fail, but fallback to timestamp-based token
		return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return base64.URLEncoding.EncodeToString(b)
}

// cleanupExpiredSessions removes old sessions periodically
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		m.mu.Lock()
		for token, session := range m.sessions {
			if now.After(session.ExpiresAt) {
				delete(m.sessions, token)
			}
		}
		m.mu.Unlock()
	}
}
