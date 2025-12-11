# Mobile CLI Assistant — Implementation Plan

**Purpose:** A detailed engineering plan to build a lightweight, offline-first, modular CLI assistant for mobile Linux (Termux, other Android terminals). Designed for low-end phones (2–4GB RAM), offline-first behavior, modular content via downloadable "modules", local SQLite cache, hybrid intent detection (keyword search → tiny local LLM → online LLM fallback), and deterministic flow engine for multi-step tasks with branching and module chaining.

**Audience:** Engineers using VSCode with Copilot and Claude Sonnet. This doc provides architecture, schemas, file layout, example modules, Go structs/pseudocode, testing strategies, CI, prompts for Copilot/Claude, and milestones.

---

## Table of Contents
1. Goals & non-goals
2. High-level architecture
3. Intent detection pipeline
4. Module format (YAML) & examples
5. SQLite schema (DDL) and rationale
6. Flow engine: state machine & pseudocode
7. Go project layout & key structs
8. Tiny LLM integration (intent classifier)
9. Online LLM fallback & API considerations
10. UX / REPL design & commands
11. Module repository, packaging, versioning, updates
12. Performance, memory, & storage optimization
13. Security, privacy, and safety
14. Testing plan & QA
15. CI/CD, builds, releases
16. Metrics & telemetry (privacy-first)
17. Development milestones & deliverables
18. VSCode + Copilot/Claude prompts and workflows
19. Appendix: examples, queries, and snippets

---

## 1. Goals & non-goals
**Goals**
- Small core binary (<20MB), lightweight runtime.
- Offline-first intent detection and deterministic multi-step flows.
- Modular downloadable content (modules) stored in SQLite.
- Hybrid intent pipeline: keyword DB search ➜ tiny local LLM classifier (1–10MB) ➜ online LLM fallback.
- Robust flow engine supporting branching, conditions, sub-flows, and module chaining.
- Safe: deterministic command sequences for operations; LLMs for classification/explanation only.

**Non-goals**
- Running full LLM reasoning locally.
- Replacing full desktop UX; avoid destructive automated execution without explicit user confirmation.

---

## 2. High-level architecture

```
[User REPL / CLI] <---> [Core Engine (Go)]
                             |
              +--------------+----------------+
              |                               |
        [SQLite Local Cache]             [Tiny Local LLM]
              |                               |
         [Downloaded Modules]        (intent classifier)
                                              |
                                       [Online LLM API]
```

**Components**
- **Core binary (Go):** REPL, module manager, flow runner, DB layer, tiny LLM wrapper, sync client.
- **SQLite DB:** Stores modules metadata, normalized flows, patterns, state, logs.
- **Module files (YAML/JSON):** Packaged modules delivered by server; imported into SQLite.
- **Tiny LLM:** Local intent classifier; label-only output.
- **Online LLM:** Explanations and ambiguous intent resolution.
- **Module registry server (optional):** hosts module packages and metadata for on-demand download.

---

## 3. Intent detection pipeline
**Order of operations** (must be fast and deterministic)
1. **Keyword/DB Search (layer 1)**
   - Tokenize input, lowercase, remove punctuation.
   - Run weighted keyword matching against `intent_patterns` table.
   - Score candidates by match count, tag weights, recency/popularity.
   - If top candidate score >= `THRESH_DB` (configurable, e.g., 0.6) → select module.

2. **Tiny Local LLM (layer 2)**
   - Model receives normalized text; returns `label` + `confidence`.
   - If confidence >= `THRESH_LLM` (e.g., 0.6) → select label (module).

3. **Online LLM (layer 3 fallback)**
   - Triggered only if both layer 1 & 2 fail or user explicitly asks to "search online".
   - Returns label and optionally a short rationale.

4. **Menu fallback**
   - When online is not available and both offline layers fail, present short menu of high-prob modules.

**Confidence thresholds**
- `THRESH_DB` = 0.6 default (depends on scoring function)
- `THRESH_LLM` = 0.6 default
- Tune via telemetry/AB tests.

**Scoring algorithm (DB search)**
- For each candidate module, compute: `score = sum(w_i * match_i) / sum(w_i)`
- `w_i` = weight of token (e.g., command token > general token)
- Boost by tags matching (e.g., distro names), user prefs, module popularity

---

## 4. Module format (YAML) & examples
**High-level design decisions**
- Modules are small, focused on a single topic or sub-task
- Modules declare `provides`, `requires`, `tags`, and `flows`
- Flows are state machines: steps, conditions, branches, actions
- Module packaging: `module-name-v1.0.tmod` (gzipped JSON/YAML)

**Top-level YAML schema (example)**

