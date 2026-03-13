# Solon: System Architecture

## High-Level Overview

Solon is a modular CLI + plugin system with three independent subsystems:

1. **Hashline Subsystem** — Line identification & editing via content hashes
2. **AST Subsystem** — Semantic search/replace via ast-grep integration
3. **LSP Subsystem** — Language intelligence via LSP client

Each subsystem is self-contained and can operate independently. The plugin layer exposes all three as callable skills with safety hooks.

---

## Hashline Subsystem

### Purpose
Provide stable, content-based line references that persist across edits in a file.

### Design Philosophy
- **Content First**: Hash is computed from line content, not line number
- **Differentiation**: Blank/punctuation lines use line number in hash to distinguish adjacent identical lines
- **Deterministic**: Same content always produces same hash (xxhash32 with fixed seed)
- **Compact**: 2-character output (CID) = 256 unique values per position

### Flow

```
Input File
    │
    ├─→ [Canonicalize]
    │   └─ Strip BOM, normalize line endings
    │
    ├─→ [Hash Each Line]
    │   └─ Classify (alphanumeric vs blank)
    │   └─ Compute xxhash32 (seed=0 for alpha, seed=line# for blank)
    │   └─ Map to 2-char CID
    │
    ├─→ [Format Output]
    │   └─ Print: LINE#CID|CONTENT
    │
    └─→ [Handle Ranges]
        └─ If --lines: filter to range before output
```

### Key Components

#### `hashline/hash.rs`
Computes 2-character content ID (CID) for each line.

```rust
pub fn compute_cid(line: &str, line_number: usize) -> [u8; 2]
// Seed: 0 for alphanumeric, line_number for blank
// Hash: xxhash32(content, seed) % 256
// Alphabet: 16 chars → 256 2-char codes
```

**Alphabet:** `ZPMQVRWSNKTXJBYH` — chosen for readability (no numbers/symbols to avoid confusion).

#### `hashline/canonicalize.rs`
Normalizes file for hashing (strips BOM, normalizes line endings).

```rust
pub fn strip_bom(content: &str) -> String
pub fn normalize_line_endings(content: &str) -> String
pub fn detect_line_ending(content: &str) -> &'static str
pub fn restore_line_endings(content: &str, ending: &str) -> String
pub fn is_binary(content: &str) -> bool
```

Preserves original line endings in output for atomic writes.

#### `hashline/format.rs`
Renders lines with hash annotations.

```rust
pub fn annotate_line(line_num: usize, cid: &str, content: &str) -> String
// Output: "LINE#CID|CONTENT"
```

#### `hashline/validate.rs`
Parses and validates line references.

```rust
pub fn parse_line_ref(s: &str) -> Result<(usize, String)>
// Input: "5#MQ" → (5, "MQ")
pub fn validate_hash(line: &str, claimed_hash: &str, line_num: usize) -> bool
// Recompute hash, compare with claimed
```

Prevents stale reference attacks (e.g., referencing old hash after file changes).

#### `hashline/edit.rs`
Core edit engine with three operations.

```rust
pub enum EditOp {
    Replace { start: usize, end: usize, content: Vec<String> },
    InsertBefore { line: usize, content: Vec<String> },
    InsertAfter { line: usize, content: Vec<String> },
    Delete { start: usize, end: usize },
}

pub fn apply_edit(lines: Vec<String>, op: &EditOp) -> Result<Vec<String>>
pub fn generate_diff(original: &[String], modified: &[String]) -> String
```

**Edit Flow:**
1. Parse content into lines
2. Validate all line refs (abort if hash mismatch)
3. Apply operation to line array
4. Generate unified diff
5. Write atomically (temp file + rename)

**Atomic Write Pattern:**
```rust
let temp_path = path.with_extension("tmp");
fs::write(&temp_path, modified_content)?;
fs::rename(&temp_path, path)?; // Atomic on POSIX & Windows
```

### Output Format

Each line printed as: `LINE#CID|CONTENT`

