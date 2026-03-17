# Solon: API Reference

## Overview

Solon exposes 4 main commands via CLI, each with specific arguments and output formats. This document details the exact API contract.

---

## Command: `sl read`

Read a file with hashline annotations.

### Signature

```
sl read <FILE> [OPTIONS]
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `FILE` | Path | Yes | File path (relative or absolute) |

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--lines <RANGE>` | String | None | Line range to read: `N:M`, `N:`, or `:M` |
| `--chunk-size <SIZE>` | Integer | 200 | Maximum lines per output chunk |

### Output Format

**Stdout** (Success):
```
LINE#CID|CONTENT
LINE#CID|CONTENT
...
```

- `LINE`: 1-based line number (integer)
- `CID`: 2-character content ID (uppercase letters A-Z)
- `CONTENT`: Exact line content (may include whitespace, special chars)

**Example:**
```
1#ZP|fn main() {
2#MQ|    println!("hello");
3#HW|}
```

**Stderr** (Error):
```
Error: File not found: /path/to/file
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success |
| 1 | File not found, invalid range, other error |

### Examples

#### Read entire file
```bash
$ sl read src/main.rs
1#ZP|fn main() {
2#MQ|    println!("hello");
3#HW|}
```

#### Read specific line range
```bash
$ sl read src/main.rs --lines 5:10
5#MQ|    let x = 5;
6#VR|    println!("{}", x);
7#RK|}
8#SN|
9#KT|fn helper() {
10#XJ|    // ...
```

#### Read from line to EOF
```bash
$ sl read src/main.rs --lines 100:
100#ZP|// last 50 lines
101#MQ|...
```

#### Read with small chunks
```bash
$ sl read src/main.rs --chunk-size 10
# Outputs in 10-line chunks
```

### Notes

- CID is deterministic: same content always produces same CID
- CID is alphanumeric lines: based on content only (seed=0)
- CID for blank lines: includes line number (seed=line_num)
- Empty lines still get a CID and appear in output
- Binary files: detected and error reported

---

## Command: `sl edit`

Edit a file using hash-validated line references.

### Signature

```
sl edit <FILE> <START_REF> [END_REF] [CONTENT] [OPTIONS]
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `FILE` | Path | Yes | File to edit |
| `START_REF` | String | Yes | Start line ref: `N#CID` (e.g., `5#MQ`) |
| `END_REF` | String | No* | End line ref: `N#CID` (for range ops) |
| `CONTENT` | String | No* | Content to insert/replace |

*Depends on operation mode.

### Options

| Option | Type | Effect | Default |
|--------|------|--------|---------|
| `--after` | Flag | Insert content AFTER START_REF | Off |
| `--before` | Flag | Insert content BEFORE START_REF | Off |
| `--delete` | Flag | Delete START_REF or range | Off |
| `--stdin` | Flag | Read content from stdin | Off |
| `--no-backup` | Flag | Skip `.bak` backup creation | Off |

### Operation Modes

#### Replace (default)
```
sl edit <FILE> <START#CID> <END#CID> <CONTENT>
```
Replace lines START through END with CONTENT.

#### Insert After
```
sl edit <FILE> <REF#CID> <CONTENT> --after
```
Insert CONTENT after the referenced line.

#### Insert Before
```
sl edit <FILE> <REF#CID> <CONTENT> --before
```
Insert CONTENT before the referenced line.

#### Delete
```
sl edit <FILE> <START#CID> [<END#CID>] --delete
```
Delete START line, or range START-END if END provided.

### Output Format

**Stdout** (Success):
Unified diff showing changes:
```
--- FILE
+++ FILE
@@ -START,LENGTH +START,LENGTH @@
-removed line
 context line
+added line
```

Also creates `FILE.bak` (original content) unless `--no-backup` used.

**Stderr** (Error):
```
Error: Hash validation failed at line 5 (file changed?)
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success (file modified) |
| 1 | Hash mismatch, file not found, invalid args |

### Examples

#### Replace a line range
```bash
$ sl read src/main.rs --lines 5:10
5#MQ|    let x = 5;
6#VR|    let y = 10;
7#RK|    println!("{}", x + y);
8#SN|}
9#KT|
10#XJ|fn helper() {