```yaml
name: install_mysql
id: org.themobileprof.install_mysql
version: 1.0.0
description: Install and configure MySQL (with secure defaults)
tags: [mysql, database, install]
provides:
  - mysql_installed
requires:
  - detect_os
size_kb: 12
flows:
  main:
    start: detect_os
    steps:
      detect_os:
        type: action
        run_module: detect_os
      install_branch:
        type: branch
        based_on: os_type
        map:
          ubuntu: install_ubuntu
          termux: install_termux
      install_ubuntu:
        type: instruction
        message: "Run: sudo apt-get update && sudo apt-get install -y mysql-server"
        command: "sudo apt-get update && sudo apt-get install -y mysql-server"
        validate:
          - check_command: "mysql --version"
      secure_mysql:
        type: instruction
        message: "Run: sudo mysql_secure_installation"
        command: "sudo mysql_secure_installation"
        next: done
  done:
    type: terminal
    message: "MySQL installation flow complete"

metadata:
  author: TheMobileProf
  license: mit
```

**Tiny module examples**
- `detect_os` - outputs `os_type` (termux/ubuntu/debian/fedora/alpine)
- `install_mysql_ubuntu` - commands for Ubuntu
- `secure_mysql` - common security steps

**Module size target**: keep modules under 50KB typically; larger modules split into submodules.

---

## 5. SQLite schema (DDL) and rationale
**Goals:** fast lookups, small footprint, simple updates, ACID flow state

```sql
-- modules metadata
CREATE TABLE modules (
  id TEXT PRIMARY KEY,
  name TEXT,
  version TEXT,
  description TEXT,
  tags TEXT, -- comma-separated for simple queries
  size_kb INTEGER,
  installed BOOLEAN DEFAULT 0,
  json_content TEXT, -- compressed JSON/YAML of module
  updated_at INTEGER
);

-- normalized steps (optional cache of full flow)
CREATE TABLE steps (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  module_id TEXT,
  step_key TEXT,
  type TEXT,
  message TEXT,
  command TEXT,
  order_num INTEGER,
  extra_json TEXT -- condition, validation rules
);

-- intent patterns
CREATE TABLE intent_patterns (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  module_id TEXT,
  pattern TEXT, -- regex or simple token
  weight REAL DEFAULT 1.0
);

-- dependencies
CREATE TABLE dependencies (
  module_id TEXT,
  requires_module_id TEXT,
  PRIMARY KEY (module_id, requires_module_id)
);

-- state: stores runtime state for active flow per user/session
CREATE TABLE state (
  key TEXT PRIMARY KEY,
  value TEXT
);

-- logs
CREATE TABLE logs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts INTEGER,
  input TEXT,
  resolved_module TEXT,
  confidence REAL,
  method TEXT
);
```

**Indexes**
- `CREATE INDEX idx_intent_pattern ON intent_patterns(pattern);`
- `CREATE INDEX idx_modules_tags ON modules(tags);`

**Notes**
- Keep `json_content` compressed to save disk; import into tables lazily.
- Use `PRAGMA journal_mode=WAL; PRAGMA synchronous = NORMAL;` for speed and reliability.

---

## 6. Flow engine: state machine & pseudocode

### Engine responsibilities
- Evaluate steps, branches, and conditions
- Execute commands only on user confirmation
- Persist runtime `state` in SQLite
- Resolve dependencies and load submodules
- Validate steps via `validate` rules (e.g., run check_command)

### Pseudocode (simplified)

```go
func RunModule(moduleID string) error {
  module := LoadModule(moduleID)
  flow := module.Flows["main"]
  cur := flow.Start
  for {
    step := flow.Steps[cur]
    switch step.Type {
    case "action":
       if step.RunModule != "" {
          RunModule(step.RunModule)
       }
       cur = step.Next
    case "instruction":
       Show(step.Message)
       if step.Command != "" {
         ok := AskUserConfirm("Run command?")
         if ok { RunShell(step.Command) }
       }
       if step.Validate != nil { RunValidators(step.Validate) }
       cur = step.Next
    case "branch":
       key := ReadState(step.BasedOn)
       next := step.Map[key]
       cur = next
    case "terminal":
       Show(step.Message)
       return nil
    }
  }
}
```

**Important behaviors**
- Persist each step transition to `logs` and `state` so user can resume after crash.
- Timeouts for long-running commands, and clear abort/rollback guidance.

---

## 7. Go project layout & key structs
**Project skeleton**

```
cmd/mobilecli/
  main.go
internal/
  engine/
    runner.go
    flow.go
  db/
    db.go
    migration.sql
  modules/
    loader.go
    packer.go
  intent/
    keyword.go
    tinyllm.go
  ui/
    repl.go
  sync/
    client.go
  utils/
    shell.go
pkg/
  models/
    module.go
    step.go
  config/
```

