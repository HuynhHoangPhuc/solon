# Solon: User Guide

## Installation

### Prerequisites

- Linux, macOS, or Windows
- Bash or equivalent shell
- Optional: `ast-grep` (for AST commands), language servers (for LSP commands)

### Quick Start

```bash
# Download install script
git clone https://github.com/solon-dev/solon.git
cd solon

# Run installer
./scripts/install.sh

# Verify installation
sl --version
```

**What it does:**
1. Detects your OS and architecture
2. Downloads the appropriate `sl` binary from GitHub Releases
3. Verifies SHA256 checksum
4. Places binary in `~/.local/bin/sl` (or similar PATH location)
5. Prints confirmation message

### Manual Installation

If you prefer to build from source:

```bash
# Install Rust (if needed)
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Clone and build
git clone https://github.com/solon-dev/solon.git
cd solon
cargo build --release

# Copy binary to PATH
cp target/release/sl ~/.local/bin/
chmod +x ~/.local/bin/sl
```

### Install Language Servers (Optional)

For LSP queries, install relevant language servers:

```bash
# Rust
cargo install rust-analyzer

# TypeScript/JavaScript
npm install -g typescript-language-server

# Python
pip install python-lsp-server

# Go
go install github.com/golang/tools/gopls@latest

# More: https://microsoft.github.io/language-server-protocol/implementors/servers/
```

### Install AST-Grep (Optional)

For AST commands:

```bash
cargo install ast-grep
```

---

## Quick Start Examples

### Reading Files with Hashes

```bash
# Read entire file with line hashes
sl read src/main.rs

# Output format: LINE#HASH|CONTENT
# 1#ZP|fn main() {
# 2#MQ|    println!("hello");
# 3#HW|}
```

Read with line range:

```bash
# Read lines 5-20 only
sl read src/main.rs --lines 5:20

# Read from line 10 to end of file
sl read src/main.rs --lines 10:

# Read from start to line 50
sl read src/main.rs --lines :50
```

Control output size:

```bash
# Limit output to 50 lines per chunk
sl read src/main.rs --chunk-size 50
```

### Editing Files with Hash References

Replace a line range:

```bash
# Replace lines 5-10 with new content
sl edit src/main.rs 5#MQ 10#HW "// new implementation"

# Output shows unified diff + creates src/main.rs.bak
```

Insert content:

```bash
# Insert after line 5
sl edit src/main.rs 5#MQ "    let new_var = 42;" --after

# Insert before line 5
sl edit src/main.rs 5#MQ "// comment above" --before
```

Delete lines:

```bash
# Delete line 5
sl edit src/main.rs 5#MQ --delete

# Delete range 5-10 (provide both refs with --delete)
sl edit src/main.rs 5#MQ 10#HW --delete
```

Read from stdin:

```bash
# Content via pipe instead of command argument
echo "new content here" | sl edit src/main.rs 5#MQ --stdin
```

Skip backup:

```bash
# Don't create .bak file
sl edit src/main.rs 5#MQ "content" --no-backup
```

### Semantic Code Search

```bash
# Search for function definitions (Rust)
sl ast search "fn $NAME($$$ARGS)" --lang rust --path src/

# Search in TypeScript
sl ast search "export const $NAME = ($$$ARGS) =>" --lang typescript

# JSON output for programmatic use
sl ast search "class $NAME" --lang python --json

# Limit results
sl ast search "TODO" --max-results 10

# Custom timeout
sl ast search "fn main" --timeout 60  # 60 seconds
```

**Output format:**
```
src/main.rs:5:0 - fn main() {
    context line
    fn main() {
    more context
```

### Semantic Code Replacement

```bash
# Replace all fn declarations with async fn
sl ast replace "fn $NAME($$$ARGS)" "async fn $NAME($$$ARGS)" --lang rust --path src/

# Returns preview of changes (no write to disk without confirmation)
```

### Language Server Queries

```bash
# Show errors and warnings in a file
sl lsp diagnostics src/main.rs

# Jump to definition at line 10, column 5 (1-based)
sl lsp goto-def src/main.rs 10 5

# Find all references to symbol at position
sl lsp references src/main.rs 10 5

# Show hover info (type info, docstring)
sl lsp hover src/main.rs 10 5
```

**Output format (goto-def):**
```
src/lib.rs:20:3 - pub fn helper_func() {
    fn helper_func() {
    implementation
```

---

## Understanding the Hashline Format

### What are Line Hashes (CIDs)?

Each line in output has a 2-character hash appended:

```
1#ZP|fn main() {
2#MQ|    println!("hello");
3#HW|}
```

The hash is **deterministic** based on line content:
- Same content always produces same hash
- Different content produces different hash (usually)

**Why?** Line numbers change as you edit. Hashes don't.

### How to Use Hashes