$ sl edit src/main.rs 5#MQ 8#SN "    let sum = 15;
    println!(\"{}\", sum);
}"

# Output: unified diff
--- src/main.rs
+++ src/main.rs
@@ -5,4 +5,2 @@
-    let x = 5;
-    let y = 10;
-    println!("{}", x + y);
-}
+    let sum = 15;
+    println!("{}", sum);
+}

# File modified, src/main.rs.bak created
```

#### Insert line after
```bash
$ sl edit src/main.rs 5#MQ "    let z = 20;" --after

# Output: diff showing insertion
--- src/main.rs
+++ src/main.rs
@@ -5,0 +5,1 @@
+    let z = 20;
```

#### Delete line
```bash
$ sl edit src/main.rs 5#MQ --delete

# Removes line 5
```

#### Read content from stdin
```bash
$ echo "    new_content();" | sl edit src/main.rs 5#MQ --stdin

# Content comes from pipe, not CLI arg
```

#### Skip backup
```bash
$ sl edit src/main.rs 5#MQ "content" --no-backup

# No .bak file created
```

### Notes

- **Hash validation:** Start and end references must match current file state
- **Atomic write:** File updated atomically; if error, original unchanged
- **Line endings:** Original line ending style (LF/CRLF/CR) preserved
- **BOM:** Original BOM preserved if present
- **Escape sequences:** `\n` → newline, `\t` → tab, `\\` → backslash
- **Backup safety:** Always review `.bak` before deleting if uncertain

---

## Command: `sl ast search`

Semantic code search using ast-grep patterns.

### Signature

```
sl ast search <PATTERN> [OPTIONS]
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `PATTERN` | String | Yes | ast-grep pattern (e.g., `fn $NAME($$$ARGS)`) |

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--lang <LANG>` | String | Auto-detect | Language: rust, typescript, python, javascript, java, go, c, cpp, etc. |
| `--path <PATH>` | Path | `.` (current dir) | Directory or file to search |
| `--json` | Flag | Off | Output raw JSON from sg |
| `--max-results <N>` | Integer | 50 | Maximum matches to display |
| `--timeout <SECS>` | Integer | 30 | Search timeout in seconds |

### Output Format

**Stdout** (Success):
```
file/path.rs:LINE:COL - MATCHED_TEXT
    context line 1
    MATCHED_LINE
    context line 2

file/path2.rs:LINE:COL - MATCHED_TEXT
    ...
```

With `--json` flag, outputs raw JSON from `sg`:
```json
[
  {
    "file": "src/main.rs",
    "line": 5,
    "col": 0,
    "text": "fn main() {",
    "context": {
      "before": [],
      "after": ["    println!(\"hello\");"]
    }
  }
]
```

**Stderr** (Error):
```
Error: ast-grep not found in PATH. Install: cargo install ast-grep
Error: Operation timed out after 30s
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success (may have 0 matches) |
| 1 | Invalid pattern, ast-grep error |
| 124 | Timeout |
| 127 | ast-grep not found |

### Examples

#### Search for function definitions
```bash
$ sl ast search "fn $NAME($$$ARGS)" --lang rust --path src/

src/main.rs:5:0 - fn main() {
    fn main() {
    println!("hello");
    }

src/lib.rs:10:0 - fn helper(x: i32) {
    fn helper(x: i32) {
    // implementation
    }
```

#### Search in TypeScript
```bash
$ sl ast search "const $NAME = ($$$ARGS) =>" --lang typescript --path src/

src/utils.ts:3:0 - const getData = (url: string) =>
    const getData = (url: string) =>
    fetch(url).then(r => r.json())
```

#### JSON output for programmatic use
```bash
$ sl ast search "TODO" --lang rust --json

[
  {"file": "src/main.rs", "line": 15, "col": 0, "text": "// TODO: fix this"},
  {"file": "src/lib.rs", "line": 20, "col": 0, "text": "// TODO: optimize"}
]
```

#### Limit results
```bash
$ sl ast search "fn " --lang rust --max-results 5

# Only shows first 5 matches
```

#### Custom timeout
```bash
$ sl ast search "complex_pattern" --lang rust --timeout 60

# Waits up to 60 seconds for results
```

### Notes

