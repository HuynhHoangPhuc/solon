# Solon: API Reference

Complete CLI command documentation for all `sl` commands.

---

## Command: `sl read`

Read a file with hashline annotations.

### Signature
```
sl read <FILE> [OPTIONS]
```

### Arguments & Options

| Argument | Type | Description |
|----------|------|-------------|
| `FILE` | Path | File path (required) |
| `--lines <RANGE>` | String | Line range: `N:M`, `N:`, or `:M` |
| `--chunk-size <SIZE>` | Integer | Max lines per chunk (default: 200) |

### Output Format

**Success:**
```
LINE#CID|CONTENT
1#ZP|fn main() {
2#MQ|    println!("hello");
3#HW|}
```

**Error:** `Error: File not found: /path/to/file`

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (file not found, invalid range, etc.) |

### Examples
```bash
# Entire file
sl read src/main.rs

# Lines 5-20
sl read src/main.rs --lines 5:20

# From line 10 to EOF
sl read src/main.rs --lines 10:

# With smaller chunks
sl read src/main.rs --chunk-size 50
```

---

## Command: `sl edit`

Edit a file using hash-validated line references.

### Signature
```
sl edit <FILE> <START_REF> [END_REF] [CONTENT] [OPTIONS]
```

### Arguments & Options

| Argument | Type | Description |
|----------|------|-------------|
| `FILE` | Path | File to edit |
| `START_REF` | String | Start line: `N#CID` (e.g., `5#MQ`) |
| `END_REF` | String | End line (for range ops) |
| `CONTENT` | String | Content to insert/replace |
| `--after` | Flag | Insert after line |
| `--before` | Flag | Insert before line |
| `--delete` | Flag | Delete line(s) |
| `--stdin` | Flag | Read content from stdin |
| `--no-backup` | Flag | Skip `.bak` backup |

### Operation Modes

**Replace (default):**
```bash
sl edit <FILE> <START#CID> <END#CID> <CONTENT>
```
Replace lines START through END.

**Insert After:**
```bash
sl edit <FILE> <REF#CID> <CONTENT> --after
```

**Insert Before:**
```bash
sl edit <FILE> <REF#CID> <CONTENT> --before
```

**Delete:**
```bash
sl edit <FILE> <START#CID> [END#CID] --delete
```

### Output Format

**Success:** Unified diff showing changes + creates `FILE.bak`

**Error:** `Error: Hash validation failed at line 5 (file changed?)`

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (hash mismatch, file not found, etc.) |

### Examples
```bash
# Replace lines 5-10
sl edit src/main.rs 5#MQ 10#HW "    let sum = 15;"

# Insert after line 5
sl edit src/main.rs 5#MQ "    let z = 20;" --after

# Delete line 5
sl edit src/main.rs 5#MQ --delete

# From stdin
echo "new content" | sl edit src/main.rs 5#MQ --stdin

# Skip backup
sl edit src/main.rs 5#MQ "content" --no-backup
```

---

## Command: `sl ast search`

Semantic code search using ast-grep patterns.

### Signature
```
sl ast search <PATTERN> [OPTIONS]
```

### Arguments & Options

| Argument | Type | Default | Description |
|----------|------|---------|-------------|
| `PATTERN` | String | — | ast-grep pattern |
| `--lang <LANG>` | String | Auto-detect | Language (rust, typescript, python, etc.) |
| `--path <PATH>` | Path | `.` | Directory/file to search |
| `--json` | Flag | Off | Output raw JSON |
| `--max-results <N>` | Integer | 50 | Max matches |
| `--timeout <SECS>` | Integer | 30 | Timeout in seconds |

### Output Format

**Success:**
```
file/path.rs:LINE:COL - MATCHED_TEXT
    context line
    MATCHED_LINE
    context line
```

With `--json`, outputs JSON array of match objects.

**Error:** `Error: ast-grep not found in PATH`

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success (may have 0 matches) |
| 1 | Invalid pattern or ast-grep error |
| 124 | Timeout |
| 127 | ast-grep not found |

### Examples
```bash
# Search function definitions
sl ast search "fn $NAME($$$ARGS)" --lang rust --path src/

# TypeScript arrow functions
sl ast search "const $NAME = ($$$ARGS) =>" --lang typescript

# JSON output
sl ast search "TODO" --lang rust --json

# Limit results
sl ast search "fn " --lang rust --max-results 5

# Custom timeout
sl ast search "pattern" --lang rust --timeout 60
```