```
1#ZP|fn main() {
2#MQ|    println!("hello");
3#HW|
4#RK|}
```

---

## AST Subsystem

### Purpose
Enable semantic code search/replace without line-number fragility.

### Design Philosophy
- **Delegate to Experts**: Use ast-grep (`sg` binary) for pattern matching
- **Thin Wrapper**: CLI wrapping around `sg` process
- **Structured Output**: Parse JSON results, format for Claude readability
- **Timeout Safety**: Default 30s timeout prevents hanging

### Flow

```
Pattern + Language + Path
    │
    ├─→ [Validate Inputs]
    │   └─ Check path exists, language valid
    │
    ├─→ [Launch sg Process]
    │   └─ Execute: sg --pattern "pattern" --lang lang --json
    │
    ├─→ [Parse JSON Results]
    │   └─ Extract: file, line, column, matched_text, context
    │
    ├─→ [Format Output]
    │   └─ Print: FILE:LINE:COL - MATCHED_TEXT
    │   └─ Include code context
    │
    └─→ [Handle Errors]
        └─ Timeout, invalid pattern, no results
```

### Key Components

#### `ast/sg.rs`
Wrapper around `sg` binary.

```rust
pub fn require_sg() -> Result<()>
// Checks if `sg` is in PATH, errors if missing

pub fn search(
    pattern: &str,
    lang: &Option<String>,
    path: &Path,
    timeout: Duration,
) -> Result<Vec<Match>>
// Executes: sg search --pattern "pattern" --json --lang lang path
// Returns: Vec<Match> with file/line/col/text/context

pub fn replace(
    pattern: &str,
    replacement: &str,
    lang: &Option<String>,
    path: &Path,
    timeout: Duration,
) -> Result<Vec<Replacement>>
// Executes: sg replace --pattern "pattern" --rewrite "replacement" --json --lang lang path
```

#### `ast/format.rs`
Formats search/replace results for readability.

```rust
pub fn format_search_results(matches: &[Match], max_results: usize) -> String
// Each match: FILE:LINE:COL - MATCHED_TEXT + context lines

pub fn format_replace_preview(replacements: &[Replacement]) -> String
// Show before/after diff for each replacement
```

### Integration with `cmd/ast.rs`

```
AstArgs (CLI input)
    │
    ├─→ [Check sg installed]
    │
    ├─→ [Dispatch to subcommand]
    │   ├─ Search: call ast::sg::search()
    │   └─ Replace: call ast::sg::replace()
    │
    └─→ [Format & print results]
        └─ Use ast::format for output
```

### Limitations & Guarantees

- **Requires `sg`**: Not bundled; must be installed separately
- **Pattern Language**: Specific to ast-grep syntax (not regex)
- **Language Detection**: Optional `--lang` flag; if missing, `sg` auto-detects
- **Timeout**: Default 30s prevents infinite loops in large codebases

---

## LSP Subsystem

### Purpose
Provide language-aware code intelligence: diagnostics, goto-def, references, hover.

### Design Philosophy
- **Protocol-Based**: Implement JSON-RPC 2.0 LSP spec
- **Multi-Server Support**: Auto-detect appropriate server for file type
- **Streaming I/O**: Read/write to LSP server's stdin/stdout
- **Request/Response**: Stateless query model (request → parse response → exit)

### Flow

```
LSP Query (diagnostics, goto, etc.)
    │
    ├─→ [Detect LSP Server]
    │   ├─ Parse file extension
    │   ├─ Find server in PATH (rust-analyzer, ts-language-server, etc.)
    │   └─ Find project root (look for Cargo.toml, package.json, etc.)
    │
    ├─→ [Initialize LSP]
    │   ├─ Start server process
    │   └─ Send initialize request with workspace
    │
    ├─→ [Execute Query]
    │   ├─ Send diagnostics, goto, references, or hover request
    │   └─ Read response
    │
    ├─→ [Parse & Format Response]
    │   └─ Extract relevant fields (location, hover text, etc.)
    │
    └─→ [Print Output]
        └─ Format for readability
```