- **Pattern syntax:** Specific to ast-grep; not regex
- **Language auto-detection:** If `--lang` omitted, `sg` guesses from file extension
- **Timeout:** Prevents hanging on slow/large searches; default 30s
- **Max results:** Limits output for readability; actual search may find more
- **Column numbers:** 0-based (0 = first character of line)
- **Context:** Shows surrounding lines for context; not in JSON mode

---

## Command: `sl ast replace`

Semantic code replacement using ast-grep patterns.

### Signature

```
sl ast replace <PATTERN> <REPLACEMENT> [OPTIONS]
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `PATTERN` | String | Yes | ast-grep pattern to match |
| `REPLACEMENT` | String | Yes | Replacement text (supports `$NAME` captures) |

### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `--lang <LANG>` | String | Auto-detect | Language |
| `--path <PATH>` | Path | `.` | Directory/file to replace in |
| `--timeout <SECS>` | Integer | 30 | Timeout in seconds |

### Output Format

**Stdout** (Success):
Shows preview of all replacements:
```
file/path.rs:LINE:COL
- ORIGINAL_TEXT
+ REPLACEMENT_TEXT

file/path2.rs:LINE:COL
- ORIGINAL_TEXT
+ REPLACEMENT_TEXT
```

**Note:** This is a **preview only**. No files are modified. (Future: add `--apply` flag for actual writes.)

**Stderr** (Error):
```
Error: Invalid replacement pattern
Error: ast-grep not found in PATH
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success (shows preview) |
| 1 | Invalid pattern/replacement |
| 124 | Timeout |
| 127 | ast-grep not found |

### Examples

#### Replace function signatures
```bash
$ sl ast replace "fn $NAME($$$ARGS)" "async fn $NAME($$$ARGS)" --lang rust --path src/

src/main.rs:5:0
- fn helper(x: i32) {
+ async fn helper(x: i32) {

src/lib.rs:10:0
- fn process(data: &[u8]) {
+ async fn process(data: &[u8]) {
```

#### Replace with capture groups
```bash
$ sl ast replace "let $VAR = $VALUE;" "let mut $VAR = $VALUE;" --lang rust

src/main.rs:5:0
- let x = 5;
+ let mut x = 5;
```

#### Specific language
```bash
$ sl ast replace "console.log($$$ARGS)" "console.error($$$ARGS)" --lang typescript --path src/

src/index.ts:15:0
- console.log("debug", x);
+ console.error("debug", x);
```

### Notes

- **Preview mode:** Current version only shows preview, doesn't write files
- **Captures:** `$NAME` captures are reusable in replacement (e.g., `$NAME` in replacement matches `$NAME` in pattern)
- **No files modified:** Safe to use; preview for review before applying manually
- **Future:** Planned `--apply` flag to auto-write all changes

---

## Command: `sl lsp diagnostics`

Show errors and warnings in a file (LSP diagnostics).

### Signature

```
sl lsp diagnostics <FILE>
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `FILE` | Path | Yes | File to check |

### Output Format

**Stdout** (Success):
```
FILE:LINE:COL [LEVEL] MESSAGE

src/main.rs:10:5 [error] expected identifier, found keyword `let`
src/main.rs:15:10 [warning] unused variable `x`
src/main.rs:20:1 [info] this import is not used
```

Columns and lines are 1-based.

**Stderr** (Error):
```
Error: Language server not found for .rs files. Install rust-analyzer
Error: Failed to initialize language server
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success (may have 0 diagnostics) |
| 1 | Language server error, file not found |
| 127 | Language server binary not found |

### Examples

#### Check Rust file
```bash
$ sl lsp diagnostics src/main.rs

src/main.rs:5:10 [error] cannot find function `foo` in this scope
src/main.rs:15:5 [warning] unused variable: `x`
```

#### Check TypeScript file
```bash
$ sl lsp diagnostics src/index.ts

src/index.ts:10:3 [error] Property 'bar' does not exist on type 'Foo'
```

#### No errors
```bash
$ sl lsp diagnostics src/clean.rs

(empty output means no issues)
```

### Notes

- **Auto-detection:** Language determined from file extension
- **Server auto-start:** LSP server started on demand, may be slow first time
- **Level values:** error, warning, information, hint
- **No edits:** Diagnostics are read-only

