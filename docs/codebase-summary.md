# Solon: Codebase Summary

## Quick Overview

**Solon** is a 2,200+ line Rust CLI + 2 Claude Code plugins for hash-validated file editing, code intelligence (AST-grep + LSP), and workflow orchestration.

| Metric | Value |
|--------|-------|
| **Total LOC** | 2,200+ (Rust crates + root binary) |
| **Binary Size** | ~2.5 MB (release, all subsystems) |
| **Unit Tests** | 27 passing |
| **Integration Tests** | 11 passing |
| **Workspace Crates** | 3 (solon-common, solon-cli, solon-core) |
| **Plugins** | 3 (solon-cli: 5 skills; solon-core: 14 skills + 14 agents + 21 hooks + shared references; solon-skills: 8 domain skills) |
| **Supported Platforms** | Linux, macOS, Windows |
| **Status** | Production-ready (v0.6.0) |

---

## Entry Points

### CLI Binary (`src/main.rs`)

Root binary dispatches to workspace crates:

**File Operations** (solon-cli crate):
- `sl read <FILE> [--lines N:M] [--chunk-size N]`
- `sl edit <FILE> <START#HASH> [<END#HASH>] [<CONTENT>] [--after|--before|--delete]`
- `sl ast search <PATTERN> [--lang LANG] [--path PATH]`
- `sl ast replace <PATTERN> <REPLACEMENT> [--lang LANG] [--path PATH]`
- `sl lsp diagnostics/goto-def/references/hover <FILE> <LINE> <COL>`

**Workflow Operations** (solon-core crate):
- `sl plan resolve [--session ID] [--branch BRANCH]`
- `sl task hydrate <PLAN_DIR>`
- `sl workflow status <PLAN_DIR>`
- `sl report index <PLAN_DIR>`

---

## Workspace Crates

### solon-common
Shared types, utilities, and error handling used by cli and core crates.

### solon-cli
File operations: hashline (read/edit), AST search/replace, LSP queries.

### solon-core
Workflow operations: plan resolution, task hydration, workflow status, reporting.

---

## Module Breakdown (solon-cli)

### 1. Command Handlers (`cmd/` - 480 LOC)

#### `cmd/read.rs` (120 LOC)
Implements `sl read` — reads file and annotates lines with hashes.

**Key Functions:**
- `run(ReadArgs)` — Main entry point
- `parse_range(s: &str)` — Parse "5:20" format to (start, end)

**Args:**
```rust
pub struct ReadArgs {
    pub file: PathBuf,
    pub lines: Option<String>,      // "5:20" or "5:"
    pub chunk_size: usize,          // default 200
}
```

**Flow:**
1. Read file, canonicalize (BOM/line endings)
2. Split into lines, compute CID for each
3. Filter to requested range
4. Print: `LINE#CID|CONTENT`

#### `cmd/edit.rs` (180 LOC)
Implements `sl edit` — edits file via hash-validated line references.

**Key Functions:**
- `run(EditArgs)` — Main entry point
- `is_line_ref(s: &str)` — Check if string is "N#XX" format
- `parse_content_escapes(s: &str)` — Handle escape sequences in content

**Args:**
```rust
pub struct EditArgs {
    pub file: PathBuf,
    pub start_ref: String,              // "5#MQ"
    pub second: Option<String>,         // END_REF or CONTENT
    pub content: Option<String>,        // CONTENT if second is END_REF
    pub after: bool, pub before: bool, pub delete: bool,
    pub stdin: bool, pub no_backup: bool,
}
```

**Operations:**
- Replace: `sl edit file 5#MQ 10#HW "content"` (lines 5-10)
- Insert-after: `sl edit file 5#MQ "content" --after`
- Insert-before: `sl edit file 5#MQ "content" --before`
- Delete: `sl edit file 5#MQ --delete`

**Flow:**
1. Read file, parse lines
2. Validate hashes at start/end positions
3. Apply operation (replace/insert/delete)
4. Generate unified diff
5. Backup original to `.bak`
6. Atomic write (temp file + rename)

