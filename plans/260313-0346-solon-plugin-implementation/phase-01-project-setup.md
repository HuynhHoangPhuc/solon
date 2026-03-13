---
phase: 1
title: "Project Setup"
status: complete
priority: P1
effort: 3h
---

# Phase 1: Project Setup

## Context Links

- [Plan Overview](plan.md)
- [Brainstorm](../reports/brainstorm-260313-0346-solon-plugin-architecture.md)

## Overview

Scaffold the Rust project, Claude Code plugin structure, CI pipeline, and development tooling. This is the foundation everything builds on.

## Requirements

### Functional
- Rust binary compiles and prints version with `sl --version`
- Plugin manifest recognized by Claude Code
- CI builds on push/PR

### Non-Functional
- Binary name: `sl`
- Minimum Rust edition: 2021
- Target platforms: linux-x64, linux-arm64, darwin-x64, darwin-arm64, windows-x64

## Architecture

```
solon/
├── .claude-plugin/
│   └── plugin.json           # Plugin manifest
├── skills/                   # (empty, populated in Phase 6)
├── hooks/
│   └── hooks.json            # (empty array initially)
├── scripts/
│   └── install.sh            # Platform-detect + download
├── src/
│   ├── main.rs               # CLI entry point (clap)
│   ├── cmd/
│   │   └── mod.rs            # Command module (stubs)
│   ├── hashline/
│   │   └── mod.rs            # Hashline module (stubs)
│   ├── ast/
│   │   └── mod.rs            # AST module (stub)
│   └── lsp/
│       └── mod.rs            # LSP module (stub)
├── Cargo.toml
├── .github/
│   └── workflows/
│       └── ci.yml            # Build + test on PR
└── README.md
```

## Related Code Files

### Create
- `Cargo.toml` — workspace config, dependencies
- `src/main.rs` — clap CLI with subcommands (read, edit, ast, lsp)
- `src/cmd/mod.rs` — re-export subcommand modules
- `src/hashline/mod.rs` — module declaration
- `src/ast/mod.rs` — module declaration
- `src/lsp/mod.rs` — module declaration
- `.claude-plugin/plugin.json` — plugin manifest
- `hooks/hooks.json` — empty hooks array
- `scripts/install.sh` — install script skeleton
- `.github/workflows/ci.yml` — CI pipeline

### Modify
- `README.md` — project description, install instructions

## Implementation Steps

1. **Initialize Cargo project**
   ```toml
   [package]
   name = "solon-cli"
   version = "0.1.0"
   edition = "2021"

   [[bin]]
   name = "sl"
   path = "src/main.rs"

   [dependencies]
   clap = { version = "4", features = ["derive"] }
   anyhow = "1"
   ```

2. **Create `src/main.rs`** with clap derive API
   - Subcommands: `Read`, `Edit`, `Ast`, `Lsp`
   - Each subcommand prints "not yet implemented" stub
   - `--version` flag

3. **Create module stubs** — `cmd/mod.rs`, `hashline/mod.rs`, `ast/mod.rs`, `lsp/mod.rs`

4. **Create plugin manifest** `.claude-plugin/plugin.json`
   ```json
   {
     "name": "solon",
     "version": "0.1.0",
     "description": "Hashline read/edit, AST-grep, LSP tools for Claude Code",
     "skills": [],
     "hooks": "hooks/hooks.json"
   }
   ```

5. **Create `hooks/hooks.json`** — empty array `[]`

6. **Create `scripts/install.sh`**
   - Detect OS + arch
   - Download binary from GitHub Releases
   - Place in `~/.local/bin/` or `~/.solon/bin/`
   - Add to PATH if needed

7. **Create `.github/workflows/ci.yml`**
   - Trigger on push/PR to main
   - Matrix: ubuntu-latest, macos-latest, windows-latest
   - Steps: checkout, install Rust, `cargo build`, `cargo test`, `cargo clippy`

8. **Update README.md** — brief project description

9. **Verify**: `cargo build` succeeds, `sl --version` prints version

## Todo List

- [ ] Create Cargo.toml with dependencies
- [ ] Create src/main.rs with clap subcommands
- [ ] Create module stubs (cmd, hashline, ast, lsp)
- [ ] Create .claude-plugin/plugin.json
- [ ] Create hooks/hooks.json
- [ ] Create scripts/install.sh
- [ ] Create .github/workflows/ci.yml
- [ ] Update README.md
- [ ] Verify cargo build passes

## Success Criteria

- `cargo build` succeeds
- `sl --version` prints `solon-cli 0.1.0`
- `sl read`, `sl edit`, `sl ast`, `sl lsp` print stub messages
- CI workflow triggers on push

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Plugin manifest format changes | Medium | Pin to Claude Code v1.0.33+ API |
| Cross-platform build issues | Low | CI matrix catches early |
