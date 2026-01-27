# Client-Server Architecture Flow

This document maps the end-to-end flow of a user command in dynamic CLIPilot architecture.

## 🔄 High Level Overview

```mermaid
graph TD
    User([User Input]) --> Client[Local Client (CLI)]
    Client -->|1. Detect Intent| Intent{Local Intent?}
    
    Intent -->|Yes| LocalExec[Local Execution]
    LocalExec -->|Run| System[System Shell]
    
    Intent -->|No| Cloud{Offline?}
    Cloud -->|Yes| Fallback[Keyword/Man Page Search]
    
    Cloud -->|No| Server[Registry Server]
    Server -->|2. Semantic Search| Cache{Cached?}
    
    Cache -->|Yes| ServerResp[Server Response]
    Cache -->|No| LLM(Gemini LLM)
    
    LLM -->|3. Generate Candidates| Cache
    ServerResp -->|JSON| Client
    
    Client -->|4. Confirmation| User
    User -->|Approve| LocalExec
```

## 🔍 Detailed Code Path

### 1. Client Side (The Terminal)

**Entry Point**: `cmd/clipilot/main.go`
- Initializes `repl.New()`
- Starts the interactive loop.

**Step 1: Input Handling**
- File: `internal/ui/repl.go`
- Function: `handleInput(text)`
- Action: Passes user text to `IntentDetector`.

**Step 2: Intent Detection**
- File: `internal/intent/keyword.go` (and `hybrid.go`)
- Function: `Detect(input)`
- Logic:
    1.  **Exact Match**: Checks `internal/commands/catalog.go` (In-Memory).
    2.  **Fuzzy Match**: Checks TF-IDF index of local man pages.
    3.  **Registry Fallback**: If confidence < threshold, calls `registry.SearchCommands`.

**Step 3: Registry Client**
- File: `internal/registry/client.go`
- Function: `SearchCommands(query)`
- Action: Sends `POST /api/commands/search` to `clipilot.themobileprof.com`.
- Safety: Uses `newTransportWithDNSFallback()` to bypass Termux DNS issues.

### 2. Server Side (The Registry)

**Entry Point**: `cmd/registry/main.go`
- Starts HTTP server on port 8080.
- wrapper `handlers.New()`.

**Step 4: API Handling**
- File: `server/handlers/commands.go` (Logic) & `handlers.go` (Routing)
- Function: `HandleSemanticSearch` -> `searchWithGemini`
- Logic:
    1.  **Cache Check**: Hashes the query (SHA-256) and checks `query_cache` (SQLite).
    2.  **LLM Call**: If cache miss, calls **Gemini Flash API** (`searchWithGemini`) to generate candidates.
    3.  **Response**: Results are cached for 24h and returned as `CommandCandidate` objects.

### 3. Execution (Back to Client)

**Step 5: User Selection**
- `REPL` displays the server suggestions.
- User selects an option (e.g., `#1`).

**Step 6: Safe Execution**
- File: `internal/engine/runner.go`
- Function: `Run(command)`
- Safety: Uses `internal/utils/safeexec` to resolve paths without crashing (`safeexec.LookPath`).
- Action: Executes `bash -c <command>` and streams output.

## 🛠️ Key Components via File

| Component | File Path | Purpose |
|-----------|-----------|---------|
| **REPL** | `internal/ui/repl.go` | Main UI loop, user input reading. |
| **Intent** | `internal/intent/keyword.go` | Brain. Decides *what* to do. |
| **Catalog** | `internal/commands/catalog.go` | fast local database of 50 common commands. |
| **Safety** | `internal/utils/safeexec/safeexec.go` | Prevents SIGSYS crashes on Android. |
| **Client API** | `internal/registry/client.go` | Talks to the server. |
| **Server API** | `server/handlers/handlers.go` | Handles requests, talks to DB. |