#### `cmd/ast.rs` (140 LOC)
Implements `sl ast search` and `sl ast replace`.

**Key Functions:**
- `run(AstArgs)` — Dispatch to search/replace
- Search handler calls `ast::sg::search()`
- Replace handler calls `ast::sg::replace()`

**Args:**
```rust
pub struct SearchArgs {
    pub pattern: String,            // ast-grep pattern
    pub lang: Option<String>,
    pub path: PathBuf,
    pub json: bool,
    pub max_results: usize,
    pub timeout: u64,
}

pub struct ReplaceArgs {
    pub pattern: String,
    pub replacement: String,
    pub lang: Option<String>,
    pub path: PathBuf,
    pub timeout: u64,
}
```

**Flow:**
1. Validate `sg` binary exists
2. Execute `sg` process with pattern
3. Parse JSON output
4. Format results (file:line:col - matched_text)
5. Print with context

#### `cmd/lsp.rs` (160 LOC)
Implements `sl lsp diagnostics/goto-def/references/hover`.

**Key Functions:**
- `run(LspArgs)` — Dispatch to subcommand
- Each subcommand creates LSP client and sends request

**Args:**
```rust
pub struct DiagnosticsArgs { pub file: PathBuf }
pub struct GotoDefArgs { pub file: PathBuf, pub line: u32, pub col: u32 }
pub struct ReferencesArgs { pub file: PathBuf, pub line: u32, pub col: u32 }
pub struct HoverArgs { pub file: PathBuf, pub line: u32, pub col: u32 }
```

**Flow:**
1. Detect appropriate LSP server (rust-analyzer, ts-language-server, etc.)
2. Find project root (Cargo.toml, package.json, etc.)
3. Create LSP client, initialize
4. Send appropriate request (diagnostics, definition, references, hover)
5. Parse response, format output
6. Print results

---

### 2. Hashline Protocol (`hashline/` - 560 LOC)

#### `hashline/hash.rs` (50 LOC)
Computes 2-character content ID (CID) for lines.

**Key Functions:**
```rust
pub fn classify_line(line: &str) -> bool
// Returns true if line contains alphanumeric chars

pub fn compute_cid(line: &str, line_number: usize) -> [u8; 2]
// Hash computation:
// - Alphanumeric: seed=0 (content-based)
// - Blank/punctuation: seed=line_number (differentiate adjacent blanks)
// - Algorithm: xxhash32(content, seed) % 256
// - Mapping: 256 values → 16×16 alphabet grid → 2-char output

pub fn hash_to_cid(line: &str, line_number: usize) -> String
// Returns CID as String
```

**Alphabet:** `ZPMQVRWSNKTXJBYH` (16 chars, chosen for readability)

**Tests (5):**
- Alphanumeric lines use seed zero
- Blank lines use line number seed
- Deterministic output
- Alphabet coverage
- CID format (2 chars)

#### `hashline/canonicalize.rs` (120 LOC)
Normalizes files for processing (strips BOM, detects/normalizes line endings).

**Key Functions:**
```rust
pub fn strip_bom(content: &str) -> String
pub fn normalize_line_endings(content: &str) -> String
pub fn detect_line_ending(content: &str) -> &'static str
pub fn restore_line_endings(content: &str, ending: &str) -> String
pub fn is_binary(content: &str) -> bool
```

**Supported Line Endings:**
- LF (`\n`) — Unix/Linux/macOS
- CRLF (`\r\n`) — Windows
- CR (`\r`) — Old Mac

**BOM Handling:**
- UTF-8 BOM: `EF BB BF`
- UTF-16/32: Not supported

**Tests (6):**
- BOM detection and stripping
- Line ending detection
- Line ending restoration
- Normalization correctness
- Binary file detection
- Mixed line ending handling

#### `hashline/format.rs` (80 LOC)
Renders annotated lines and parses references.

