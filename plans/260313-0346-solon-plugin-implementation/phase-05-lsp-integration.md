---
phase: 5
title: "LSP Integration"
status: complete
priority: P2
effort: 5h
depends_on: [1]
---

# Phase 5: LSP Integration

## Context Links

- [Plan Overview](plan.md)
- [LSP Specification](https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/)

## Overview

Implement `sl lsp` — on-demand LSP client that connects to language servers for diagnostics, goto-definition, and find-references. Lazy connection model: start server on first use, cache connection, idle timeout.

## Key Insights

- LSP servers already exist per language — no need to bundle them
- One server at a time (YAGNI: multi-server adds complexity, defer)
- Stdio transport is simplest and most portable
- Idle timeout (60s) prevents orphaned server processes
- Output must be concise — Claude doesn't need full LSP JSON

## Requirements

### Functional
- `sl lsp diagnostics <file>` — show errors/warnings for file
- `sl lsp goto-def <file> <line> <col>` — go to definition
- `sl lsp references <file> <line> <col>` — find all references
- `sl lsp hover <file> <line> <col>` — show hover info (type signature, docs)
- Auto-detect language server from file extension
- Graceful fallback when no server available

### Non-Functional
- Server startup timeout: 10s
- Request timeout: 5s
- Idle shutdown: 60s
- Cache server connection across calls within same session

## Architecture

```
sl lsp <subcommand> <file> [line] [col]
        │
        ▼
  ┌──────────────┐
  │ Detect server │ ← file extension → server command lookup
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Connect/reuse │ ← stdio transport, cached process
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Send request  │ ← LSP JSON-RPC
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Format result │ ← concise file:line:col display
  └──────────────┘
```

### Server Detection Map

| Extension | Server | Command |
|---|---|---|
| .rs | rust-analyzer | `rust-analyzer` |
| .ts/.tsx/.js/.jsx | typescript-language-server | `typescript-language-server --stdio` |
| .py | pyright | `pyright-langserver --stdio` |
| .go | gopls | `gopls serve` |
| .java | jdtls | `jdtls` |

## Related Code Files

### Create
- `src/lsp/client.rs` — LSP stdio client (connect, send, receive)
- `src/lsp/detect.rs` — language server detection from file extension
- `src/lsp/format.rs` — output formatting (diagnostics, locations, hover)
- `src/cmd/lsp.rs` — `sl lsp` command implementation

### Modify
- `Cargo.toml` — add `lsp-types`, `tokio`, `serde_json`
- `src/main.rs` — wire Lsp subcommand
- `src/cmd/mod.rs` — re-export lsp
- `src/lsp/mod.rs` — re-export submodules

## Implementation Steps

1. **Add dependencies**:
   ```toml
   lsp-types = "0.97"
   tokio = { version = "1", features = ["full"] }
   serde = { version = "1", features = ["derive"] }
   serde_json = "1"
   ```

2. **Implement `src/lsp/detect.rs`**
   - `fn detect_server(file_path: &Path) -> Option<ServerConfig>`
   - `struct ServerConfig { command: String, args: Vec<String>, root_detection: Vec<String> }`
   - Detect project root from marker files (Cargo.toml, package.json, go.mod, etc.)

3. **Implement `src/lsp/client.rs`**
   - `struct LspClient` — holds child process handle, stdin/stdout
   - `fn connect(config: &ServerConfig, root: &Path) -> Result<LspClient>`
     - Spawn process with stdio
     - Send `initialize` request
     - Wait for `initialized` response
   - `fn send_request<R: Request>(&mut self, params: R::Params) -> Result<R::Result>`
     - Serialize JSON-RPC, write to stdin
     - Read response from stdout
     - Timeout handling
   - `fn shutdown(&mut self)` — send shutdown + exit
   - Connection caching: use a lock file or socket for cross-invocation reuse
     - **v1 simplification**: no caching, fresh connection per invocation (add caching in v2)

4. **Implement `src/lsp/format.rs`**
   - `fn format_diagnostics(diags: &[Diagnostic]) -> String`
     - `file:line:col: severity: message`
   - `fn format_location(loc: &Location) -> String`
     - `file:line:col`
   - `fn format_hover(hover: &Hover) -> String`
     - Extract markdown content, truncate if long

5. **Implement `src/cmd/lsp.rs`**
   - Subcommands: `diagnostics`, `goto-def`, `references`, `hover`
   - Each: detect server → connect → send request → format → print
   - Error handling: server not found, timeout, server crash

6. **Wire into `main.rs`**

## Todo List

- [ ] Add lsp-types, tokio, serde_json dependencies
- [ ] Implement detect.rs (server detection from file extension)
- [ ] Implement client.rs (stdio LSP client)
- [ ] Implement format.rs (diagnostics, locations, hover formatting)
- [ ] Implement cmd/lsp.rs (subcommands)
- [ ] Wire Lsp command into main.rs
- [ ] Test with rust-analyzer
- [ ] Test with typescript-language-server
- [ ] Test server-not-found error handling
- [ ] Test timeout handling

## Success Criteria

- `sl lsp diagnostics src/main.rs` shows compile errors
- `sl lsp goto-def src/main.rs 10 5` outputs definition location
- `sl lsp references src/main.rs 10 5` lists all references
- Graceful error when server not installed
- Output is concise and parseable

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| LSP server not installed | Medium | Clear error + install instructions per language |
| Server startup slow (jdtls) | Medium | 10s timeout, warn user |
| Connection caching complexity | Low | Skip for v1, fresh connection per call |
| LSP protocol edge cases | Medium | Use lsp-types crate for correct serialization |

## Security Considerations

- Don't execute arbitrary commands — server map is hardcoded
- Validate file paths before sending to server
- Server output is untrusted — sanitize before display