### Key Components

#### `lsp/detect.rs`
Detects appropriate language server and project root.

```rust
pub fn find_project_root(path: &Path) -> Option<PathBuf>
// Look for Cargo.toml (Rust), package.json (TS), setup.py (Python), etc.

pub fn detect_server(path: &Path) -> Result<(String, Vec<String>)>
// Returns: (binary_name, argv_template)
// Examples:
//   .rs → ("rust-analyzer", [])
//   .ts/.js → ("typescript-language-server", ["--stdio"])
//   .py → ("pylsp", [])
```

#### `lsp/client.rs`
Core LSP protocol client.

```rust
pub struct LspClient {
    process: Child,
    stdin: BufWriter<ChildStdin>,
    stdout: BufReader<ChildStdout>,
}

impl LspClient {
    pub fn new(server: &str, args: &[String], root: &Path) -> Result<Self>
    // Start server process, send initialize request

    pub async fn diagnostics(&mut self, path: &Path) -> Result<Vec<Diagnostic>>
    pub async fn goto_definition(&mut self, path: &Path, line: u32, col: u32) -> Result<Vec<Location>>
    pub async fn references(&mut self, path: &Path, line: u32, col: u32) -> Result<Vec<Location>>
    pub async fn hover(&mut self, path: &Path, line: u32, col: u32) -> Result<Option<String>>
}
```

**Key Methods:**
- `send_request()` — Encode & send JSON-RPC request
- `read_response()` — Read & parse JSON-RPC response
- `wait_for_notification()` — Handle server-initiated messages

#### `lsp/format.rs`
Formats LSP responses for Claude readability.

```rust
pub fn format_diagnostics(diags: &[Diagnostic]) -> String
// FILE:LINE:COL [LEVEL] MESSAGE

pub fn format_locations(locs: &[Location]) -> String
// FILE:LINE:COL - CONTEXT (line content)

pub fn format_hover(hover: &str) -> String
// Markdown + code blocks
```

### LSP Protocol Details

**Initialization:**
```json
{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "rootPath": "/path/to/project",
    "capabilities": { ... }
  }
}
```

**Diagnostics Request:**
```json
{
  "method": "textDocument/didOpen",
  "params": {
    "textDocument": {
      "uri": "file:///path/to/file.rs",
      "languageId": "rust",
      "text": "..."
    }
  }
}
// Then listen for `textDocument/publishDiagnostics` notifications
```

**Goto-Definition Request:**
```json
{
  "method": "textDocument/definition",
  "params": {
    "textDocument": { "uri": "file:///path/to/file.rs" },
    "position": { "line": 4, "character": 9 }
  }
}
// Response: Location { uri, range: { start: { line, character }, end: ... } }
```

---

## Plugin Layer

### Architecture

```
┌──────────────────────────────────────────────┐
│     Claude Code Plugin (.claude-plugin/)     │
└──────────────────────────────────────────────┘
         │              │              │
    ┌────┴────┐    ┌────┴────┐    ┌───┴─────┐
    │ Skills  │    │  Hooks  │    │Install  │
    │ (5 JS)  │    │(2 CJS)  │    │ Script  │
    └────┬────┘    └────┬────┘    └───┬─────┘
         │              │              │
    ┌────┴────────────┬─┴──────────────┴────┐
    │                 │                      │
    ▼                 ▼                      ▼
[Skill Handlers] [Safety Hooks]      [Binary Download]
    │                 │                      │
    └─────────────────┼──────────────────────┘
                      │
                      ▼
         ┌─────────────────────────┐
         │   Solon CLI (sl)        │
         │   Binary in PATH        │
         └─────────────────────────┘
```

### Skills (5 total)

Each skill wraps a `sl` subcommand:

#### 1. `skills/hashline-read`
```markdown
# Hashline Read
Use `sl read` to annotate files with line hashes

- `sl read FILE`
- `sl read FILE --lines START:END`
- `sl read FILE --chunk-size N`
```