**Key Functions:**
```rust
pub fn annotate_line(line_num: usize, cid: &str, content: &str) -> String
// Output: "5#MQ|some code here"

pub fn parse_line_ref(s: &str) -> Result<(usize, String)>
// Input: "5#MQ" → (5, "MQ")
```

**Tests (5):**
- Annotation format
- Edge cases (empty content, special chars)
- Line ref parsing
- Error handling

#### `hashline/validate.rs` (60 LOC)
Validates line references before applying edits.

**Key Functions:**
```rust
pub fn parse_line_ref(s: &str) -> Result<(usize, String)>
pub fn validate_hash(line: &str, claimed_hash: &str, line_num: usize) -> bool
// Recompute CID, compare with claimed hash
// Returns false if mismatch (prevents stale refs)
```

**Tests (4):**
- Parse valid refs
- Parse invalid refs
- Hash validation success
- Hash validation failure

#### `hashline/edit.rs` (200 LOC)
Core edit engine with atomic writes and diff generation.

**Key Structures:**
```rust
pub enum EditOp {
    Replace { start: usize, end: usize, content: Vec<String> },
    InsertBefore { line: usize, content: Vec<String> },
    InsertAfter { line: usize, content: Vec<String> },
    Delete { start: usize, end: usize },
}
```

**Key Functions:**
```rust
pub fn apply_edit(lines: Vec<String>, op: &EditOp) -> Result<Vec<String>>
// Apply operation to line array, return modified lines

pub fn generate_diff(original: &[String], modified: &[String]) -> String
// Generate unified diff output

pub fn parse_content_escapes(s: &str) -> String
// Handle escape sequences: \n → newline, \t → tab, \\ → backslash
```

**Atomic Write Pattern:**
```rust
let temp_path = path.with_extension("tmp");
fs::write(&temp_path, modified_content)?;
fs::rename(&temp_path, path)?;  // Atomic on POSIX & Windows
```

**Tests (7):**
- Replace operation
- Insert-before/after operations
- Delete operation
- Diff generation correctness
- Escape sequence handling
- Boundary conditions
- Error cases

---

### 3. AST-Grep Integration (`ast/` - 210 LOC)

#### `ast/sg.rs` (90 LOC)
Wrapper around `sg` (ast-grep) binary for semantic search/replace.

**Key Functions:**
```rust
pub fn require_sg() -> Result<()>
// Check if `sg` binary is in PATH
// Error if missing: "ast-grep not found in PATH. Install via: cargo install ast-grep"

pub fn search(pattern: &str, lang: &Option<String>, path: &Path, timeout: Duration)
  -> Result<Vec<Match>>
// Execute: sg search --pattern "pattern" --json [--lang LANG] PATH
// Parse JSON results

pub fn replace(pattern: &str, replacement: &str, lang: &Option<String>,
               path: &Path, timeout: Duration) -> Result<Vec<Replacement>>
// Execute: sg replace --pattern "pattern" --rewrite "replacement" --json [--lang LANG] PATH
```

**Data Structures:**
```rust
pub struct Match {
    pub file: String,
    pub line: usize,
    pub column: usize,
    pub text: String,
    pub context_before: Vec<String>,
    pub context_after: Vec<String>,
}

pub struct Replacement {
    pub file: String,
    pub matches: Vec<Match>,
    pub replacements: Vec<String>,
}
```

**Process Handling:**
- Spawn `sg` via `std::process::Command`
- Pipe stderr to stderr, stdout to parser
- Timeout: default 30 seconds
- Kill process on timeout

#### `ast/format.rs` (100 LOC)
Formats search/replace results for readability.

**Key Functions:**
```rust
pub fn format_search_results(matches: &[Match], max_results: usize) -> String
// Output format:
// file.rs:5:0 - fn main() {
//     context line 1
//     matched line

pub fn format_replace_preview(replacements: &[Replacement]) -> String
// Show before/after diff for each replacement
```

---

### 4. LSP Client (`lsp/` - 550 LOC)

#### `lsp/detect.rs` (150 LOC)
Detects appropriate LSP server and project root.

