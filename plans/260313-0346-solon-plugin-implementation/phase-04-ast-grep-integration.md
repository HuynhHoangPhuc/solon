---
phase: 4
title: "AST-grep Integration"
status: complete
priority: P2
effort: 4h
depends_on: [1]
---

# Phase 4: AST-grep Integration

## Context Links

- [Plan Overview](plan.md)
- [ast-grep docs](https://ast-grep.github.io/)

## Overview

Integrate ast-grep (`sg`) for semantic code search and replace. Shell out to the `sg` binary rather than embedding tree-sitter grammars — simpler, smaller binary, always up-to-date grammars.

## Key Insights

- ast-grep supports 25+ languages with tree-sitter
- Shelling out to `sg` avoids embedding ~50MB of grammars
- `sg` must be installed separately — `sl` auto-downloads on first use
- Output should be formatted for Claude consumption (concise, with file:line refs)

## Requirements

### Functional
- `sl ast search <pattern> [--lang <LANG>] [--path <DIR>]` — semantic search
- `sl ast replace <pattern> <replacement> [--lang <LANG>] [--path <DIR>]` — semantic replace
- `sl ast search <pattern> --json` — JSON output for programmatic use
- Auto-download `sg` binary if not found
- Support pattern files via `--pattern-file`

### Non-Functional
- Timeout: 30s default for search operations
- Output limit: 50 matches default (configurable)

## Architecture

```
sl ast search/replace
        │
        ▼
  ┌──────────────┐
  │ Locate sg     │ ← PATH lookup → auto-download if missing
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Build sg args │ ← translate sl args → sg CLI args
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Execute sg    │ ← subprocess with timeout
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Format output │ ← parse sg JSON → concise display
  └──────────────┘
```

## Related Code Files

### Create
- `src/ast/sg.rs` — sg binary management (locate, download, execute)
- `src/ast/format.rs` — output formatting
- `src/cmd/ast.rs` — `sl ast` command implementation

### Modify
- `src/main.rs` — wire Ast subcommand
- `src/cmd/mod.rs` — re-export ast
- `src/ast/mod.rs` — re-export submodules

## Implementation Steps

1. **Implement `src/ast/sg.rs`**
   - `fn find_sg() -> Option<PathBuf>` — check PATH, then `~/.solon/bin/sg`
   - `fn download_sg() -> Result<PathBuf>` — download from ast-grep GitHub releases
     - Detect OS + arch
     - Download to `~/.solon/bin/sg`
     - Make executable
   - `fn run_sg(args: &[&str], timeout: Duration) -> Result<String>`
     - Spawn process, capture stdout/stderr
     - Kill on timeout

2. **Implement `src/ast/format.rs`**
   - `fn format_search_results(json_output: &str, max_results: usize) -> String`
     - Parse sg JSON output
     - Format as: `file:line: matched_code`
     - Truncate at max_results with count summary
   - `fn format_replace_preview(json_output: &str) -> String`
     - Show before/after for each match

3. **Implement `src/cmd/ast.rs`**
   - Subcommands: `search`, `replace`
   - Common args: `--lang`, `--path`, `--json`, `--max-results`
   - Search: `sg run --pattern <P> --lang <L> --json` → format
   - Replace: `sg run --pattern <P> --rewrite <R> --lang <L>` with `--update-all` or `--interactive` flag
   - Handle sg not found: prompt auto-download

4. **Wire into `main.rs`**

5. **Handle edge cases**:
   - sg binary not found and download fails → clear error with manual install instructions
   - No matches → clean "0 matches found" message
   - Pattern syntax error → forward sg error message

## Todo List

- [ ] Implement sg.rs (locate, download, execute)
- [ ] Implement format.rs (search/replace output formatting)
- [ ] Implement cmd/ast.rs (search + replace subcommands)
- [ ] Wire Ast command into main.rs
- [ ] Auto-download logic for sg binary
- [ ] Unit tests for output formatting
- [ ] Integration test: search a known pattern in test files
- [ ] Test sg-not-found error handling

## Success Criteria

- `sl ast search "fn $NAME($$$ARGS)" --lang rust` finds function definitions
- `sl ast replace "println!" "eprintln!" --lang rust --path src/` replaces macros
- Auto-downloads sg if missing
- Output is concise and Claude-friendly
- Timeout prevents hanging

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| sg binary not available for platform | Medium | Support cargo install ast-grep as fallback |
| sg version incompatibility | Low | Pin minimum version, test against it |
| Large codebases slow search | Medium | Default timeout + path scoping |

## Security Considerations

- Downloaded binaries should be checksum-verified
- sg replace modifies files — warn before bulk operations
- Respect .gitignore (sg does this by default)
