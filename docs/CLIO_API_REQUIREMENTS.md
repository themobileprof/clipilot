# Clio Client - API Requirements for CLIPilot Server

This document outlines all API endpoints that Clio needs from CLIPilot server to function properly.

## Current State Analysis

### What Clio Currently Uses

**Module Sync (GitHub API - NEEDS REPLACEMENT):**
```go
// Current: internal/modules/sync.go
GET https://api.github.com/repos/themobileprof/clipilot/contents/modules
GET {download_url} // For each YAML file
```

**Remote Search (Already Exists on CLIPilot):**
```go
// Current: internal/layer4/client.go
POST https://clipilot.themobileprof.com/api/commands/search
Request: {"query": "...", "os": "linux", "arch": "arm64"}
Response: {"results": [{"name": "...", "description": "...", "usage": "..."}]}
```

---

## Required API Endpoints

### 1. Module Discovery & Sync

#### `GET /api/v1/modules`
List all available modules with optional filtering.

**Request:**
```bash
GET /api/v1/modules?tags=git,termux&updated_since=2026-01-01T00:00:00Z&limit=50&offset=0
```

**Query Parameters:**
- `tags` (optional): Comma-separated list of tags to filter by
- `updated_since` (optional): ISO8601 timestamp, return only modules updated after this time
- `limit` (optional): Max results per page (default: 50, max: 100)
- `offset` (optional): Pagination offset (default: 0)
- `platform` (optional): Filter by platform (termux, linux, macos)
- `search` (optional): Search query for name/description
- `sort_by` (optional): Sort field (downloads, name, updated_at) (default: name)
- `order` (optional): Sort order (asc, desc) (default: asc)

**Response:**
```json
{
  "modules": [
    {
      "id": "org.themobileprof.copy_file",
      "name": "copy_file",
      "description": "Copy files with validation",
      "version": "1.0.0",
      "tags": ["file-ops", "atomic"],
      "requires": ["check_file_exists"],
      "provides": ["file_copied"],
      "size_kb": 2.5,
      "download_count": 1234,
      "checksum_sha256": "abc123...",
      "uploaded_by": "user123",
      "uploaded_at": "2026-01-15T10:00:00Z",
      "updated_at": "2026-01-20T15:30:00Z"
    }
  ],
  "total": 150,
  "limit": 50,
  "offset": 0
}
```

**Headers:**
- `ETag`: Module list version hash
- `Last-Modified`: Timestamp of most recent module update

---

#### `GET /api/v1/modules/:id`
Get metadata for a specific module.

**Request:**
```bash
GET /api/v1/modules/org.themobileprof.copy_file
```

**Response:**
```json
{
  "id": "org.themobileprof.copy_file",
  "name": "copy_file",
  "description": "Copy files with validation",
  "version": "1.0.0",
  "tags": ["file-ops", "atomic"],
  "requires": ["check_file_exists", "check_disk_space"],
  "provides": ["file_copied"],
  "size_kb": 2.5,
  "download_count": 1234,
  "checksum_sha256": "abc123...",
  "uploaded_by": "user123",
  "uploaded_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-20T15:30:00Z"
}
```

**Headers:**
- `ETag`: Module content hash
- `Last-Modified`: Module update timestamp

---

#### `GET /api/v1/modules/:id/download`
Download the raw YAML content for a module.

**Request:**
```bash
GET /api/v1/modules/org.themobileprof.copy_file/download
```

**Response:**
```yaml
name: copy_file
id: org.themobileprof.copy_file
version: 1.0.0
description: Copy files with validation
tags: [file-ops, atomic]
requires: [check_file_exists]
provides: [file_copied]

flows:
  main:
    start: validate_source
    steps:
      # ... full YAML content
```

**Headers:**
- `Content-Type`: application/x-yaml
- `Content-Disposition`: attachment; filename="copy_file.yaml"
- `ETag`: Content hash
- `Last-Modified`: Update timestamp

**Side Effects:**
- Increments download counter in background

---

#### `GET /api/v1/modules/changed`
Delta sync - get only modules that have changed since last sync.

**Request:**
```bash
GET /api/v1/modules/changed?since=2026-02-01T00:00:00Z
```

**Query Parameters:**
- `since` (required): ISO8601 timestamp, return modules updated after this time

**Response:**
```json
{
  "changed_modules": [
    {
      "id": "org.themobileprof.copy_file",
      "version": "1.0.1",
      "checksum_sha256": "def456...",
      "updated_at": "2026-02-15T10:00:00Z",
      "change_type": "updated"
    },
    {
      "id": "org.themobileprof.new_module",
      "version": "1.0.0",
      "checksum_sha256": "ghi789...",
      "updated_at": "2026-02-10T14:30:00Z",
      "change_type": "added"
    }
  ],
  "sync_timestamp": "2026-02-28T12:00:00Z"
}
```

**Usage Pattern:**
1. Clio stores last sync timestamp in local DB
2. On sync command, calls this endpoint with last timestamp
3. For each changed module, downloads full YAML if checksum differs
4. Updates local DB and saves new sync timestamp

---

#### `GET /api/v1/modules/:id/dependencies`
Get recursive dependency tree for a module.

**Request:**
```bash
GET /api/v1/modules/org.themobileprof.copy_file/dependencies
```