**Key Functions:**
```rust
pub fn find_project_root(path: &Path) -> Option<PathBuf>
// Look for markers: Cargo.toml (Rust), package.json (Node), setup.py (Python),
// pyproject.toml, go.mod, etc.

pub fn detect_server(path: &Path) -> Result<(String, Vec<String>)>
// Map file extension to server binary name + args
// Examples:
//   .rs → ("rust-analyzer", [])
//   .ts/.js → ("typescript-language-server", ["--stdio"])
//   .py → ("pylsp", [])
//   .go → ("gopls", ["serve"])
```

**Server Detection:**
- Check if binary exists in PATH
- Return error if not found
- Suggest installation command

#### `lsp/client.rs` (280 LOC)
LSP protocol client (JSON-RPC 2.0 implementation).

**Key Structure:**
```rust
pub struct LspClient {
    process: Child,
    stdin: BufWriter<ChildStdin>,
    stdout: BufReader<ChildStdout>,
    request_id: AtomicUsize,
}
```

**Key Methods:**
```rust
pub fn new(server: &str, args: &[String], root: &Path) -> Result<Self>
// Start server process, send initialize request, wait for response

pub async fn diagnostics(&mut self, path: &Path) -> Result<Vec<Diagnostic>>
pub async fn goto_definition(&mut self, path: &Path, line: u32, col: u32)
  -> Result<Vec<Location>>
pub async fn references(&mut self, path: &Path, line: u32, col: u32)
  -> Result<Vec<Location>>
pub async fn hover(&mut self, path: &Path, line: u32, col: u32)
  -> Result<Option<String>>
```

**Internal Methods:**
```rust
fn send_request(&mut self, method: &str, params: Value) -> Result<Value>
// Encode JSON-RPC request, send via stdin, read response

async fn read_response(&mut self) -> Result<Value>
// Read from stdout until complete JSON message

async fn wait_for_notification(&mut self, method: &str) -> Result<Value>
// Listen for server-initiated messages
```

**Protocol Details:**
- JSON-RPC 2.0
- LSP spec v3.17
- Async/await with Tokio
- No connection pooling (stateless per invocation)

#### `lsp/format.rs` (120 LOC)
Formats LSP responses for readability.

**Key Functions:**
```rust
pub fn format_diagnostics(diags: &[Diagnostic]) -> String
// Output: FILE:LINE:COL [LEVEL] MESSAGE

pub fn format_locations(locs: &[Location]) -> String
// Output: FILE:LINE:COL - CONTEXT (show line content)

pub fn format_hover(hover: &str) -> String
// Clean up markdown, highlight code blocks
```

---

## Dependencies

### Runtime Dependencies

| Crate | Version | Purpose | Size Impact |
|-------|---------|---------|-------------|
| `clap` | 4 | CLI argument parsing | Medium |
| `anyhow` | 1 | Error handling | Minimal |
| `xxhash-rust` | 0.8 | Fast hashing (CID) | Small |
| `lsp-types` | 0.97 | LSP protocol types | Medium |
| `tokio` | 1 | Async runtime | Large |
| `serde` | 1 | Serialization | Medium |
| `serde_json` | 1 | JSON parsing | Medium |

### Dev Dependencies

| Crate | Version | Purpose |
|-------|---------|---------|
| `tempfile` | 3 | Test file creation |

### External Binaries (Runtime)

| Binary | Purpose | Optional? |
|--------|---------|-----------|
| `sg` (ast-grep) | AST search/replace | Yes (only if using ast cmds) |
| Language servers | LSP queries | Yes (only if using lsp cmds) |

**Language Server Examples:**
- `rust-analyzer` — Rust
- `typescript-language-server` — TypeScript/JavaScript
- `pylsp` — Python
- `gopls` — Go
- `clangd` — C/C++

---

## Test Structure

### Unit Tests (27 total)

