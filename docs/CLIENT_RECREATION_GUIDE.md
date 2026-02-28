# CLIPilot Client Recreation Guide

This document serves as a blueprint for recreating the functionalities of the CLIPilot client. It details the core logic, data structures, and algorithms required to build a compatible client from scratch.

## 1. Core Architecture

The client is an interactive REPL (Read-Eval-Print Loop) that acts as a natural language interface for the system shell.

**Key Components:**
*   **REPL:** Handles user input, history, and output formatting.
*   **Intent Detector:** The "brain" that maps natural language to actionable commands or modules.
*   **Command Catalog (Static):** Compiled list of common tools for instant lookup.
*   **Executor:** Safely executes shell commands.
*   **Registry Client:** Connects to the remote server for semantic search.

## 2. Keyword Isolation Process

To effectively search both the offline catalog and `man` pages, the client **must** isolate the most important keywords from a free-form query *before* running detection layers.

**Process:**
1.  **Tokenization:** Split input string by spaces.
2.  **Normalization:** Convert to lowercase, remove punctuation.
3.  **Stopword Removal:** Filter out common English words ("the", "is", "how", "to", "I", "want", etc.).
4.  **Synonym Expansion:** (Optional) Map common verbs to system terms (e.g., "duplicate" -> "copy").
5.  **Selection:** The remaining tokens are treated as the core search keywords.

**Example:**
*   Input: "How do I duplicate a directory?"
*   Tokens: `["how", "do", "i", "duplicate", "a", "directory"]`
*   Filter: `["duplicate", "directory"]`
*   Keywors: `["copy", "directory"]` (after limit expansion)

## 3. Intent Detection Algorithm (The "Brain")

The heart of CLIPilot is the `Detect(input)` function. It uses the **isolated keywords** from Section 2 in a 4-layer fallback strategy.

### Layer 1: Offline Basic Commands (Static)
*   **Source:** Compiled Key-Value pair (in binary) or a simple text file. **NOT** a database.
*   **Input:** Isolated Keywords.
*   **Goal:** Instant resolution for common tools (e.g., `git`, `docker`) or known keywords (e.g., "list" -> `ls`).
*   **Implementation:** A hardcoded map or `common_commands.txt` loaded into memory at startup.
*   **Optimization for 80% Resolution (Low Memory):**
    To achieve high resolution on a 2GB device without heavy DB lookups:
    1.  **Compiled Alias Map:** Use a Go `map[string]string` compiled into the binary. This is ~10x faster and uses less memory than parsing a text file at startup.
    2.  **Verb-Noun Pairing:** Map common actions to tools (e.g., `list files`->`ls`) using simple string splitting, avoiding regex.
    3.  **Fuzzy Matching:** Use a lightweight Levenshtein implementation only when exact match fails.
    4.  **Stemming:** Simple suffix checking (`installing` -> `install`) instead of full NLP libraries.
    5.  **Curated "Top 100":** Hardcode the most essential commands to ensure they are instant.

### Layer 2: Man Pages (System Search)
*   **Prerequisite:** `man` must be installed.
*   **Input:** Isolated Keywords.
*   **Logic:**
    1.  **Iterative Search:** For *each* isolated keyword, run `man -k -s 1,8 [keyword]`.
    2.  **Collection:** Gather all `name - description` lines from all runs.
    3.  **Post-Processing:**
        *   Count occurrence frequency (tools matching multiple keywords rank higher).
        *   Calculate word overlap between description and original query.
        *   Check for exact name matches.
    4.  **Selection:** Return the highest-ranked tool installed on the system.
*   **Goal:** Leverage the OS's built-in manual pages to find installed tools the user describes.

### Layer 3: Client Modules
*   **Source:** SQLite `modules` table (and `intent_patterns`).
*   **Logic:**
    1.  Search the local database for installed automations/modules that match the user's intent.
    2.  This allows custom workflows to override or augment system commands.

### Layer 4: Remote Fallback (Online)
*   **Source:** Remote Registry API (`POST clipilot.themobileprof.com/api/commands/search`).
*   **Logic:**
    1.  If all local layers fail.
    2.  Send query to server.
    3.  Server uses LLM/Vector search to find relevant commands not present locally.

## 4. REPL Presentation & Interaction

The client provides an interactive, menu-driven interface. Below is the **exact UI** from the project:

**Match Found Display:**
```text
✓ Command found: cp
────────────────────────
Purpose : Copy files and directories

What would you like to do?
  1) Show examples and usage  (recommended)
  2) Run the command          (interactive)
  3) Show command only        (exit)
  4) Search for another command
  0) Cancel

Choice [1-4, 0]: 
```

**Interaction Flow:**
*   **Option 1 (Show examples):** Shows detailed help (like `man cp` snippet).
*   **Option 2 (Run command):** Pre-fills the REPL line with `cp ` or enters interactive mode.
*   **Option 3 (Show command):** Just prints the command usage and exits the menu.
*   **Option 4 (Search again):** Prompts "What would you like to search for instead?", enabling refinement loop.
*   **Option 0 (Cancel):** Returns to main prompt.

## 5. Critical Components Responsibilities

| Component | Responsibility |
|-----------|----------------|
| **Entry Point** | Initializes Configuration, Static Maps (Layer 1), and REPL. |
| **REPL Loop** | Handles `ReadLine`, special commands (`exit`, `clear`), and calls Detector. |
| **Intent Detector** | Implements the 4-layer detection logic. |
| **Database Manager** | Handles SQLite connection (Layer 3) and migrations. |
| **Safe Executor** | **Essential for Android**: Wraps `exec.LookPath` to avoid `faccessat2` syscall crashes on older Android kernels. |

## 6. Recreating the "Magic" (Tips)
*   **Termux Compatibility:** You MUST use `safeexec` logic. Standard Go `exec.Command` can crash on some Android devices due to syscall blocking.
*   **Embedded Database:** Use `modernc.org/sqlite` (CGO-free) to ensure the binary runs everywhere without system dependencies.
*   **Pre-seeding:** Ensure the Layer 1 static map is comprehensive (see Optimization section).
*   **Fast Startup:** The REPL must start in <200ms. Load DB connections lazily if possible.

## 7. Optimizing for 2GB Termux (Critical)

Your target environment is resource-constrained. Efficiency is paramount.

*   **Avoid CGO Dependency:** Use `modernc.org/sqlite` instead of `mattn/go-sqlite3`. This ensures the binary is static and portable without requiring a C compiler on the device.
*   **Lazy Loading:** Do NOT load the `modules` or `intent_patterns` database tables until Layer 3 is actually needed.
*   **Memory Management:**
    *   Avoid loading large text files into memory. use `bufio.Scanner` if you must read files.
    *   Prefer **Compiled Maps** (Layer 1) over in-memory caches regarding large datasets.
*   **Binary Size:** Strip debug symbols (`go build -ldflags="-s -w"`) to keep the binary small (sub-10MB).
*   **Syscall Safety:** Older Android kernels block certain syscalls (like `faccessat2`). You **MUST** use a `safeexec` wrapper for all command lookups to prevent SIGSYS crashes.