**Key structs (pkg/models/module.go)**

```go
type Module struct {
  ID string `json:"id"`
  Name string `json:"name"`
  Version string `json:"version"`
  Description string `json:"description"`
  Tags []string `json:"tags"`
  Flows map[string]*Flow `json:"flows"`
}

type Flow struct {
  Start string `json:"start"`
  Steps map[string]*Step `json:"steps"`
}

type Step struct {
  Key string `json:"key"`
  Type string `json:"type"`
  Message string `json:"message"`
  Command string `json:"command"`
  RunModule string `json:"run_module,omitempty"`
  Map map[string]string `json:"map,omitempty"` // for branch steps
  Validate []Validation `json:"validate,omitempty"`
}
```

---

## 8. Tiny LLM integration (intent classifier)

**Model type:** classifier-only GGML quantized model (e.g., gpt4all-tiny or custom distilled classifier) sized for 1–10MB.

**Approach**
- Create a small labeled dataset: user utterances → module labels. Start with 200–1000 examples per popular module.
- Export classifier model in GGML/ONNX format.
- Use a Go wrapper (cgo alternative: execute a tiny binary or use `llama.c` bindings via cgo or use `ggml-go` bindings). Prefer pure-Go bindings like `gorgonia` if feasible.
- Model returns `label, confidence`.

**Inference**
- Run classification with a short timeout (100–500ms typical). If hardware slow, fallback to DB search.

**Training**
- Fine-tune or train a simple classifier (fastText, DistilBERT small, or logistic regression on embeddings) on server; export quantized model for mobile.

**Storage**
- Keep model file small; store in app directory.

---

## 9. Online LLM fallback & API considerations
**When to call:**
- both offline layers fail
- user requests explanation
- user allows online mode

**API behavior**
- send minimal context: input text, top N candidate modules, relevant module metadata
- ask for: `label`, `confidence`, and optional short explanation
- enforce rate limits and cost caps

**Prompt template (short)**
```
Given the user's request: "<user_input>", return JSON: {"label": "module.id", "confidence": 0.0, "explain": "short reason"}
Only use module ids from: [list of available module ids]
```

**Security & privacy**
- Do not send system-identifying data without explicit consent.
- Provide opt-in/out for cloud features.

---

## 10. UX / REPL design & commands
**Primary CLI commands**
- `help` - general help
- `search <text>` - search modules
- `run <module_id>` - run a module
- `modules list` - list installed modules
- `modules install <module_id>` - download & install module
- `modules remove <module_id>`
- `settings` - toggle online/auto-download
- `logs` - view past runs

**REPL features**
- Accept free-form text; run intent pipeline; suggest module(s).
- Show compact menu only when ambiguous.
- Keep output minimal and copy-friendly.
- Confirm before running commands; highlight the command to copy/run.
- Allow `--dry-run` to show commands without executing.

**Accessibility**
- Use short lines for small screens
- Respect TERM width
- Avoid colors by default; allow optional color

---

## 11. Module repository, packaging, versioning, updates
**Module format**
- `.tmod` gzipped JSON/YAML with metadata and compressed content

**Registry API**
- `GET /modules` - list metadata
- `GET /modules/:id` - download package

**Versioning**
- Semantic versioning (MAJOR.MINOR.PATCH)
- `module_id@version` format possible for pinning

**Update flow**
- `modules update-index` - fetch latest metadata
- `modules install foo` - download package, import to SQLite
- modules can declare migrations; runner must handle importing new versions

---

## 12. Performance, memory, & storage optimization
- Keep core binary lean (Go static build, strip symbols)
- Lazy-load module JSON into DB; do not parse entire YAML unless needed
- Use `PRAGMA` and small page size for SQLite tuned for flash storage
- Avoid loading tiny LLM unless DB search fails; load model file on demand
- Keep module sizes small; compress packages

**Memory tips**
- Close DB connections quickly
- Stream large commands’ outputs
- Use small buffers

---

## 13. Security, privacy, and safety
- Never run commands without clear explicit user consent
- If `sudo` commands are shown, warn user
- Provide a `--dry-run` by default for dangerous flows
- Opt-in for telemetry; anonymize logs
- Sign module packages (HMAC or signature) to prevent tampering
- Verify downloaded packages before importing
- Rate limit online LLM calls

---

## 14. Testing plan & QA
**Unit tests**
- DB migrations and schema validations
- Module import/export
- Intent pipeline tests (DB search + tinyLLM mocks)

