---
name: sl:lsp-tools
description: LSP-powered diagnostics, goto-definition, references, and hover info
---

# LSP Tools

Use `sl lsp` for language-server-powered code intelligence.

## Commands

### Show diagnostics (errors/warnings)
```bash
sl lsp diagnostics <file>
```
Output: `file:line:col: error: message`

### Go to definition
```bash
sl lsp goto-def <file> <line> <col>
```
Output: `file:line:col` of the definition

### Find all references
```bash
sl lsp references <file> <line> <col>
```
Output: one `file:line:col` per reference

### Show hover info (type signature, docs)
```bash
sl lsp hover <file> <line> <col>
```

## When to Use

| Task | Command |
|------|---------|
| Check for compile errors after editing | `diagnostics` |
| Understand what a symbol refers to | `goto-def` |
| Find all usages before renaming | `references` |
| Learn a function's type signature | `hover` |

## Supported Languages (auto-detected from extension)

| Extension | Server Required |
|-----------|----------------|
| `.rs` | `rust-analyzer` |
| `.ts` `.tsx` `.js` `.jsx` | `typescript-language-server` |
| `.py` | `pyright-langserver` |
| `.go` | `gopls` |
| `.java` | `jdtls` |

## Notes

- Line and column are 1-based
- Server must be installed separately (error message includes install command)
- Fresh connection per invocation (v1 — no persistent server)