**Response:**
```json
{
  "module_id": "org.themobileprof.copy_file",
  "dependencies": [
    {
      "id": "org.themobileprof.check_file_exists",
      "required_by": "org.themobileprof.copy_file",
      "depth": 1
    },
    {
      "id": "org.themobileprof.check_disk_space",
      "required_by": "org.themobileprof.copy_file",
      "depth": 1
    },
    {
      "id": "org.themobileprof.check_path_exists",
      "required_by": "org.themobileprof.check_file_exists",
      "depth": 2
    }
  ],
  "install_order": [
    "org.themobileprof.check_path_exists",
    "org.themobileprof.check_file_exists",
    "org.themobileprof.check_disk_space",
    "org.themobileprof.copy_file"
  ]
}
```

---

### 2. Remote Search (Already Exists)

#### `POST /api/commands/search`
Semantic search for commands (fallback when local intent detection fails).

**Request:**
```bash
POST /api/commands/search
Content-Type: application/json

{
  "query": "copy a file",
  "os": "linux",
  "arch": "arm64"
}
```

**Response:**
```json
{
  "results": [
    {
      "name": "cp",
      "description": "Copy files and directories",
      "usage": "cp source dest"
    },
    {
      "name": "rsync",
      "description": "Remote file synchronization",
      "usage": "rsync -av source dest"
    }
  ]
}
```

**Note:** This endpoint already exists and works. No changes needed.

---

### 3. Health Check

#### `GET /health`
Server health status.

**Request:**
```bash
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "database": "connected",
  "disk_free_gb": 145.2,
  "timestamp": "2026-02-28T12:00:00Z"
}
```

**Usage:** Clio can check server availability before attempting sync.

---

## Caching Strategy

### ETag Support
Clio should implement ETag caching to reduce bandwidth:

```go
// Store ETag with module in local DB
type CachedModule struct {
    ID      string
    Content string
    ETag    string
    CachedAt time.Time
}

// When downloading, send If-None-Match header
req.Header.Set("If-None-Match", cachedModule.ETag)

// If server returns 304 Not Modified, use cached content
if resp.StatusCode == http.StatusNotModified {
    return cachedModule.Content, nil
}
```

### Delta Sync Optimization
Instead of downloading all modules on every sync:

```go
func SyncFromRegistry() error {
    // Get last sync timestamp
    lastSync := getLastSyncTimestamp()
    
    // Query only changed modules
    changed, err := getChangedModules(lastSync)
    if err != nil {
        return err
    }
    
    // Download only changed modules
    for _, mod := range changed {
        if mod.Checksum != getLocalChecksum(mod.ID) {
            content, err := downloadModule(mod.ID)
            if err != nil {
                log.Printf("Failed to download %s: %v", mod.ID, err)
                continue
            }
            updateLocalModule(mod.ID, content, mod.Checksum)
        }
    }
    
    // Update last sync timestamp
    saveLastSyncTimestamp(time.Now())
    return nil
}
```

---

## Rate Limiting

**Public Endpoints (no auth):**
- 60 requests per minute per IP address
- 429 Too Many Requests returned when exceeded
- `Retry-After` header indicates when to retry

**Authenticated Endpoints (API keys):**
- 1000 requests per hour per API key
- Configurable per key in admin dashboard

---

## Error Responses

All API endpoints follow consistent error format:

```json
{
  "error": {
    "code": "MODULE_NOT_FOUND",
    "message": "Module 'invalid_id' does not exist",
    "details": "Check available modules at /api/v1/modules"
  }
}
```

**Common Error Codes:**
- `MODULE_NOT_FOUND` (404): Requested module doesn't exist
- `INVALID_PARAMS` (400): Invalid query parameters
- `RATE_LIMIT_EXCEEDED` (429): Too many requests
- `SERVER_ERROR` (500): Internal server error

---

## Migration Path for Clio

### Phase 1: Implement Registry Client (Alongside GitHub)
```go
// internal/modules/sync.go - Add new function
func SyncFromRegistry() error {
    // Call /api/v1/modules/changed
    // Download changed modules
    // Update local DB
}

// Keep old Sync() as fallback
func Sync() error {
    err := SyncFromRegistry()
    if err != nil {
        log.Printf("Registry sync failed, falling back to GitHub: %v", err)
        return syncFromGitHub() // Old implementation
    }
    return nil
}
```

### Phase 2: Default to Registry
```go
// Add config: ~/.clio/config.yaml
registry_url: https://clipilot.themobileprof.com
fallback_to_github: true
```

### Phase 3: Remove GitHub Fallback
- Delete `syncFromGitHub()` function
- Remove GitHub API constants
- Update documentation

---

## Summary

**Endpoints Clio Needs (New):**
1. `GET /api/v1/modules` - List modules
2. `GET /api/v1/modules/:id` - Get module metadata
3. `GET /api/v1/modules/:id/download` - Download YAML
4. `GET /api/v1/modules/changed` - Delta sync
5. `GET /api/v1/modules/:id/dependencies` - Dependency resolution
6. `GET /health` - Health check

**Endpoints Already Working:**
1. `POST /api/commands/search` - Remote semantic search

**Key Features:**
- ETag caching to reduce bandwidth
- Delta sync to download only changes
- Pagination for large module lists
- Filtering by tags, platform, search query
- Dependency resolution for installation planning