**hashline/hash.rs (5 tests)**
- `alphanumeric_line_uses_seed_zero` — Content-based hashing
- `blank_line_uses_line_number_seed` — Differentiation of blanks
- `deterministic_output` — Same input → same hash
- `alphabet_coverage` — All 256 values reachable
- `cid_format_is_two_chars` — Output always 2 chars

**hashline/canonicalize.rs (6 tests)**
- `detect_lf_line_ending`
- `detect_crlf_line_ending`
- `strip_utf8_bom`
- `normalize_mixed_line_endings`
- `restore_original_line_endings`
- `detect_binary_file`

**hashline/validate.rs (4 tests)**
- `parse_valid_line_ref`
- `parse_invalid_line_ref_fails`
- `validate_hash_matches`
- `validate_hash_mismatch_fails`

**hashline/edit.rs (7 tests)**
- `replace_single_line`
- `replace_range`
- `insert_before_line`
- `insert_after_line`
- `delete_single_line`
- `delete_range`
- `generate_correct_diff`

**hashline/format.rs (5 tests)**
- `annotate_line_format`
- `parse_line_ref_success`
- `parse_line_ref_with_hash`
- `handle_empty_lines`
- `handle_special_characters`

### Integration Tests (11 total)

**read_test.rs (6 tests)**
- `test_read_entire_file` — Full file read with hashes
- `test_read_with_line_range` — Line range filtering
- `test_read_with_line_range_open_end` — "5:" syntax
- `test_read_with_chunk_size` — Chunked output
- `test_read_with_utf8_bom` — BOM handling
- `test_read_with_crlf_endings` — Windows line endings

**edit_test.rs (5 tests)**
- `test_edit_replace_lines` — End-to-end replace
- `test_edit_insert_after` — Insert operation
- `test_edit_delete_line` — Delete operation
- `test_edit_with_hash_validation` — Hash checking
- `test_edit_creates_backup` — .bak file creation

### Test Fixtures

**tests/fixtures/sample.rs**
```rust
fn main() {
    let x = 5;
    println!("{}", x);
}
```

Small Rust file for consistent testing across platforms.

---

## Plugin Architecture

### 2 Claude Code Plugins

**solon-cli Plugin** (`plugins/solon-cli/.claude-plugin/`)
- 5 skills wrapping `sl` file operations
- hashline-read, hashline-edit, ast-search, ast-replace, lsp-tools

**solon-core Plugin** (`plugins/solon-core/.claude-plugin/`)
- 14 skills for workflow operations:
  - **Workflow loop:** sl:brainstorm, sl:plan, sl:ship, sl:test, sl:review (5 skills)
  - **Foundation:** sl:scout (codebase exploration), sl:git (smart commits + PRs) (2 skills)
  - **Core workflow:** sl:fix (bug diagnosis+repair), sl:debug (root cause analysis), sl:refactor (AST+LSP semantic transforms) (3 skills)
  - **Productivity:** sl:docs-seeker (external docs lookup), sl:simplify (post-edit cleanup), sl:watzup (session wrap-up) (3 skills)
  - **Polish:** sl:ask (quick expert consultation), sl:preview (visual explanations + diagrams) (2 skills)
- 14 agents for planning, implementation, testing, reviewing, documentation
- Shared references layer (`references/shared/`): workflow-modes, skill-decision-tree, verification-protocol, subagent-patterns
- Hooks system (21 hooks in hooks.json + Rust binary subcommands)

### Hooks Subsystem (solon-core)

**Location:** `plugins/solon-core/hooks/`

**Binary:** `sl hook` (Rust binary subcommand, 21 hooks)

**Hook Categories (21 total):**
- Session lifecycle (4): session-init, subagent-init, team-context, ship-reminder
- Access control (2): privacy-block, scout-block
- Intent & strategy (1): intent-gate
- Developer guidance (3): dev-rules, usage-awareness, descriptive-name
- Quality assurance (3): post-edit, comment-slop-checker, todo-continuation-enforcer
- Token management (3): preemptive-compaction, tool-output-truncation, semantic-compression
- Knowledge & wisdom (1): wisdom-accumulator
- Context preservation (1): compaction-context-preservation
- Notifications (4): notify, statusline, task-completed, teammate-idle