When editing, reference lines by hash:

```
File state at read time:
5#MQ|    let x = 5;
6#VR|    println!("{}", x);

You want to replace these lines. Use:
sl edit file.rs 5#MQ 6#VR "    let y = 10;"

The hashes prove you read lines 5-6, preventing accidental edits
to wrong lines if file changed in between.
```

### Hash Validation

If the file has changed and hashes don't match:

```bash
sl edit file.rs 5#MQ "content"
# Error: Hash validation failed at line 5 (file changed?)

# Solution: Re-read file to get new hashes
sl read file.rs
# Then use new hashes in edit command
```

---

## Workflow

The full development workflow chains skills in sequence:

### Core Workflow Loop

| Step | Skill | Description |
|------|-------|-------------|
| 1 | `/sl:brainstorm` | Generate ideas and problem framing for a feature |
| 2 | `/sl:plan` | Create a structured implementation plan with phase files |
| 3 | `/sl:ship` | Execute the plan phase by phase via subagents |
| 4 | `/sl:test` | Run tests and validate implementation quality |
| 5 | `/sl:review` | Code review, cleanup, and final sign-off |

### Supporting Skills (14 total)

| Skill | Description |
|-------|-------------|
| `/sl:scout` | Fast codebase exploration using parallel agents |
| `/sl:git` | Git operations with conventional commits, security scanning, PR creation |
| `/sl:fix` | Structured bug fix: diagnose → fix → verify (auto-activates for bug fixes) |
| `/sl:debug` | Systematic root cause analysis with evidence chain |
| `/sl:refactor` | Semantic refactoring via AST-grep + LSP (renames, transforms, migrations) |
| `/sl:docs-seeker` | External documentation lookup via context7.com llms.txt |
| `/sl:simplify` | Post-edit code cleanup (dead code, DRY violations, complexity) |
| `/sl:watzup` | Session wrap-up summary (what's done, remaining, blockers) |
| `/sl:ask` | Quick technical Q&A with project-aware context |
| `/sl:preview` | Visual explanations: ASCII diagrams, Mermaid charts, architecture viz |

**Total: 14 skills** (5 workflow + 2 foundation + 3 core + 3 productivity + 1 polish)

### `sl` CLI Workflow Commands Reference

| Subcommand | Description |
|------------|-------------|
| `sl plan scaffold --slug <name> --mode fast` | Create a new plan directory |
| `sl plan resolve` | Print active plan path (session → branch fallback) |
| `sl plan validate <dir>` | Check plan completeness and todo counts |
| `sl task hydrate <dir>` | Extract task list from phase files |
| `sl task sync <dir> --phases 1,2` | Mark phases as completed |
| `sl workflow status <dir>` | Show progress (completed/pending/in-progress) |
| `sl report index <dir>` | List report files in a plan |

---

## Common Workflows

### Scenario 1: Fix a Bug in a Specific Function

```bash
# 1. Read file to find the function
sl read src/main.rs --lines 1:50

# Output shows line numbers + hashes:
# 10#ZP|fn buggy_function() {
# 11#MQ|    let x = wrong_value;
# 12#HW|    x + 1
# 13#RK|}

# 2. Fix line 11 using hash
sl edit src/main.rs 11#MQ "    let x = correct_value;"

# 3. Verify the change
sl read src/main.rs --lines 10:14
```

### Scenario 2: Add Error Handling

```bash
# 1. Find the function that needs error handling
sl lsp goto-def src/main.rs 10 5  # Jump to definition

# 2. Read the function
sl read src/main.rs --lines 15:30

# 3. Insert error handling after opening brace
sl read src/main.rs --lines 15:15
# 15#ZP|fn helper(x: i32) -> Result<i32> {

sl edit src/main.rs 15#ZP "fn helper(x: i32) -> Result<i32> {
    if x < 0 { return Err(anyhow!(\"invalid\")) }" --after

# 4. Verify
sl read src/main.rs --lines 15:25
```

### Scenario 3: Find All Uses of a Pattern

```bash
# 1. Semantic search for pattern
sl ast search "println\!" --lang rust --path src/

# 2. Review results (file:line:col shown)
# src/main.rs:15:4 - println!("{}", x);
# src/lib.rs:30:8 - println!("Debug: {}", y);

# 3. Remove unnecessary println! at main.rs:15
sl edit src/main.rs 15#<HASH> --delete

# Use sl read to get the exact hash first if needed
sl read src/main.rs --lines 15:15
```

### Scenario 4: Multi-File Refactoring

```bash
# 1. Find function definition
sl ast search "fn process($$$ARGS)" --lang rust --path src/

# 2. Update signature in implementation file
sl read src/lib.rs --lines 50:60
sl edit src/lib.rs 55#MQ 55#MQ "fn process(data: &[u8], config: Config) -> Result<Vec<u8>> {"

# 3. Find and update all call sites
sl ast search "process\(" --lang rust --path src/

# 4. Update each call site with new signature
# (repeat edit commands for each file)
```

### Scenario 5: Code Review with Diagnostics

```bash
# 1. Check for errors in a file
sl lsp diagnostics src/main.rs

# Output shows issues:
# src/main.rs:10:5 [error] expected identifier, found keyword `let`
# src/main.rs:15:10 [warning] unused variable `x`

# 2. Fix errors using sl edit with hashes

# 3. Verify fixes
sl lsp diagnostics src/main.rs  # Should show fewer issues
```

---

## Pattern Reference

### AST-Grep Patterns

Patterns use placeholders like `$NAME` (single node) and `$$$ARGS` (multiple nodes).

**Rust Examples:**

```
# Function definitions
fn $NAME($$$ARGS) { $$$BODY }
fn $NAME($$$ARGS) -> $TYPE { $$$BODY }

# Variable assignments
let $VAR = $VALUE;
let mut $VAR = $VALUE;

# Function calls
$FUNC($$$ARGS)
$OBJ.$METHOD($$$ARGS)

# Type annotations
$VAR: $TYPE

# Match expressions
match $EXPR { $$$CASES }
```

**TypeScript/JavaScript Examples:**

```
# Function declarations
function $NAME($$$ARGS) { $$$BODY }
const $NAME = ($$$ARGS) => { $$$BODY }

# Variable declarations
const $VAR = $VALUE;
let $VAR = $VALUE;

# Import statements
import { $$$ITEMS } from "$MODULE"

# Function calls
$FUNC($$$ARGS)
await $FUNC($$$ARGS)
```

**Python Examples:**

```
# Function definitions
def $NAME($$$ARGS): $$$BODY

# Class definitions
class $NAME($$$BASE): $$$BODY

# Variable assignments
$VAR = $VALUE

# Function calls
$FUNC($$$ARGS)
```

For more patterns, see: https://ast-grep.github.io/

---

## Troubleshooting

### Issue: "Hash validation failed at line N"

**Cause:** File was modified since you read it.

**Solution:**
```bash
# Re-read the file to get updated hashes
sl read file.rs --lines N:N

# Then use new hash in edit command
sl edit file.rs N#NEWHASH "content"
```

### Issue: "Language server not found"

**Cause:** LSP server not installed or not in PATH.

**Solution:**
```bash
# Check if server is installed
which rust-analyzer

# If not, install it
cargo install rust-analyzer

# Or use installation instructions from language docs
```

### Issue: "ast-grep not found in PATH"

**Cause:** `sg` binary not installed.

**Solution:**
```bash
cargo install ast-grep

# Verify
which sg
```

### Issue: LSP query returns nothing

**Cause:**
1. Wrong file path
2. Language server doesn't support this operation
3. Server is still starting up

**Solution:**
```bash
# Verify file exists
ls -la src/main.rs

# Check if server supports the query
# (not all servers support all LSP features)

# Try explicit language setting if available
# (some servers auto-detect, others need hint)
```

### Issue: Edit creates unwanted `.bak` file

**Cause:** Backup file creation is default behavior.

**Solution:**
```bash
# Use --no-backup flag to skip backup
sl edit file.rs 5#MQ "content" --no-backup

# Or remove .bak file manually
rm file.rs.bak
```

### Issue: Large file output is cut off

**Cause:** Default chunk size (200 lines) limits output.

**Solution:**
```bash
# Increase chunk size
sl read large_file.rs --chunk-size 500

# Or read specific line range instead
sl read large_file.rs --lines 100:200
```

### Issue: Pattern matching returns no results

**Cause:** Pattern syntax incorrect or no matches found.

**Solution:**
```bash
# Test pattern directly with sg
sg --pattern "fn $NAME($$$ARGS)" --lang rust

# Verify the language is correct
sl ast search "fn main" --lang rust --path src/

# Try simpler pattern
sl ast search "fn " --lang rust --path src/
```

---

## Best Practices

### 1. Always Read Before Editing

```bash
# ✅ Good: Read to get accurate hashes
sl read src/main.rs --lines 5:10
# Review output, note the hashes
sl edit src/main.rs 5#MQ 10#HW "new content"

# ❌ Avoid: Guessing hashes
sl edit src/main.rs 5#ZZ 10#XX "content"  # Hash mismatch!
```

### 2. Use Line Ranges to Reduce Output

```bash
# ✅ Good: Only read the lines you need
sl read src/main.rs --lines 50:100

# ❌ Avoid: Reading huge files to get one hash
sl read src/main.rs  # Could be thousands of lines!
```

### 3. Verify Changes with Diff Output

```bash
# The edit command prints unified diff
sl edit src/main.rs 5#MQ 10#HW "content"

# Output shows:
# --- src/main.rs
# +++ src/main.rs
# @@ -5,6 +5,1 @@
# -    old line 5
# -    old line 6
# ...
# +new content

# Review the diff before accepting
```

### 4. Keep Backups Initially

```bash
# ✅ Good: Keep .bak file during development
sl edit src/main.rs 5#MQ "content"
# Creates src/main.rs.bak (original state)

# ❌ Avoid: Skipping backups early in development
sl edit src/main.rs 5#MQ "content" --no-backup
```

### 5. Test Edits Before Large Changes

```bash
# ✅ Good: Edit one small section, test, then continue
sl edit src/main.rs 5#MQ "first change"
cargo test  # Verify compilation
sl edit src/main.rs 10#VR "second change"
cargo test  # Verify again

# ❌ Avoid: Making many edits without testing
sl edit src/main.rs 5#MQ "change1"
sl edit src/main.rs 10#VR "change2"
sl edit src/main.rs 15#ZP "change3"
cargo test  # Now multiple things could be broken!
```

### 6. Use Semantic Search for Refactoring

```bash
# ✅ Good: Find all function definitions to refactor
sl ast search "fn $NAME($$$ARGS)" --lang rust --path src/

# Then update each one (they're likely similar)

# ❌ Avoid: Manual line-by-line search
grep -n "fn " src/*.rs  # Error-prone, misses context
```

### 7. Combine LSP with Edit for Precision

```bash
# ✅ Good: Use LSP to find symbol, then edit with hash
sl lsp goto-def src/main.rs 10 5  # Shows: src/lib.rs:20:3
sl read src/lib.rs --lines 20:25
sl edit src/lib.rs 20#ZP "updated implementation"

# This is more precise than text search
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (file not found, invalid args, etc.) |
| 124 | Operation timed out (AST/LSP query exceeded timeout) |
| 127 | External binary not found (sg, language server) |

---

## Environment Variables

Currently no environment variables are used. The following are planned for future versions:

- `SOLON_TIMEOUT` — Default timeout for AST/LSP queries (seconds, proposed)
- `SOLON_CHUNK_SIZE` — Default chunk size for large file reads (proposed)
- `SOLON_LSP_DEBUG` — Enable LSP protocol logging (debug builds only, proposed)

---

## Integration with Claude Code

### Using Skills in Claude Code

```javascript
// In Claude Code context, skills are available:

// Read with hashes
const content = await skills.hashline_read("src/main.rs")

// Edit with hash references
const diff = await skills.hashline_edit("src/main.rs", "5#MQ", "10#HW", "new content")

// Search AST
const matches = await skills.ast_search("fn $NAME($$$ARGS)", { lang: "rust" })

// Replace AST
const preview = await skills.ast_replace("fn main() {}", "fn main() -> Result<()> {}", { lang: "rust" })

// LSP queries
const diags = await skills.lsp_tools("diagnostics", "src/main.rs")
const def = await skills.lsp_tools("goto-def", "src/main.rs", 10, 5)
```

### Privacy & Security

The plugin includes safety hooks that:
- **Block sensitive files:** `.env`, `.aws/`, `.ssh/`, `*.pem`, `secrets/`
- **Respect workspace boundaries:** Only accessible files within your workspace

These hooks run automatically and prevent accidental exposure of secrets.

---

## Performance Tips

### Reading Large Files

For files >10 MB, use line ranges:

```bash
# ✅ Good: Read in chunks
sl read huge_file.rs --lines 1:500
sl read huge_file.rs --lines 501:1000
# ... continue

# ❌ Avoid: Reading entire huge file
sl read huge_file.rs  # Slow, lots of output
```

### AST Search on Large Codebases

For large projects, narrow the search:

```bash
# ✅ Good: Limit to relevant directory
sl ast search "fn $NAME" --path src/core/

# Or limit results
sl ast search "TODO" --max-results 10

# ❌ Avoid: Searching entire codebase with broad patterns
sl ast search "." --path /  # Will timeout!
```

### LSP Queries

LSP servers cold-start slowly:

```bash
# ✅ Good: First query is slow, but subsequent queries are fast
sl lsp diagnostics src/main.rs  # ~500ms (cold start)
sl lsp goto-def src/main.rs 10 5  # ~100ms (warm server)

# Future: Daemon mode will keep server warm
```

---

## Getting Help

### View Help

```bash
sl --help
sl read --help
sl edit --help
sl ast --help
sl lsp --help
```

### Check Version

```bash
sl --version
```

### Report Issues

- GitHub: https://github.com/solon-dev/solon/issues
- Include: OS, version, command that failed, error message

---

**Last Updated:** 2026-03-20
**Guide Version:** 1.1