---

## Command: `sl ast replace`

Semantic code replacement using ast-grep patterns.

### Signature
```
sl ast replace <PATTERN> <REPLACEMENT> [OPTIONS]
```

### Arguments & Options

| Argument | Type | Description |
|----------|------|-------------|
| `PATTERN` | String | ast-grep pattern to match |
| `REPLACEMENT` | String | Replacement (supports `$NAME` captures) |
| `--lang <LANG>` | String | Language |
| `--path <PATH>` | Path | Directory/file to replace in |
| `--timeout <SECS>` | Integer | Timeout (default: 30) |

### Output Format

**Success:** Preview of all replacements (no files modified):
```
file/path.rs:LINE:COL
- ORIGINAL_TEXT
+ REPLACEMENT_TEXT
```

**Note:** Current version is **preview only**. Future: `--apply` flag for writes.

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Invalid pattern/replacement |
| 124 | Timeout |
| 127 | ast-grep not found |

### Examples
```bash
# Make functions async
sl ast replace "fn $NAME($$$ARGS)" "async fn $NAME($$$ARGS)" --lang rust

# Add mut to variables
sl ast replace "let $VAR = $VALUE;" "let mut $VAR = $VALUE;" --lang rust

# Replace logging
sl ast replace "console.log($$$ARGS)" "console.error($$$ARGS)" --lang typescript
```

---

## Command: `sl lsp diagnostics`

Show errors and warnings in a file.

### Signature
```
sl lsp diagnostics <FILE>
```

### Output Format
```
FILE:LINE:COL [LEVEL] MESSAGE

src/main.rs:10:5 [error] expected identifier
src/main.rs:15:10 [warning] unused variable `x`
```

### Exit Codes
| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Language server error |
| 127 | Language server not found |

### Example
```bash
sl lsp diagnostics src/main.rs
```

---

## Command: `sl lsp goto-def`

Jump to definition of symbol at position.

### Signature
```
sl lsp goto-def <FILE> <LINE> <COL>
```

### Output Format
```
FILE:LINE:COL - CONTEXT_LINE
    definition code
```

### Example
```bash
# Line/column are 1-based
sl lsp goto-def src/main.rs 10 5
```

---

## Command: `sl lsp references`

Find all references to symbol at position.

### Signature
```
sl lsp references <FILE> <LINE> <COL>
```

### Output Format
```
FILE:LINE:COL - CONTEXT_LINE
```

Multiple locations for all uses of symbol.

### Example
```bash
sl lsp references src/main.rs 10 5
```

---

## Command: `sl lsp hover`

Show type info, docstring, and hover details.

### Signature
```
sl lsp hover <FILE> <LINE> <COL>
```

### Output Format
```
TYPE_INFO
DOCSTRING
```

Markdown formatting may be included.

### Example
```bash
sl lsp hover src/main.rs 10 5
```

---

## Workflow Commands

### Plan Management
```bash
sl plan scaffold --slug <name> --mode fast|hard|parallel|two
sl plan resolve [--session ID] [--branch BRANCH]
sl plan validate <dir>
```

### Task Management
```bash
sl task hydrate <dir>
sl task sync <dir> --phases 1,2
```

### Workflow Status
```bash
sl workflow status <dir> [--detail]
```

### Report Indexing
```bash
sl report index <dir>
```

---

## Common Patterns

### Hash Format
Lines are printed as: `LINE#CID|CONTENT`
- `LINE`: 1-based line number
- `CID`: 2-character content hash (uppercase letters)
- `CONTENT`: Exact line content

### Range Syntax
- `5:10` — Lines 5 to 10 (inclusive)
- `5:` — Line 5 to EOF
- `:10` — Lines 1 to 10

### Line Reference Format
- `5#MQ` — Line 5 with hash MQ
- Must match current file state (hash validation)

### Escape Sequences
In edit content: `\n` (newline), `\t` (tab), `\\` (backslash)

---

## Notes

- All commands are **atomic** — no partial edits
- **Backups:** `FILE.bak` created by edit (skip with `--no-backup`)
- **Line endings:** Original style (LF/CRLF/CR) preserved
- **BOM:** Preserved if present
- **LSP positions:** 1-based (line 1, column 1 = first char)
- **Timeouts:** AST commands default 30s to prevent hangs

---

**Last Updated:** 2026-03-20
**API Version:** 1.1