---

## Command: `sl lsp goto-def`

Jump to definition of symbol at position.

### Signature

```
sl lsp goto-def <FILE> <LINE> <COL>
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `FILE` | Path | Yes | File path |
| `LINE` | Integer | Yes | Line number (1-based) |
| `COL` | Integer | Yes | Column number (1-based) |

### Output Format

**Stdout** (Success):
```
FILE:LINE:COL - CONTEXT_LINE
    definition code
```

Multiple results if symbol has overloads:
```
file1.rs:10:3 - pub fn method()
    pub fn method() { ... }

file2.rs:20:3 - pub fn method()  // overload
    pub fn method() { ... }
```

**No match:**
```
(empty output - symbol not found or not supported by server)
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success (may have 0 results) |
| 1 | Language server error, file not found |
| 127 | Language server not found |

### Examples

#### Jump to function definition
```bash
$ sl read src/main.rs --lines 10:10
10#ZP|    helper(x);

$ sl lsp goto-def src/main.rs 10 5

src/lib.rs:20:3 - fn helper(x: i32) -> i32 {
    fn helper(x: i32) -> i32 {
    x * 2
    }
```

#### Jump to type definition
```bash
$ sl lsp goto-def src/main.rs 5 20

src/lib.rs:100:0 - pub struct MyType {
    pub struct MyType {
    field1: i32,
    field2: String,
    }
```

### Notes

- **1-based indexing:** Lines and columns are 1-based (user-facing)
- **Position accuracy:** Point at the symbol name or identifier
- **Multi-definition:** May return multiple locations (overloads, traits)
- **Not found:** Returns empty if symbol not resolvable
- **Server support:** Not all servers support all features

---

## Command: `sl lsp references`

Find all references to symbol at position.

### Signature

```
sl lsp references <FILE> <LINE> <COL>
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `FILE` | Path | Yes | File path |
| `LINE` | Integer | Yes | Line number (1-based) |
| `COL` | Integer | Yes | Column number (1-based) |

### Output Format

**Stdout** (Success):
```
FILE:LINE:COL - CONTEXT_LINE
    surrounding code

FILE:LINE:COL - CONTEXT_LINE
    surrounding code
```

**No match:**
```
(empty output)
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success (0+ results) |
| 1 | Language server error |
| 127 | Language server not found |

### Examples

#### Find all uses of a variable
```bash
$ sl lsp references src/main.rs 5 5

src/main.rs:5:5 - let x = 5;
    let x = 5;

src/main.rs:10:10 - println!("{}", x);
    println!("{}", x);

src/main.rs:15:10 - x + 1
    x + 1
```

#### Find all calls to a function
```bash
$ sl lsp references src/main.rs 20 10

src/main.rs:5:5 - helper();
    helper();

src/lib.rs:100:3 - helper();
    helper();
```

### Notes

- **Definition included:** First result is usually the definition itself
- **All scopes:** Finds references across project, not just current file
- **May be slow:** Large projects may take time to search all files
- **Accurate scope:** Respects variable scoping, won't return unrelated symbols

---

## Command: `sl lsp hover`

Show type info and documentation for symbol at position.

### Signature

```
sl lsp hover <FILE> <LINE> <COL>
```

### Arguments

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `FILE` | Path | Yes | File path |
| `LINE` | Integer | Yes | Line number (1-based) |
| `COL` | Integer | Yes | Column number (1-based) |

### Output Format

**Stdout** (Success):
Markdown-formatted information:
```
fn helper(x: i32) -> i32

This function doubles the input value.

# Examples

```rust
let result = helper(5);
assert_eq!(result, 10);
```
```

**No info:**
```
(empty output)
```

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | Success (may be empty) |
| 1 | Language server error |
| 127 | Language server not found |

### Examples

#### Get type info for variable
```bash
$ sl lsp hover src/main.rs 5 5

let x: i32

Assigned at line 5
```

#### Get function signature and docs
```bash
$ sl lsp hover src/main.rs 10 10

fn println!(format_string: &str, ...) -> ()

Macro that prints to stdout with a newline.

# Examples

```rust
println!("hello {}", name);
```
```

#### Get type for expression
```bash
$ sl lsp hover src/main.rs 15 5

Vec<String>

A growable vector of strings.
```

