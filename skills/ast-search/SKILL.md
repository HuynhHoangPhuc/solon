---
name: ast-search
description: Semantic code search using ast-grep tree-sitter patterns
---

# AST Search

Use `sl ast search` for semantic code search — finds patterns by structure, not text.

## Usage

```bash
sl ast search "<pattern>" --lang <lang> [--path <dir>]
sl ast search "<pattern>" --lang <lang> --json      # raw JSON output
sl ast search "<pattern>" --lang <lang> --max-results 100
```

## Pattern Syntax

- `$NAME` — matches any single AST node, captures as NAME
- `$$$ARGS` — matches zero or more nodes (variadic)
- Literal code matches exactly

## Common Patterns by Language

### Rust
```bash
sl ast search "fn $NAME($$$ARGS) -> $RET" --lang rust
sl ast search "println!($$$)" --lang rust
sl ast search "unwrap()" --lang rust
```

### TypeScript / JavaScript
```bash
sl ast search "console.log($$$)" --lang typescript
sl ast search "async function $NAME($$$ARGS)" --lang typescript
sl ast search "import $NAME from '$PATH'" --lang typescript
```

### Python
```bash
sl ast search "def $NAME($$$ARGS):" --lang python
sl ast search "raise $ERR" --lang python
```

## Supported Languages

rust, typescript, javascript, python, go, java, c, cpp, html, css, and 15+ more.

## Notes

- Requires `sg` binary (auto-downloaded on first use)
- Default: searches current directory, max 50 results, 30s timeout
- Use `--path src/` to scope searches