#### 2. `hashline-edit`
```markdown
# Hashline Edit
Edit via hash-validated line references

- Replace: `sl edit FILE START#HASH END#HASH "content"`
- Insert: `sl edit FILE LINE#HASH "content" --after/--before`
- Delete: `sl edit FILE LINE#HASH --delete`
```

#### 3. `ast-search`
```markdown
# AST Search
Semantic code search using ast-grep patterns

- `sl ast search PATTERN --lang LANG --path PATH`
```

#### 4. `ast-replace`
```markdown
# AST Replace
Semantic code replace

- `sl ast replace PATTERN REPLACEMENT --lang LANG --path PATH`
```

#### 5. `lsp-tools`
```markdown
# LSP Tools
Language server queries: diagnostics, goto-def, references, hover

- `sl lsp diagnostics FILE`
- `sl lsp goto-def FILE LINE COL`
- `sl lsp references FILE LINE COL`
- `sl lsp hover FILE LINE COL`
```

### Safety Hooks (2 total)

#### 1. `hooks/privacy-block.cjs`
Blocks reading/editing of sensitive files:
- `.env`, `.env.*`
- `.aws/`, `.ssh/`
- `secrets/`, `credentials/`
- `.git/`, node_modules/
- `*.pem`, `*.key`

Returns user-friendly error: *"This file is protected for privacy/security reasons."*

#### 2. `hooks/scout-block.cjs`
Respects Claude Code's file visibility rules:
- Reads `.scoutignore` or scout config
- Filters output to only files user can access
- Prevents exposing files outside workspace

---

## Data Flow Diagrams

### Read Flow

```
User: sl read src/main.rs --lines 5:20
    │
    ├─→ Parse args: file=src/main.rs, range=(5,20)
    │
    ├─→ Read file → content: String
    │
    ├─→ Canonicalize: strip BOM, detect/normalize line endings
    │
    ├─→ Split lines: Vec<String>
    │
    ├─→ Compute CID for each line
    │   ├─ Line 5: classify_line("    let x = 5;") → true (alphanumeric)
    │   ├─         hash = xxh32("    let x = 5;", seed=0)
    │   └─         cid = "MQ" (from alphabet[high][low])
    │
    ├─→ Filter to range 5:20
    │
    ├─→ Format & print
    │   └─ "5#MQ|    let x = 5;"
    │   └─ "6#VR|    println!(\"{}\", x);"
    │   └─ ...
    │
    └─→ Exit with code 0
```

### Edit Flow

```
User: sl edit src/main.rs 5#MQ 10#HW "// comment"
    │
    ├─→ Parse args: file, start_ref=(5, "MQ"), end_ref=(10, "HW"), content
    │
    ├─→ Read file → lines: Vec<String>
    │
    ├─→ Validate hashes
    │   ├─ Compute CID for line 5 in current file
    │   ├─ Compare with provided "MQ": MATCH ✓
    │   ├─ Compute CID for line 10
    │   └─ Compare with provided "HW": MATCH ✓
    │   └─ If mismatch: error "Hash validation failed (file changed?)"
    │
    ├─→ Apply edit (Replace 5:10)
    │   └─ new_lines = lines[0:5] + ["// comment"] + lines[10:]
    │
    ├─→ Generate diff
    │   └─ Print unified diff to stdout
    │
    ├─→ Backup original
    │   └─ cp src/main.rs src/main.rs.bak
    │
    ├─→ Atomic write
    │   ├─ Detect original line ending (LF, CRLF, CR)
    │   ├─ Restore line endings in new_lines
    │   ├─ Write to src/main.rs.tmp
    │   └─ Rename .tmp → .rs (atomic)
    │
    └─→ Exit with code 0 + print diff
