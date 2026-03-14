# Changelog

All notable changes to solon-cli are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [0.1.0] - 2026-03-14

### Added

#### Hashline Read/Edit (`sl read`, `sl edit`)
- Line-addressed file reading with xxHash32 content IDs (CIDs) per line
- `sl read <file>` — output file content with `<lineno>#<CID>` prefix on every line
- `sl read <file> --lines <start>:<end>` — range-limited read
- `sl read <file> --chunk-size <n>` — paginated reads for large files
- `sl edit <file> <start-ref> <end-ref> <new-content>` — replace line range validated by CID
- `sl edit ... --after` — insert content after a referenced line
- `sl edit ... --delete` — delete a referenced line range
- `sl edit ... --stdin` — accept replacement content from stdin
- CID mismatch detection prevents stale-reference edits

#### AST-grep Integration (`sl ast`)
- `sl ast search <pattern> --lang <lang>` — semantic pattern search over source files
- `sl ast search ... --path <dir>` — scope search to a directory subtree
- `sl ast replace <pattern> <replacement> --lang <lang>` — dry-run structural replacement preview
- Auto-downloads `sg` binary on first use; no manual installation required
- Supports all languages covered by ast-grep (Rust, TypeScript, Python, Go, and more)

#### LSP Client (`sl lsp`)
- `sl lsp diagnostics <file>` — fetch errors/warnings from the language server
- `sl lsp goto-def <file> <line> <col>` — resolve definition location
- `sl lsp references <file> <line> <col>` — list all reference locations
- `sl lsp hover <file> <line> <col>` — retrieve hover/documentation info
- Communicates with any stdio JSON-RPC language server (rust-analyzer, typescript-language-server, etc.)

#### Claude Code Plugin System
- **5 Skills** — reusable Claude Code skill modules installable into `.claude/skills/`
- **20 Go Hooks** — compiled hook binaries covering:
  - Token management: preemptive compaction, per-tool output truncation
  - Semantic compression: 20-40% context reduction without information loss
  - Quality enforcement: todo enforcement, comment slop detection
  - Wisdom accumulation: post-session insight extraction and replay
  - Intent gate: user-prompt classification to block off-topic requests
  - Context recovery: structured context reconstruction after compaction

#### Cross-Platform Support
- Pre-built binaries for Linux (x86_64, aarch64), macOS (x86_64, Apple Silicon), Windows (x86_64)
- Both `sl` CLI and hook binaries ship for each platform
- Release profile: stripped, LTO, size-optimized (`opt-level = "z"`)

### Dependencies

- `clap 4` — CLI argument parsing with derive macros
- `xxhash-rust 0.8` (xxh32 feature) — fast non-cryptographic content hashing
- `lsp-types 0.97` — LSP protocol type definitions
- `tokio 1` (full) — async runtime for LSP stdio transport
- `serde / serde_json 1` — JSON serialization for LSP messages and hook payloads
- `anyhow 1` — ergonomic error propagation

---

[0.1.0]: https://github.com/solon-dev/solon/releases/tag/v0.1.0
