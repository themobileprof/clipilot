# Clio Migration Guide - Client-Server Architecture Split

This document provides comprehensive guidance for migrating Clio from a GitHub-based module sync to a CLIPilot registry-based architecture.

## Table of Contents
1. [Overview](#overview)
2. [Architecture Changes](#architecture-changes)
3. [Components to Remove](#components-to-remove)
4. [Components to Replace](#components-to-replace)
5. [New Registry Client Implementation](#new-registry-client-implementation)
6. [Installation Flow Changes](#installation-flow-changes)
7. [CI/CD Integration](#ci-cd-integration)
8. [Configuration Changes](#configuration-changes)
9. [Testing Checklist](#testing-checklist)
10. [Migration Timeline](#migration-timeline)

---

## Overview

### Why This Split?

**Before:** Clio fetched modules directly from GitHub's CLIPilot repository using GitHub API.

**After:** Clio fetches modules from CLIPilot's dedicated registry server at `clipilot.themobileprof.com`.

**Benefits:**
- **Curated modules**: Registry validates and approves modules before listing
- **Better search**: Server-side semantic search with Gemini AI
- **Version management**: Proper module versioning and dependency resolution
- **Analytics**: Track module downloads and popularity
- **Delta sync**: Only download changed modules, reducing bandwidth
- **ETag caching**: HTTP caching reduces repeated downloads

### Separation of Concerns

| Component | Hosted Where | Responsibility |
|-----------|-------------|----------------|
| **Clio CLI Binary** | GitHub Releases | Client application, installed on user systems |
| **Install Script** | CLIPilot Server (`/clio`) | Downloads Clio binary from GitHub, handles installation |
| **Module YAMLs** | CLIPilot Registry | Workflow modules, synced to client |
| **Web UI** | CLIPilot Server | Module browsing, upload, management |

---

## Architecture Changes

### Current Clio Architecture (Before)

```
┌─────────────────┐
│  Clio Client    │
│  (Go binary)    │
└────────┬────────┘
         │
         │ GitHub API
         │ /repos/themobileprof/clipilot/contents/modules
         ↓
┌─────────────────┐
│  GitHub Repo    │
│  clipilot       │
│  /modules/*.yaml│
└─────────────────┘
```

### New Architecture (After)

```
┌─────────────────┐
│  Clio Client    │
│  (Go binary)    │
└────────┬────────┘
         │
         │ HTTPS REST API
         │ /api/v1/modules
         ↓
┌──────────────────────┐
│  CLIPilot Server     │
│  clipilot.themobileprof.com │
│  - Module Registry   │
│  - Web UI            │
│  - Install Script    │
└──────────────────────┘
```

---

## Components to Remove

### 1. Delete `install.sh` from Clio Repo

**Location:** `/home/samuel/sites/clio/install.sh`

**Reason:** CLIPilot server now hosts the install script at `clipilot.themobileprof.com/clio`. Clio's CI/CD uploads it there.

**Action:**
```bash
cd /path/to/clio
git rm install.sh
git commit -m "Remove install.sh - now hosted by CLIPilot server"
```

**Update README.md installation instructions to:**
```bash
curl -fsSL clipilot.themobileprof.com/clio | sh
```

---

### 2. Remove GitHub API Integration from `sync.go`

**File:** `internal/modules/sync.go`

**Delete these constants:**
```go
const (
	RepoOwner = "themobileprof"
	RepoName  = "clipilot"
	ModulesPath = "modules"
	GitHubAPI = "https://api.github.com/repos"
)
```

**Delete this struct:**
```go
type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}
```

**Delete the entire `Sync()` function** - it will be replaced with a new implementation.

---

### 3. Remove Redundant Module Validation

If Clio has any YAML schema validation logic for modules, it can be simplified. The registry server now handles validation on upload.

**Keep:** Basic YAML parsing to execute modules  
**Remove:** Complex schema validation (server does this)

---

## Components to Replace

### Replace `Sync()` Function in `sync.go`

**Current implementation** (to be deleted):
```go
func Sync() error {
	// Fetches from GitHub API
	url := fmt.Sprintf("%s/%s/%s/contents/%s", GitHubAPI, RepoOwner, RepoName, ModulesPath)
	resp, err := http.Get(url)
	// ... downloads each YAML file
}
```

**New implementation** (use registry API):
```go
package modules

import (
	"clio/internal/layer3"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Registry API base URL (configurable via config)
	DefaultRegistryURL = "https://clipilot.themobileprof.com"
)

type RegistryModule struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Tags           []string `json:"tags"`
	DownloadCount  int      `json:"download_count"`
	UploadedAt     string   `json:"uploaded_at"`
	UpdatedAt      string   `json:"updated_at"`
	ChecksumSHA256 string   `json:"checksum_sha256"`
}

type ChangedModulesResponse struct {
	ChangedModules []struct {
		ID             string `json:"id"`
		Version        string `json:"version"`
		ChecksumSHA256 string `json:"checksum_sha256"`
		UpdatedAt      string `json:"updated_at"`
		ChangeType     string `json:"change_type"`
	} `json:"changed_modules"`
	SyncTimestamp string `json:"sync_timestamp"`
}

// Sync downloads modules from CLIPilot registry using delta sync
func Sync() error {
	registryURL := getRegistryURL() // Read from config, default to DefaultRegistryURL
	
	fmt.Println("🔄 Syncing modules from registry...")
	
	// Get last sync timestamp
	lastSync, err := layer3.GetLastSyncTimestamp()
	if err != nil {
		lastSync = time.Time{} // First sync, get all modules
	}
	
	// Call delta sync endpoint
	url := fmt.Sprintf("%s/api/v1/modules/changed?since=%s", 
		registryURL, lastSync.Format(time.RFC3339))
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to registry: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned status %d", resp.StatusCode)
	}
	
	var changedResp ChangedModulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&changedResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	count := 0
	for _, mod := range changedResp.ChangedModules {
		// Check if we need to download (checksum differs)
		localChecksum, err := layer3.GetModuleChecksum(mod.ID)
		if err == nil && localChecksum == mod.ChecksumSHA256 {
			// Already up to date
			continue
		}
		
		fmt.Printf("  Downloading %s...\n", mod.ID)
		if err := downloadAndSaveModule(registryURL, mod.ID); err != nil {
			fmt.Printf("  ❌ Failed %s: %v\n", mod.ID, err)
		} else {
			count++
		}
		
		// Be nice to server
		time.Sleep(100 * time.Millisecond)
	}
	
	// Save sync timestamp
	if err := layer3.SaveLastSyncTimestamp(time.Now()); err != nil {
		fmt.Printf("Warning: failed to save sync timestamp: %v\n", err)
	}
	
	fmt.Printf("✅ Sync complete. Updated %d modules.\n", count)
	return nil
}

// downloadAndSaveModule fetches a module YAML and saves to local DB
func downloadAndSaveModule(registryURL, moduleID string) error {
	url := fmt.Sprintf("%s/api/v1/modules/%s/download", registryURL, moduleID)
	
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	
	// Calculate checksum
	hash := sha256.Sum256(body)
	checksum := fmt.Sprintf("%x", hash)
	
	// Parse YAML for metadata
	var mod ModuleYAML
	if err := yaml.Unmarshal(body, &mod); err != nil {
		return fmt.Errorf("yaml parse error: %w", err)
	}
	
	// Validate
	if mod.ID == "" || mod.Name == "" {
		return fmt.Errorf("missing id or name")
	}
	
	tags := strings.Join(mod.Tags, ",")
	
	// Save to DB
	return layer3.UpsertModule(mod.ID, mod.Name, mod.Description, tags, mod.Version, string(body), checksum)
}

// getRegistryURL reads registry URL from config or returns default
func getRegistryURL() string {
	// TODO: Read from ~/.clio/config.yaml
	// For now, return default
	return DefaultRegistryURL
}
```

---

### Update `layer3/db.go` for Checksum and Timestamp Tracking

**Add to `layer3/db.go`:**

```go
// Add checksum column to modules table
func initSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS modules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		module_id TEXT UNIQUE,
		name TEXT NOT NULL,
		description TEXT,
		tags TEXT,
		version TEXT,
		content TEXT,
		checksum TEXT, -- SHA256 checksum for cache validation
		synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS sync_metadata (
		id INTEGER PRIMARY KEY CHECK (id = 1), -- Only one row
		last_sync_timestamp TIMESTAMP
	);
	`
	_, err := db.Exec(query)
	return err
}

// GetModuleChecksum returns the stored checksum for a module
func GetModuleChecksum(moduleID string) (string, error) {
	db, err := GetDB()
	if err != nil {
		return "", err
	}
	
	var checksum string
	err = db.QueryRow("SELECT checksum FROM modules WHERE module_id = ?", moduleID).Scan(&checksum)
	return checksum, err
}

// UpdateModuleWithChecksum updates module and stores checksum
func UpdateModuleWithChecksum(modID, name, desc, tags, version, content, checksum string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	
	_, err = db.Exec(`
		INSERT INTO modules (module_id, name, description, tags, version, content, checksum, synced_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(module_id) DO UPDATE SET
			name=excluded.name,
			description=excluded.description,
			tags=excluded.tags,
			version=excluded.version,
			content=excluded.content,
			checksum=excluded.checksum,
			synced_at=CURRENT_TIMESTAMP
	`, modID, name, desc, tags, version, content, checksum)
	
	return err
}

// GetLastSyncTimestamp returns when modules were last synced
func GetLastSyncTimestamp() (time.Time, error) {
	db, err := GetDB()
	if err != nil {
		return time.Time{}, err
	}
	
	var timestamp time.Time
	err = db.QueryRow("SELECT last_sync_timestamp FROM sync_metadata WHERE id = 1").Scan(&timestamp)
	if err == sql.ErrNoRows {
		return time.Time{}, nil // Never synced before
	}
	return timestamp, err
}

// SaveLastSyncTimestamp updates the last sync time
func SaveLastSyncTimestamp(t time.Time) error {
	db, err := GetDB()
	if err != nil {
		return err
	}
	
	_, err = db.Exec(`
		INSERT INTO sync_metadata (id, last_sync_timestamp) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET last_sync_timestamp=excluded.last_sync_timestamp
	`, t)
	
	return err
}
```

---

## Installation Flow Changes

### Current Flow
1. User downloads `install.sh` from Clio's GitHub repo
2. Script downloads `clio` binary from GitHub releases
3. Script installs binary

###New Flow
1. User runs: `curl -fsSL clipilot.themobileprof.com/clio | sh`
2. CLIPilot server returns `install.sh` (uploaded by Clio's CI/CD)
3. Script downloads `clio` binary from Clio's GitHub releases (unchanged)
4. Script installs binary

**Key Point:** Only the script hosting changes. Binary distribution stays on GitHub.

---

## CI/CD Integration

### Update `.github/workflows/release.yml`

Add a step to upload the install script to CLIPilot server after building binaries:

```yaml
name: Release Binaries

on:
  release:
    types: [published]

permissions:
  contents: write

jobs:
  # ... existing build jobs ...
  
  upload-install-script:
    name: Upload Install Script to CLIPilot
    runs-on: ubuntu-latest
    needs: releases-matrix # Run after binaries are uploaded
    steps:
      - uses: actions/checkout@v4
      
      - name: Upload install script to CLIPilot server
        run: |
          curl -X POST https://clipilot.themobileprof.com/api/install-script/upload \
            -H "Authorization: Bearer ${{ secrets.CLIPILOT_API_KEY }}" \
            -F "file=@install.sh" \
            -F "version=${{ github.ref_name }}"
        continue-on-error: true # Don't fail release if upload fails
```

**Required GitHub Secret:**
- `CLIPILOT_API_KEY` - Generate from CLIPilot admin dashboard (requires admin role)

---

## Configuration Changes

### Add `~/.clio/config.yaml`

Clio should support a configuration file for registry URL:

```yaml
# ~/.clio/config.yaml
registry_url: https://clipilot.themobileprof.com
cache_ttl: 24h # How long to cache module list
sync_interval: 168h # Auto-sync every 7 days (optional)
```

### Implementation

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"
	
	"gopkg.in/yaml.v3"
)

type Config struct {
	RegistryURL    string `yaml:"registry_url"`
	CacheTTL       string `yaml:"cache_ttl"`
	SyncInterval   string `yaml:"sync_interval"`
}

var defaultConfig = Config{
	RegistryURL:    "https://clipilot.themobileprof.com",
	CacheTTL:       "24h",
	SyncInterval:   "168h",
}

// Load reads config from ~/.clio/config.yaml or returns defaults
func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultConfig, nil
	}
	
	configPath := filepath.Join(home, ".clio", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist, return defaults
		return defaultConfig, nil
	}
	
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig, err
	}
	
	// Fill in missing values with defaults
	if cfg.RegistryURL == "" {
		cfg.RegistryURL = defaultConfig.RegistryURL
	}
	
	return cfg, nil
}
```

---

## Testing Checklist

### Local Testing

- [ ] **Module Sync**: Run `clio sync` - verify downloads modules from registry
- [ ] **Delta Sync**: Run sync twice - second run should be faster (only changed modules)
- [ ] **ETag Caching**: Check that unchanged modules return HTTP 304
- [ ] **Offline Mode**: Disconnect network, run `clio` - verify cached modules execute
- [ ] **Install Script**: Test `curl -fsSL clipilot.themobileprof.com/clio | sh` (when deployed)
- [ ] **First Run**: Delete `~/.clio/`, run `clio` - verify initializes properly

### Integration Testing

- [ ] **API Connectivity**: Verify Clio can reach `clipilot.themobileprof.com`
- [ ] **Module Download**: Test `/api/v1/modules/:id/download` endpoint
- [ ] **Changed Endpoint**: Test `/api/v1/modules/changed?since=<timestamp>`
- [ ] **Error Handling**: What happens if registry is down? (Should use cached modules)
- [ ] **Rate Limiting**: Test that Clio respects HTTP 429 responses

### CI/CD Testing

- [ ] **Install Script Upload**: Tag a release, verify script uploads to CLIPilot
- [ ] **Binary Integrity**: Download binary via install script, verify checksums
- [ ] **Version Matching**: Ensure install script downloads correct binary version

---

## Migration Timeline

### Phase 1: Preparation (Non-Breaking)

**Goal:** Add registry support alongside GitHub sync

**Tasks:**
1. Implement new `SyncFromRegistry()` function (keep old `Sync()` as fallback)
2. Add config file support for `registry_url`
3. Update database schema with checksum and sync_metadata tables
4. Add error handling to gracefully fall back to GitHub if registry fails

**Command:**
```go
func Sync() error {
    // Try registry first
    if err := SyncFromRegistry(); err != nil {
        log.Printf("Registry sync failed: %v, falling back to GitHub", err)
        return SyncFromGitHub() // Old implementation
    }
    return nil
}
```

**Deploy:** Can deploy immediately, won't break existing users

---

### Phase 2: Transition (Default to Registry)

**Goal:** Make registry the default, keep GitHub as fallback

**Tasks:**
1. Update README to mention registry-first approach
2. Set `registry_url` in default config
3. Monitor error rates for registry failures
4. Add user notification: "Registry sync complete. Use --github flag to sync from GitHub instead."

**timeline:** After Phase 1 has been stable for 1-2 weeks

---

### Phase 3: Full Migration (Remove GitHub Sync)

**Goal:** Remove GitHub API dependency completely

**Tasks:**
1. Delete `SyncFromGitHub()` function and GitHub API constants
2. Remove `--github` fallback flag
3. Update all documentation to only mention registry
4. Delete `install.sh` from Clio repo (now hosted by CLIPilot)

**Timeline:** After Phase 2 has been stable for 1 month

---

## Summary

**What Changes:**
- Module sync source: GitHub API → CLIPilot registry
- Install script hosting: Clio repo → CLIPilot server
- Binary hosting: **No change** (stays on GitHub releases)

**What Clio Gains:**
- Faster sync (delta updates only)
- Better caching (HTTP ETags)
- Semantic search (registry server)
- Module analytics (download counts)
- Dependency resolution (planned)

**Implementation Effort:**
- **Low**: Database schema updates, config file
- **Medium**: Replace `Sync()` function with registry client
- **Low**: CI/CD update to upload install script

**Risk Mitigation:**
- Phase 1 keeps GitHub as fallback
- Registry is lightweight (SQLite, no external dependencies)
- Clio works 100% offline after initial sync

---

## Support & Questions

**Registry API Documentation:**  
See `docs/CLIO_API_REQUIREMENTS.md` in CLIPilot repo for full endpoint specs.

**Testing the Registry:**
```bash
# List modules
curl https://clipilot.themobileprof.com/api/v1/modules?limit=5

# Get module metadata
curl https://clipilot.themobileprof.com/api/v1/modules/copy_file

# Download module
curl https://clipilot.themobileprof.com/api/v1/modules/copy_file/download

# Check changed modules
curl "https://clipilot.themobileprof.com/api/v1/modules/changed?since=2026-01-01T00:00:00Z"
```

**Contact:** Open an issue in either CLIPilot or Clio repos for migration support.
