# Changelog

All notable changes to solon are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [0.4.0] - 2026-03-17

### Changed
- Restructured as 2-plugin marketplace (solon-cli + solon-core)
- Migrated all Go code to Rust (single `sl` binary)
- Cargo workspace with 3 crates (solon-common, solon-cli, solon-core)
- Workflow commands now use `sl plan/task/workflow/report` (was `sc`)
- All hooks now use `sl hook <name>` subcommands (was `solon-hooks`)
- CI/CD simplified to Rust-only (removed Go jobs)

### Removed
- Go binaries (`sc`, `solon-hooks`) — replaced by `sl` subcommands
- Root `plugin.json` — replaced by per-plugin manifests
- `notify` hook — dropped to avoid HTTP dependency

---

## [0.1.1] - 2026-03-14

### Added
- Full test coverage: 67 Rust unit tests, 15 Rust integration tests
- CI: test jobs, sg + rust-analyzer install for integration tests
- Release workflow: cross-compiled for 5 targets with SHA256 checksums

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

#### AST-grep Integration (`sl ast`)
- `sl ast search <pattern> --lang <lang>` — semantic pattern search
- `sl ast replace <pattern> <replacement> --lang <lang>` — dry-run structural replacement
- Auto-downloads `sg` binary on first use

#### LSP Client (`sl lsp`)
- `sl lsp diagnostics <file>` — fetch errors/warnings from language server
- `sl lsp goto-def <file> <line> <col>` — resolve definition location
- `sl lsp references <file> <line> <col>` — list all reference locations
- `sl lsp hover <file> <line> <col>` — retrieve hover/documentation info

#### Claude Code Plugin System
- **10 Skills** — reusable Claude Code skill modules across 2 plugins
- **20 Hooks** — built into `sl` binary covering token management, quality enforcement, wisdom accumulation

#### Cross-Platform Support
- Pre-built `sl` binary for Linux (x86_64, aarch64), macOS (x86_64, Apple Silicon), Windows (x86_64)
- Release profile: stripped, LTO, size-optimized (`opt-level = "z"`)

---

[0.4.0]: https://github.com/HuynhHoangPhuc/solon/releases/tag/v0.4.0
[0.1.1]: https://github.com/HuynhHoangPhuc/solon/releases/tag/v0.1.1
[0.1.0]: https://github.com/HuynhHoangPhuc/solon/releases/tag/v0.1.0