**Internal Packages (9):**
- `config/` — Configuration management
- `plan/` — Planning context builder
- `privacy/` — Sensitive file patterns
- `scout/` — File visibility validation
- `statusline/` — Progress rendering
- `wisdom/` — Knowledge accumulation
- `intent/` — User intent classification (7 categories)
- `compress/` — Semantic text compression
- `truncation/` — Output truncation budgets

### Marketplace Registration

**File:** `.claude-plugin/marketplace.json`

Registers both plugins (solon-cli v0.6.0, solon-core v0.6.0) in single marketplace.

---

## CI/CD Pipeline

### Build & Test (`.github/workflows/ci.yml`)

**Triggers:** Push to main, pull requests

**Matrix:**
- OS: ubuntu-latest, macos-latest, windows-latest
- Rust: stable

**Steps:**
1. Checkout code
2. Install Rust stable + components (rustfmt, clippy)
3. Cache cargo registry & target
4. Check formatting: `cargo fmt -- --check`
5. Lint: `cargo clippy -- -D warnings`
6. Build: `cargo build --release`
7. Test: `cargo test`

**Status:** All tests pass on all platforms ✅

### Release (`.github/workflows/release.yml`)

**Trigger:** Git tag `v*`

**Matrix:**
- linux-x86_64 (native)
- linux-aarch64 (cross-compile)
- darwin-x86_64 (native)
- darwin-arm64 (native)
- windows-x64 (native)

**Steps:**
1. Build for target platform
2. Rename binary to platform name
3. Compute SHA256 checksum
4. Upload artifact to GitHub
5. Create release with all artifacts + checksums

**Output:** Binaries + checksums available on GitHub Releases page

---

See [System Architecture](./system-architecture.md) for detailed algorithm descriptions.

---

## Performance Profile

### Memory Usage

| Operation | Size | Notes |
|-----------|------|-------|
| Read 1 MB file | ~2 MB | File + lines vec + output buffer |
| Hash computation | O(n) | Linear in file size |
| LSP initialization | ~50 MB | Server process memory (not our code) |
| AST search | Varies | Depends on `sg` server |

### CPU Usage

| Operation | Time | Notes |
|-----------|------|-------|
| Startup | ~30ms | Clap parsing, minimal I/O |
| Hash 1 MB | ~20ms | xxhash32 is very fast |
| Edit + diff | ~5ms | Hash validation + line ops |
| LSP query | ~50-200ms | Server startup dominates |
| AST search | ~200-1000ms | `sg` process spawn + execution |

### Binary Size

| Optimization | Size |
|--------------|------|
| Debug build | ~50 MB |
| Release (default) | ~5 MB |
| Release (stripped + LTO) | ~1.8 MB |

---

## Safety & Security

### Memory Safety
- 100% safe Rust (no `unsafe` blocks)
- All deps reviewed for safety
- No memory leaks (RAII, no manual allocation)

### Input Validation
- File paths validated before read/write
- Line refs validated: must be `\d+#[A-Z]{2}`
- LSP positions validated (0-based line/col)

### Permissions
- Respects filesystem ACLs
- No chmod calls (respects umask)
- Backups inherit original permissions

### DoS Prevention
- AST timeout: 30s default (configurable)
- LSP timeout: implicit via process lifecycle
- No infinite loops in hash/edit code

---

## How to Extend

See [Code Standards](./code-standards.md) for detailed development guidelines.

**Quick Steps:**
- **New Subcommand:** Create `src/cmd/newcmd.rs`, update `cmd/mod.rs`, register in `main.rs`
- **New LSP Language:** Add extension mapping in `lsp/detect.rs` and test
- **New Test:** Add to relevant test module, run `cargo test`

---

---

**Last Updated:** 2026-03-20
**Document Version:** 1.4 (14 skills + 21 hooks in solon-core v0.6.0)
