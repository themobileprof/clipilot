package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
    "bytes"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/themobileprof/clipilot/pkg/models"
)

// Client handles communication with the module registry
type Client struct {
	db          *sql.DB
	registryURL string
	httpClient  *http.Client
}

// ModuleMetadata represents cached module information from registry
type ModuleMetadata struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Downloads   int      `json:"downloads"`
}

// SyncStatus represents registry cache status
type SyncStatus struct {
	LastSync      time.Time
	TotalModules  int
	CachedModules int
	Status        string
	Error         string
}

// NewClient creates a new registry client with DNS fallback support
func NewClient(db *sql.DB, registryURL string) *Client {
	return &Client{
		db:          db,
		registryURL: strings.TrimSuffix(registryURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: newTransportWithDNSFallback(),
		},
	}
}

// newTransportWithDNSFallback creates a transport with DNS fallback for Termux/Android
func newTransportWithDNSFallback() *http.Transport {
	// DNS fallback map for known registries (avoids Termux DNS issues)
	dnsFallback := map[string]string{
		"clipilot.themobileprof.com": "157.230.148.144",
	}

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Try normal DNS first
			conn, err := dialer.DialContext(ctx, network, addr)
			if err == nil {
				return conn, nil
			}

			// If DNS fails and we have a fallback IP, try that
			if fallbackIP, ok := dnsFallback[host]; ok {
				fallbackAddr := net.JoinHostPort(fallbackIP, port)
				conn, fallbackErr := dialer.DialContext(ctx, network, fallbackAddr)
				if fallbackErr == nil {
					// DNS fallback succeeded
					return conn, nil
				}
			}

			// Return original error if fallback also failed
			return nil, err
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// GetRegistryURL retrieves the configured registry URL from settings
func GetRegistryURL(db *sql.DB) (string, error) {
	// First check environment variable (highest priority)
	if envURL := os.Getenv("REGISTRY_URL"); envURL != "" {
		return envURL, nil
	}

	// Then check settings table
	var url string
	err := db.QueryRow("SELECT value FROM settings WHERE key = 'registry_url'").Scan(&url)
	if err == nil && url != "" {
		return url, nil
	}

	// Fallback to default production URL
	defaultURL := "https://clipilot.themobileprof.com"
	return defaultURL, nil
}

// SyncRegistry fetches the module list from registry and updates local cache
func (c *Client) SyncRegistry() error {
	// Update sync status
	_, err := c.db.Exec(`
		UPDATE registry_cache 
		SET sync_status = 'syncing', updated_at = strftime('%s', 'now')
		WHERE id = 1
	`)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Fetch module list from registry
	resp, err := c.httpClient.Get(c.registryURL + "/api/modules")
	if err != nil {
		c.updateSyncStatus("failed", err.Error(), 0)
		return fmt.Errorf("failed to fetch modules: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("registry returned status %d", resp.StatusCode)
		c.updateSyncStatus("failed", errMsg, 0)
		return fmt.Errorf("%s", errMsg)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.updateSyncStatus("failed", err.Error(), 0)
		return fmt.Errorf("failed to read response: %w", err)
	}

	var modules []ModuleMetadata
	if err := json.Unmarshal(body, &modules); err != nil {
		c.updateSyncStatus("failed", err.Error(), 0)
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Begin transaction
	tx, err := c.db.Begin()
	if err != nil {
		c.updateSyncStatus("failed", err.Error(), 0)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error as we might commit
	}()

	now := time.Now().Unix()
	cached := 0

	// Insert or update modules in cache
	for _, mod := range modules {
		moduleID := fmt.Sprintf("%s.%s.%s", "org.themobileprof", mod.Name, mod.Version)
		downloadURL := fmt.Sprintf("%s/api/modules/%d",
			c.registryURL, mod.ID)

		tagsStr := strings.Join(mod.Tags, ",")

		// Check if already installed
		var installed bool
		err := tx.QueryRow("SELECT installed FROM modules WHERE id = ?", moduleID).Scan(&installed)

		if err == sql.ErrNoRows {
			// New module - insert as cached (not installed)
			_, err = tx.Exec(`
				INSERT INTO modules (id, name, version, description, tags, registry_id, download_url, author, last_synced, installed)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
			`, moduleID, mod.Name, mod.Version, mod.Description, tagsStr, mod.ID, downloadURL, mod.Author, now)

			if err != nil {
				continue
			}
			cached++
		} else if err == nil && !installed {
			// Existing cached module - update metadata
			_, err = tx.Exec(`
				UPDATE modules 
				SET description = ?, tags = ?, registry_id = ?, author = ?, last_synced = ?
				WHERE id = ?
			`, mod.Description, tagsStr, mod.ID, mod.Author, now, moduleID)

			if err != nil {
				continue
			}
			cached++
		} else if err != nil {
			// Database error - skip this module
			continue
		}
		// else: module is already installed, don't update
		// If installed, don't overwrite local version
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.updateSyncStatus("failed", err.Error(), 0)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update sync status
	c.updateSyncStatus("success", "", len(modules))

	return nil
}

// updateSyncStatus updates the registry cache metadata
func (c *Client) updateSyncStatus(status, errorMsg string, totalModules int) {
	_, _ = c.db.Exec(`
		UPDATE registry_cache 
		SET sync_status = ?,
			sync_error = ?,
			total_modules = ?,
			last_sync = strftime('%s', 'now'),
			updated_at = strftime('%s', 'now')
		WHERE id = 1
	`, status, errorMsg, totalModules)
}

// GetSyncStatus returns the current registry sync status
func (c *Client) GetSyncStatus() (*SyncStatus, error) {
	var status SyncStatus
	var lastSync, totalModules, cachedModules int64
	var statusStr, errorStr string

	// Query basic cache info first
	err := c.db.QueryRow(`
		SELECT COALESCE(last_sync, 0), COALESCE(total_modules, 0), 
		       COALESCE(sync_status, 'never'), COALESCE(sync_error, '')
		FROM registry_cache WHERE id = 1
	`).Scan(&lastSync, &totalModules, &statusStr, &errorStr)

	if err != nil {
		// Registry cache not initialized yet
		return &SyncStatus{
			Status: "never",
		}, nil
	}

	// Try to count cached modules (may fail if schema not updated)
	_ = c.db.QueryRow(`
		SELECT COUNT(*) FROM modules WHERE installed = 0 AND registry_id IS NOT NULL
	`).Scan(&cachedModules)
	// Ignore error - column may not exist in old schema

	if lastSync > 0 {
		status.LastSync = time.Unix(lastSync, 0)
	}
	status.TotalModules = int(totalModules)
	status.CachedModules = int(cachedModules)
	status.Status = statusStr
	status.Error = errorStr

	return &status, nil
}

// ShouldAutoSync checks if auto-sync is enabled and due
func (c *Client) ShouldAutoSync() (bool, error) {
	var autoSync string
	var syncInterval int64
	var lastSync sql.NullInt64

	// Get settings
	err := c.db.QueryRow("SELECT value FROM settings WHERE key = 'auto_sync'").Scan(&autoSync)
	if err != nil || autoSync != "true" {
		return false, nil
	}

	err = c.db.QueryRow("SELECT value FROM settings WHERE key = 'sync_interval'").Scan(&syncInterval)
	if err != nil {
		syncInterval = 86400 // Default 24h
	}

	err = c.db.QueryRow("SELECT last_sync FROM registry_cache WHERE id = 1").Scan(&lastSync)
	if err != nil {
		return true, nil // Never synced
	}

	if !lastSync.Valid {
		return true, nil // Never synced
	}

	// Check if sync is due
	elapsed := time.Now().Unix() - lastSync.Int64
	return elapsed > syncInterval, nil
}

// DownloadModule downloads a module YAML from registry and installs it
func (c *Client) DownloadModule(moduleID string) (*models.Module, error) {
	// Get download URL from cache
	var downloadURL string
	var name, version string
	err := c.db.QueryRow(`
		SELECT name, version, download_url 
		FROM modules 
		WHERE id = ? AND registry_id IS NOT NULL
	`, moduleID).Scan(&name, &version, &downloadURL)

	if err != nil {
		return nil, fmt.Errorf("module not found in registry cache: %w", err)
	}

	// Download YAML
	resp, err := c.httpClient.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download module: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	// Parse YAML
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read module: %w", err)
	}

	// TODO: Parse and validate YAML then install
	// For now, return empty module
	var module models.Module
	// This would use yaml.Unmarshal and ImportModule in real implementation

	return &module, nil
}

// ListAvailableModules returns modules in cache but not installed
func (c *Client) ListAvailableModules() ([]ModuleMetadata, error) {
	rows, err := c.db.Query(`
		SELECT COALESCE(registry_id, 0), name, version, description, COALESCE(author, ''), tags
		FROM modules
		WHERE installed = 0 AND registry_id IS NOT NULL
		ORDER BY name, version
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []ModuleMetadata
	for rows.Next() {
		var mod ModuleMetadata
		var tagsStr string
		err := rows.Scan(&mod.ID, &mod.Name, &mod.Version, &mod.Description, &mod.Author, &tagsStr)
		if err != nil {
			continue
		}
		if tagsStr != "" {
			mod.Tags = strings.Split(tagsStr, ",")
		}
		modules = append(modules, mod)
	}

	return modules, nil
}

// SubmitModuleRequest sends a module request to the registry when no matching module is found
func (c *Client) SubmitModuleRequest(query string, userContext string) error {
	if c.registryURL == "" {
		return fmt.Errorf("registry URL not configured")
	}

	requestData := map[string]string{
		"query":        query,
		"user_context": userContext,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	url := fmt.Sprintf("%s/api/module-request", c.registryURL)
	resp, err := c.httpClient.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry returned error: %s", string(body))
	}

	var result struct {
		Success   bool   `json:"success"`
		Message   string `json:"message"`
		RequestID int64  `json:"request_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("request failed: %s", result.Message)
	}

	return nil
}

// ListInstalledModules returns locally installed modules
func (c *Client) ListInstalledModules() ([]ModuleMetadata, error) {
	rows, err := c.db.Query(`
		SELECT COALESCE(registry_id, 0), name, version, description, COALESCE(author, ''), tags
		FROM modules
		WHERE installed = 1
		ORDER BY name, version
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []ModuleMetadata
	for rows.Next() {
		var mod ModuleMetadata
		var tagsStr string
		err := rows.Scan(&mod.ID, &mod.Name, &mod.Version, &mod.Description, &mod.Author, &tagsStr)
		if err != nil {
			continue
		}
		if tagsStr != "" {
			mod.Tags = strings.Split(tagsStr, ",")
		}
		modules = append(modules, mod)
	}

	return modules, nil
}

// CommandCandidate represents a command result from the registry
type CommandCandidate struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	UseCases    []string `json:"use_cases"`
	Keywords    []string `json:"keywords"`
}

// SearchResponse represents the registry search response
type SearchResponse struct {
	Candidates []CommandCandidate `json:"candidates"`
	Message    string             `json:"message"`
	Cached     bool               `json:"cached"`
}

// SearchCommands performs a semantic search on the registry
func (c *Client) SearchCommands(query string) ([]CommandCandidate, error) {
	reqBody := map[string]string{"query": query}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := c.httpClient.Post(c.registryURL+"/api/commands/search", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Candidates, nil
}