```

### AST Search Flow

```
User: sl ast search "fn $NAME($$$ARGS)" --lang rust --path src/
    │
    ├─→ Validate input: pattern, lang, path exists
    │
    ├─→ Check `sg` in PATH: require_sg() → (found ✓)
    │
    ├─→ Spawn `sg` process
    │   └─ argv: ["sg", "search", "--pattern", "fn $NAME($$$ARGS)",
    │            "--lang", "rust", "--json", "src/"]
    │
    ├─→ Read JSON output
    │   └─ [{"file": "src/main.rs", "line": 5, "col": 0, "text": "fn main()"},
    │      {"file": "src/lib.rs", "line": 10, "col": 0, "text": "fn helper()"}]
    │
    ├─→ Parse into Match structs
    │
    ├─→ Format for output
    │   └─ "src/main.rs:5:0 - fn main() {"
    │   └─ "    content"
    │   └─ "src/lib.rs:10:0 - fn helper()"
    │   └─ "    content"
    │
    └─→ Print + exit
```

### LSP Goto-Def Flow

```
User: sl lsp goto-def src/main.rs 10 15
    │
    ├─→ Parse args: file, line=10, col=15
    │
    ├─→ Detect server
    │   ├─ File extension: .rs → Language: Rust
    │   ├─ Find server: rust-analyzer in PATH ✓
    │   └─ Find project root: look for Cargo.toml → /home/user/project
    │
    ├─→ Start LSP client
    │   ├─ Spawn rust-analyzer process
    │   ├─ Send initialize request with workspace
    │   └─ Receive initialized notification
    │
    ├─→ Send textDocument/definition request
    │   └─ { "uri": "file:///home/user/project/src/main.rs",
    │      "position": { "line": 9, "character": 14 } }  // 0-based
    │
    ├─→ Receive response
    │   └─ { "uri": "file:///home/user/project/src/lib.rs",
    │      "range": { "start": { "line": 20, "character": 3 }, ... } }
    │
    ├─→ Format for output
    │   └─ "src/lib.rs:21:4 - pub fn helper_func() {"
    │
    └─→ Print + exit
```

---

## Error Handling Strategy

### Graceful Degradation

All subsystems follow this pattern:

```
1. Input Validation
   └─ Fail fast with clear error message (e.g., "File not found: ...")

2. Process Execution
   └─ Handle timeout, non-zero exit, broken pipe

3. Output Parsing
   └─ If JSON invalid, report and fallback to raw output

4. Cleanup
   └─ Kill spawned processes, remove temp files, log to stderr
