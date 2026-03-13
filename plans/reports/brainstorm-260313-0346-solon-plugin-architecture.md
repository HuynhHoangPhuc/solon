# Brainstorm: Solon Plugin Architecture

## Problem Statement

Build a Claude Code plugin ("Solon") that improves agent performance via hashline read/edit, AST-grep, LSP integration, and safety hooks. Must maximize token efficiency and edit reliability while avoiding MCP context bloat.

## Requirements

- **Platform**: Claude Code plugin only (official plugin system)
- **Language**: Rust (`sl` unified CLI binary)
- **Features**: Hashline read/edit, AST-grep, LSP tools, safety hooks
- **Scope**: Feature-rich v1
- **Distribution**: GitHub Releases + install script
- **License**: Apache 2.0

## Key Constraints Discovered

| Constraint | Detail |
|---|---|
| Can't modify Read output | PostToolUse only offers `additionalContext` (doubles tokens) or `updatedMCPToolOutput` (MCP only) |
| CAN modify Edit input | PreToolUse `updatedInput` transforms tool input before execution |
| MCP adds context bloat | Tool definitions always in system prompt (~100-200 tokens/tool) |
| CLI via Bash = zero overhead | Same tool-call count as native tools, no protocol wrapping |

## Evaluated Approaches

### Approach 1: MCP Server (Rejected)
- **Pros**: Standard protocol, auto-discovered tools
- **Cons**: ~800 tokens always in context for tool defs, JSON-RPC overhead per call
- **Verdict**: Rejected — user prioritizes token efficiency

### Approach 2: `sl read` + Native Edit with PreToolUse Hook
- **Pros**: Zero learning curve for Edit, transparent hash validation
- **Cons**: Keeps string-match problem (models fail to reproduce `old_string` exactly), hook complexity
- **Verdict**: Rejected — doesn't solve the fundamental harness problem

### Approach 3: `sl read` + `sl edit` via Bash (Selected)
- **Pros**: Eliminates string reproduction, line-range editing (10x reliability), Rust-native validation, no hooks needed for edit, simpler permissions
- **Cons**: Claude learns new `sl` syntax (trivial via skills)
- **Verdict**: Selected — maximum efficiency + reliability

## Final Solution

### Architecture

```
solon/
├── .claude-plugin/
│   └── plugin.json              # Plugin manifest
├── skills/
│   ├── hashline-read/SKILL.md   # Teach: sl read <file> [--lines N:M]
│   ├── hashline-edit/SKILL.md   # Teach: sl edit <file> <pos> [<end>] <content>
│   ├── ast-search/SKILL.md      # Teach: sl ast search <pattern> [--lang X]
│   ├── ast-replace/SKILL.md     # Teach: sl ast replace <pattern> <replacement>
│   └── lsp-tools/SKILL.md       # Teach: sl lsp diagnostics|goto-def|refs
├── hooks/
│   └── hooks.json               # Safety hooks (scout-block, privacy)
├── agents/                      # Optional specialized agents
├── scripts/
│   └── install.sh               # Platform-detect + download binary
├── src/                         # Rust source
│   ├── main.rs                  # CLI entry (clap)
│   ├── cmd/
│   │   ├── read.rs              # Hashline read
│   │   ├── edit.rs              # Hashline edit (line-range)
│   │   ├── ast.rs               # AST-grep wrapper
│   │   └── lsp.rs               # LSP client
│   ├── hashline/
│   │   ├── hash.rs              # xxHash32, 2-char CID
│   │   ├── format.rs            # LINE#HASH|content format
│   │   └── validate.rs          # Staleness detection
│   ├── ast/
│   │   └── sg.rs                # ast-grep binary invocation
│   └── lsp/
│       └── client.rs            # LSP stdio client
├── Cargo.toml
└── .github/
    └── workflows/
        └── release.yml          # Build binaries per platform
```

### Hashline Format

```
1#ZP|fn main() {
2#VK|    println!("hello");
3#MQ|}
```

- Algorithm: xxHash32 → mask to 0-255 → 2-char CID from `ZPMQVRWSNKTXJBYH` alphabet
- Seed: 0 for lines with alphanumeric content, line_number for blank/punctuation
- Deterministic: same content → same hash

### Edit Model (Line-Range)

```bash
# Single line replace
sl edit src/main.rs 1#ZP "fn start() {"

# Range replace (lines 5-10)
sl edit src/main.rs 5#HH 10#QQ "new content\nline 2\nline 3"

# Append after line
sl edit src/main.rs --after 3#MQ "new line here"

# Prepend before line
sl edit src/main.rs --before 1#ZP "// header comment"
```

### Token Flow

| Step | Tool calls | Tokens overhead |
|---|---|---|
| Read | 1 (Bash: `sl read`) | ~0 extra vs native Read |
| Edit | 1 (Bash: `sl edit`) | ~0 extra vs native Edit |
| AST search | 1 (Bash: `sl ast search`) | ~0 extra |
| LSP check | 1 (Bash: `sl lsp diag`) | ~0 extra |

### Safety Hooks

- **Scout-block**: PreToolUse on Glob/Grep, block overly broad patterns
- **Privacy-block**: PreToolUse on Read/Bash, block sensitive files (.env, credentials)
- **Output truncation**: PostToolUse, limit large outputs

### Distribution

- GitHub Actions: build for linux-x64, linux-arm64, darwin-x64, darwin-arm64, windows-x64
- Install: `curl -fsSL https://raw.githubusercontent.com/.../install.sh | sh`
- Or: `cargo install solon-cli`
- Plugin install: `claude plugin install solon@marketplace`

## Implementation Considerations

- Rust crates: `clap` (CLI), `xxhash-rust` (hashing), `tree-sitter` (AST via ast-grep), `lsp-types` + `tokio` (LSP client)
- ast-grep: either embed as lib or shell out to `sg` binary
- LSP: lazy connection, on-demand start, idle timeout
- File canonicalization: strip BOM, normalize CRLF→LF, restore original format after edit

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Claude ignores skill instructions, uses native Read/Edit | Agent system prompt reinforcement + PreToolUse hook as fallback |
| Hash collisions (xxHash32 mod 256) | 2-char CID + line number = sufficient for edit validation |
| ast-grep binary not installed | Auto-download on first use (like oh-my-openagent) |
| LSP server not available | Graceful degradation, clear error message |
| Large files (100K+ lines) | Chunked output (200 lines/chunk, 64KB limit) |

## Success Metrics

- Edit success rate: >60% (vs ~6.7% baseline for string-match)
- Zero MCP tool definitions in context
- Plugin install < 30 seconds
- Binary size < 15MB per platform

## Unresolved Questions

1. Should `sl edit` support oh-my-pi's full operation set (replace/append/prepend/delete) or start minimal?
2. For ast-grep: embed tree-sitter grammars in binary or download on demand?
3. LSP: support multiple language servers simultaneously or one at a time?
4. Should plugin ship a default agent via `settings.json` to enforce `sl` usage?