### Notes

- **Markdown:** Output is Markdown; may contain code blocks with syntax highlighting
- **Docstrings:** Includes doc comments from source
- **Type-aware:** Shows inferred types for expressions
- **Server-dependent:** Content varies by language server quality

---

## Error Messages Reference

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `File not found: /path/to/file` | File doesn't exist | Check path, use absolute or relative path from CWD |
| `Hash validation failed at line N (file changed?)` | File was modified since read | Re-read file to get new hashes |
| `ast-grep not found in PATH` | `sg` binary not installed | `cargo install ast-grep` |
| `Language server not found` | Language server not installed | Install appropriate server (rust-analyzer, ts-language-server, etc.) |
| `Operation timed out after Xs` | Query took too long | Increase timeout with `--timeout`, or limit search scope |
| `Invalid range 'X:Y': expected format N:M or N:` | Wrong range format | Use `N:M` (both), `N:` (from N), or `:M` (to M) |
| `Invalid line reference 'N#XX'` | Wrong ref format | Use `N#CID` where CID is 2 uppercase letters |
| `Binary file detected` | File is binary, not text | Use text files only |
| `Permission denied` | Don't have read/write access | Check file permissions |

---

## Type Reference

### LineRange

Format: `START:END` where:
- `START` — 1-based line number (omit for start of file)
- `END` — 1-based line number (omit for end of file)

Examples:
- `5:10` — Lines 5 through 10 (inclusive)
- `5:` — Lines 5 through end of file
- `:10` — Lines 1 through 10
- `:` — All lines (equivalent to omitting `--lines`)

### LineRef

Format: `N#CID` where:
- `N` — 1-based line number (integer)
- `#` — Literal hash character
- `CID` — 2-character content ID (uppercase letters)

Examples: `5#MQ`, `10#HW`, `1#ZP`

### Position

Format: `LINE COL` where:
- `LINE` — 1-based line number (integer)
- `COL` — 1-based column number (integer)

Examples:
- `10 5` — Line 10, column 5 (0-indexed in LSP, but CLI uses 1-based for user convenience)

---

## sc CLI Reference

The `sc` binary provides orchestration commands for plan lifecycle management. All commands output JSON to stdout.

### Plan Commands

| Subcommand | Flags | Output |
|------------|-------|--------|
| `sl plan resolve` | `--session <id>` | `{path, resolvedBy, absolute, planFile, phases}` |
| `sl plan scaffold` | `--slug <name>` `--mode fast\|hard\|parallel\|two` `--phases <n>` | `{planDir, mode, filesCreated}` |
| `sl plan validate <dir>` | — | `{valid, planDir, errors, warnings, stats}` |
| `sl plan archive <dir>` | — | `{archived, destination}` |
| `sl plan red-team <dir>` | `--min <n>` `--max <n>` | `{questions: [...]}` |

### Task Commands

| Subcommand | Flags | Output |
|------------|-------|--------|
| `sl task hydrate <dir>` | — | `{planDir, taskCount, tasks: [{phase, title, blockedBy, ...}]}` |
| `sl task sync <dir>` | `--phases <n,n,...>` | `{filesModified, checkboxesUpdated, details}` |

### Workflow Commands

| Subcommand | Flags | Output |
|------------|-------|--------|
| `sl workflow status <dir>` | `--detail` | `{status, progress, phases: {total, completed, inProgress, pending}}` |

### Report Commands

| Subcommand | Flags | Output |
|------------|-------|--------|
| `sl report index <dir>` | — | `{planDir, reports: [{file, type, date}], count}` |

### Resolution Strategy

`sl plan resolve` uses a cascading strategy:
1. **session** — reads `/tmp/sl-session-{id}.json` for `activePlan`
2. **branch** — extracts slug from current git branch, scans plans dir for match

Order is configurable in `.claude/.sl.json` via `plan.resolution.order`.

---

## Rate Limiting & Quotas

No rate limiting is enforced. Solon is a local CLI tool with no server-side limits.

---

## Backward Compatibility

**Version 0.1.0** — All APIs are stable and backward-compatible.

Future major versions may introduce breaking changes, which will be documented in the changelog.

---

**Last Updated:** 2026-03-13
**API Version:** 1.0
