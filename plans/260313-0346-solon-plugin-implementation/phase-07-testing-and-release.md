---
phase: 7
title: "Testing & Release"
status: complete
priority: P1
effort: 5h
depends_on: [1, 2, 3, 4, 5, 6]
---

# Phase 7: Testing & Release

## Context Links

- [Plan Overview](plan.md)
- All phase files

## Overview

Comprehensive testing, CI/CD pipeline for multi-platform releases, and documentation. Ensures production readiness.

## Requirements

### Functional
- Unit tests for all modules (hashline, ast, lsp)
- Integration tests for CLI commands
- End-to-end test: read → edit → verify
- GitHub Actions release workflow
- Cross-platform binary builds

### Non-Functional
- Code coverage > 80% for hashline module
- Binary size < 15MB per platform
- Release artifacts include checksums
- Changelog in release notes

## Architecture

```
.github/workflows/
├── ci.yml              # Build + test on every PR
└── release.yml         # Build + publish on tag push

tests/
├── unit/
│   ├── hashline_test.rs
│   └── ast_test.rs
├── integration/
│   ├── read_test.rs
│   ├── edit_test.rs
│   ├── ast_test.rs
│   └── lsp_test.rs
└── fixtures/
    ├── sample.rs
    ├── sample.py
    └── sample.ts
```

## Related Code Files

### Create
- `tests/unit/hashline_test.rs` — hash computation, CID generation, canonicalization
- `tests/integration/read_test.rs` — sl read end-to-end
- `tests/integration/edit_test.rs` — sl edit end-to-end (all operations)
- `tests/integration/ast_test.rs` — sl ast search/replace
- `tests/integration/lsp_test.rs` — sl lsp commands (if server available)
- `tests/fixtures/` — sample source files
- `.github/workflows/release.yml` — release pipeline
- `CHANGELOG.md` — release notes

### Modify
- `.github/workflows/ci.yml` — add test + clippy + fmt checks
- `Cargo.toml` — add dev-dependencies for testing

## Implementation Steps

1. **Create test fixtures**
   - `tests/fixtures/sample.rs` — Rust file with known content
   - `tests/fixtures/sample.py` — Python file
   - `tests/fixtures/sample.ts` — TypeScript file
   - `tests/fixtures/crlf.txt` — file with CRLF endings
   - `tests/fixtures/bom.txt` — file with UTF-8 BOM

2. **Write unit tests for hashline**
   - Known input → known CID vectors (golden tests)
   - Seed selection: alphanumeric vs blank lines
   - BOM stripping
   - CRLF → LF normalization
   - Line range parsing
   - Hash validation (match + mismatch)

3. **Write integration tests for read**
   - Build binary, invoke via `Command`
   - `sl read fixtures/sample.rs` → verify output format
   - `sl read fixtures/sample.rs --lines 1:3` → verify line range
   - `sl read nonexistent.rs` → verify error
   - `sl read fixtures/bom.txt` → verify BOM stripped

4. **Write integration tests for edit**
   - Copy fixture to temp dir
   - `sl edit temp/sample.rs 1#XX "new first line"` → verify content changed
   - Range replace, append, prepend, delete
   - Hash mismatch → verify rejection
   - `sl read` after edit → verify new hashes

5. **Write integration tests for ast** (conditional on sg availability)
   - `sl ast search "fn $NAME()" --lang rust --path fixtures/`
   - Verify match output format

6. **Write integration tests for lsp** (conditional on server availability)
   - Mark as `#[ignore]` for CI without servers
   - `sl lsp diagnostics fixtures/sample.rs`

7. **Create `.github/workflows/release.yml`**
   ```yaml
   on:
     push:
       tags: ["v*"]
   jobs:
     build:
       strategy:
         matrix:
           include:
             - os: ubuntu-latest, target: x86_64-unknown-linux-gnu, name: sl-linux-x64
             - os: ubuntu-latest, target: aarch64-unknown-linux-gnu, name: sl-linux-arm64
             - os: macos-latest, target: x86_64-apple-darwin, name: sl-darwin-x64
             - os: macos-latest, target: aarch64-apple-darwin, name: sl-darwin-arm64
             - os: windows-latest, target: x86_64-pc-windows-msvc, name: sl-windows-x64.exe
       steps:
         - checkout
         - install rust + cross
         - cargo build --release --target ${{ matrix.target }}
         - rename binary
         - generate sha256 checksum
         - upload artifact
     release:
       needs: build
       steps:
         - download all artifacts
         - create GitHub Release with checksums
   ```

8. **Update CI workflow**
   - Add: `cargo fmt -- --check`
   - Add: `cargo clippy -- -D warnings`
   - Add: `cargo test`

9. **Create CHANGELOG.md** for v0.1.0

## Todo List

- [ ] Create test fixtures (sample files)
- [ ] Write hashline unit tests (golden vectors)
- [ ] Write read integration tests
- [ ] Write edit integration tests (all operations)
- [ ] Write ast integration tests
- [ ] Write lsp integration tests (ignored in CI)
- [ ] Create release.yml workflow
- [ ] Update ci.yml with fmt + clippy + test
- [ ] Create CHANGELOG.md
- [ ] Test release workflow with test tag
- [ ] Verify binary sizes < 15MB
- [ ] Verify checksums in release artifacts

## Success Criteria

- All unit tests pass
- All integration tests pass (ast/lsp conditional)
- CI passes on all platforms (ubuntu, macos, windows)
- Release workflow produces binaries for 5 targets
- Each binary < 15MB
- SHA256 checksums included in release
- Install script can download from release

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Cross-compilation failures (arm64) | Medium | Use `cross` crate for cross-compilation |
| Flaky LSP tests | Low | Mark as #[ignore], run manually |
| Binary size too large | Low | Strip symbols, use LTO |

## Security Considerations

- Release artifacts include SHA256 checksums
- CI uses pinned action versions (no @latest)
- No secrets in test fixtures
- Release requires tag push (not PR merge)