**Integration tests**
- Flow runner across OS branches (mock OS)
- Module chaining tests

**E2E tests (CI)**
- Simulate REPL inputs and validate outputs
- Validate module packaging

**Performance tests**
- Cold start memory usage on 2GB, 3GB, 4GB test devices (emulation)
- Latency for classification & DB search

**Security tests**
- Package tampering
- Command injection checks

---

## 15. CI/CD, builds, releases
- Cross-compile Go for Android/Termux (armv7, arm64, x86_64)
- Use GitHub Actions for builds
- Automate module index generation on release
- Signed releases; checksum files

---

## 16. Metrics & telemetry (privacy-first)
**Collect (opt-in)**
- module run counts
- confidence scores distribution
- error/failure counts

**Do not collect**
- full user command outputs
- personal data

**Use**
- store aggregated metrics on server for improving models and modules

---

## 17. Development milestones & deliverables
**Phase 0 – Research & prototypes (1–2 weeks)**
- Prototype DB schema + flow runner with static modules
- Quick keyword search implementation

**Phase 1 – Core MVP (2–4 weeks)**
- Build core Go binary
- SQLite integration, REPL, module import
- Implement keyword intent pipeline
- Add module registry mock server

**Phase 2 – Tiny LLM + packaging (2–4 weeks)**
- Integrate tiny LLM classifier
- Build training data pipeline and initial classifier
- Add module packaging & download

**Phase 3 – Robustness & UX (3–4 weeks)**
- Flow validation, resume on crash, logs
- Add online LLM fallback
- Security (signing, validation)

**Phase 4 – Testing & Release (2–3 weeks)**
- E2E tests, performance optimization
- Cross-platform builds & publish

---

## 18. VSCode + Copilot/Claude prompts and workflows
**General approach**
- Use Copilot to expand small functions and generate boilerplate
- Use Claude Sonnet for architectural reasoning, writing prompts for the online LLM, and reviewing security and privacy sections

**Suggested Copilot prompt examples**
- "Write a Go function that loads a module YAML file and imports steps into the SQLite `steps` table. Use `database/sql` and parameterized queries. Ensure minimal memory usage."
- "Create a REPL loop in Go that reads user input, calls `DetectIntent`, and routes to `RunModule`. Include command history and `--dry-run` option."

**Suggested Claude prompts**
- "Review this intent classification pipeline and suggest improvements for mobile-first constraints. Include confidence threshold recommendations and fallbacks."
- "Draft a short privacy policy blurb for the CLI app explaining telemetry and online LLM usage in plain language for users in Nigeria and Kenya."

**Local developer workflow**
1. Create a new VSCode workspace with the project skeleton.
2. Use Copilot to scaffold CRUD for DB tables and simple REPL commands.
3. Use Claude Sonnet for high-level code reviews and to draft API prompts and tests.
4. Commit small changes and run unit tests frequently.

---

## 19. Appendix: examples, queries, and snippets

### Example: DB search pseudocode
```go
func DBSearchIntent(db *sql.DB, input string) (moduleID string, score float64) {
  tokens := Tokenize(input)
  candidates := map[string]float64{}
  for _, t := range tokens {
    rows := db.Query("SELECT module_id, weight FROM intent_patterns WHERE pattern LIKE '%'||?||'%'", t)
    for rows.Next() {
      var mid string; var w float64
      rows.Scan(&mid, &w)
      candidates[mid] += w
    }
  }
  // normalize by token count / weight
  // pick top candidate; compute score
}
```

### Example: SQL DDL for indexes
```sql
CREATE INDEX IF NOT EXISTS idx_patterns_pattern ON intent_patterns(pattern);
CREATE INDEX IF NOT EXISTS idx_modules_installed ON modules(installed);
```

### Example: Minimal module YAML (detect_os)
```yaml
name: detect_os
id: org.themobileprof.detect_os
version: 1.0.0
flows:
  main:
    start: check
    steps:
      check:
        type: instruction
        message: "Detecting OS..."
        command: "uname -a"
        validate:
          - parse_uname: true
```

---

# Closing notes
This plan is intentionally pragmatic: deterministic flows for all actionable steps, tiny LLMs for intent classification only, and online LLMs as fallbacks or explainers. The architecture is tuned for low-end phones, intermittent connectivity, and high safety (no hallucinated destructive commands).

If you'd like, I can now:
- Generate the full SQLite migration SQL file,
- Create a sample `detect_os` + `install_mysql` module package,
- Produce Go skeleton code for `modules/loader.go` and `engine/runner.go`, or
- Draft Copilot prompts for each file to speed development in VSCode.

Tell me which one to generate next and I'll produce it directly in this workspace.