```

### Error Types

| Error | Handling | Message |
|-------|----------|---------|
| File not found | Exit 1 | `File not found: <path>` |
| Binary not found | Exit 1 | `ast-grep not found in PATH. Install via: cargo install ast-grep` |
| Hash mismatch | Exit 1 | `Hash validation failed at line N (file changed?)` |
| LSP server crash | Exit 1 | `Language server exited unexpectedly: <stderr>` |
| Timeout | Exit 124 | `Operation timed out after Xs` |
| Invalid input | Exit 1 | `Invalid <field>: <reason>` |

---

## Performance Considerations

### Optimization Strategies

1. **Chunked Output** — Read/format in chunks (default 200 lines) to avoid memory bloat
2. **Process Reuse** — LSP/AST processes spawn once per query (not per file)
3. **Binary Release** — LTO + strip enabled in release profile → ~1.8 MB binary
4. **Hash Computation** — xxhash32 is ~1ns per byte (fast hashing)

### Benchmarks (Approximate)

| Operation | Time | Notes |
|-----------|------|-------|
| Read 1 MB file | <50ms | Hash computation + annotation |
| Edit 100-line range | <10ms | Hash validation, diff generation |
| AST search (1000 matches) | <500ms | Dominated by `sg` process startup |
| LSP diagnostics | <100ms | Server cold start, then fast |
| LSP goto-def | <50ms | After server initialized |

---

## Thread Safety & Concurrency

### Current Model
- **Async I/O** — Tokio runtime for LSP (async/await)
- **Sync Editing** — Hashline operations are sync (no shared state)
- **Process-Per-Query** — New LSP client per `sl lsp` invocation (stateless)

### Future Considerations
- **Daemon Mode** — Keep LSP server running between queries (not implemented)
- **Concurrent Edits** — Lock file during atomic write (not multi-process safe)
- **Connection Pooling** — Cache LSP client for multiple queries (optimization)

---

## Security Model

### Threat Model

| Threat | Mitigation |
|--------|-----------|
| Read sensitive files (.env, keys) | Privacy hook blocks patterns |
| Edit outside workspace | Scout hook validates file paths |
| LSP server injection | Validate server binary location |
| Malicious patterns in AST | ast-grep validates; no code execution |
| Concurrent edits → data loss | Atomic writes + backup files |
| LSP timeout DoS | Default 30s timeout per operation |

### Code Injection Prevention

- **No `eval()` or `system()` of user input** — All user input validated
- **AST Patterns** — ast-grep (C++) handles parsing; we just invoke `sg` binary
- **LSP Requests** — JSON-RPC format strictly parsed; no string interpolation

### File Permissions

- **Read** — Respects filesystem permissions; attempts to read beyond permission fail gracefully
- **Write** — Only writes to specified file; atomic rename prevents partial writes
- **Backup** — `.bak` file inherits permissions from original

---

## Testing Architecture

### Unit Test Coverage

**hashline module** (5 tests)
- Hash determinism
- CID alphabet correctness
- Line classification edge cases

**canonicalize module** (6 tests)
- BOM detection/strip
- Line ending detection/restoration
- Binary file detection

**validate module** (4 tests)
- Line ref parsing
- Hash validation success/failure

**edit module** (7 tests)
- Replace, insert-before, insert-after, delete operations
- Diff generation correctness

**format module** (5 tests)
- Annotation formatting
- Error message formatting

### Integration Test Coverage

**read_test.rs** (6 tests)
- Full file read
- Line range queries
- Chunk boundaries
- Binary file handling

**edit_test.rs** (5 tests)
- Replace operation end-to-end
- Insert/delete operations
- Diff correctness
- File restoration on error

### Test Fixtures

- `tests/fixtures/sample.rs` — Small Rust file with various line types
- `tests/fixtures/utf8.txt` — UTF-8 with BOM and CRLF
- Temp files created per-test via `tempfile` crate

---

## Deployment Architecture

### Build Pipeline

```
Source Code (Rust)
    │
    ├─→ [Cargo Build]
    │   └─ Debug/Release builds
    │
    ├─→ [Cross-Compilation]
    │   ├─ Ubuntu → linux-gnu (x64, arm64)
    │   ├─ macOS → darwin (x64, arm64)
    │   └─ Windows → windows-msvc (x64)
    │
    ├─→ [Artifact Signing]
    │   └─ SHA256 checksums
    │
    └─→ [GitHub Release]
        └─ Binary + checksums + release notes
```

### Distribution

1. **Direct Binary Download** — From GitHub Releases
2. **Plugin Installation** — `./scripts/install.sh` downloads + configures
3. **PATH Integration** — Binary placed in user's PATH or plugin directory

### Version Management

- **Semantic Versioning** — vMAJOR.MINOR.PATCH (e.g., v0.1.0)
- **Git Tags** — `git tag v0.1.0 && git push --tags`
- **Release Automation** — GitHub Actions auto-builds on tag push

---

## Future Architecture Improvements

### Phase 2 (Potential)
1. **Daemon Mode** — Long-running `sl daemon` with connection pooling
2. **Protocol Buffer** — Replace JSON-RPC with faster binary protocol
3. **Embedded LSP** — Link rust-analyzer as library (not subprocess)
4. **Incremental Indexing** — Cache AST/symbol tables across queries
5. **Web UI** — Browser interface for previewing edits

### Scalability
- **Horizontal** — Multiple `sl` processes handle concurrent edits
- **Vertical** — Optimize hash computation, AST parsing for large files
- **Storage** — Use memory-mapped I/O for files >1 GB

---

**Last Updated:** 2026-03-13
**Architecture Version:** 1.0
