# Solon: Code Standards & Codebase Structure

## Codebase Overview

**Workspace:** 3 Rust crates + root binary
**Total Lines:** ~2,200 LOC (Rust, excluding plugins)
**Binary Size:** ~2.5 MB (optimized release, all subsystems)
**Test Coverage:** 27 unit + 11 integration tests
**Compilation Time:** ~35s (clean build)
**Plugins:** 2 (solon-cli, solon-core)

---

## Rust Code Standards

### Naming Conventions

**Files & Modules**
- Use `snake_case` for file names (Rust convention)
- Match module hierarchy to filesystem structure
- Example: `src/cmd/read.rs` → `mod read { ... }`

**Functions & Methods**
- `snake_case` for function names
- Prefix with `is_`, `has_`, `get_` for clarity
- Async functions: no special prefix

**Variables & Constants**
- `snake_case` for local variables and struct fields
- `SCREAMING_SNAKE_CASE` for module-level constants
- Single-letter vars allowed only in loops: `for i in 0..10`

**Types & Traits**
- `PascalCase` for struct, enum, trait names
- Derive macros: `#[derive(Debug, Clone, Copy)]` order: Debug, Clone, Copy, Default, etc.

### Code Organization

#### Directory Structure (Workspace)

```
.
├── Cargo.toml                  # Workspace definition
├── src/main.rs                 # Root binary entry point
├── crates/
│   ├── solon-common/           # Shared types & utilities
│   │   └── src/
│   ├── solon-cli/              # File operations (hashline, ast, lsp)
│   │   └── src/
│   │       ├── cmd/            # read, edit, ast, lsp commands
│   │       ├── hashline/       # Core hashline protocol
│   │       ├── ast/            # AST-grep integration
│   │       └── lsp/            # LSP client
│   └── solon-core/             # Workflow & orchestration
│       └── src/
│           └── cmd/            # plan, task, workflow, report
├── plugins/
│   ├── solon-cli/              # Plugin wrapper (5 skills)
│   │   └── .claude-plugin/
│   └── solon-core/             # Plugin wrapper (5 skills + hooks)
│       ├── .claude-plugin/
│       └── hooks/
│           └── hooks.json      # Lifecycle matchers (20 hooks → sl hook)
└── .claude-plugin/marketplace.json  # Plugin marketplace registration
```

**Workspace Crates:**
- **solon-common**: Shared types, errors, utilities
- **solon-cli**: File operations (read, edit, ast, lsp)
- **solon-core**: Workflow operations (plan, task, workflow, report)
- **Root binary** (`sl`): Dispatches to crates

#### Module Boundaries

- **`cmd/`** — Public CLI interface; delegates to subsystems
- **`hashline/`** — Self-contained line hashing & editing
- **`ast/`** — Wraps external `sg` binary; formats results
- **`lsp/`** — Standalone LSP client; manages server lifecycle

**No Cross-Subsystem Dependencies:**
- `hashline` doesn't import from `ast` or `lsp`
- `ast` and `lsp` are independent

### Error Handling

#### Pattern: `Result<T>` with `anyhow`

```rust
use anyhow::{Result, Context, bail};

pub fn read_file(path: &Path) -> Result<String> {
    fs::read_to_string(path)
        .context("Failed to read file")
}

pub fn validate(x: usize) -> Result<()> {
    if x < 100 {
        bail!("Value too small: {}", x);
    }
    Ok(())
}
```

**Rules:**
- Use `bail!()` for early returns with error message
- Use `.context()` to add context to propagated errors
- Never `.unwrap()` or `.expect()` in library code
- Main can use `.unwrap()` for final error reporting

#### Error Messages

- **Clear & Actionable:** `"File not found: /path/to/file"`
- **No Jargon:** Avoid `"IO error code 13"` → use `"Permission denied"`
- **Include Context:** `"Hash validation failed at line 5 (file changed?)"`
- **Suggest Fix:** `"ast-grep not found. Install: cargo install ast-grep"`

### Testing

#### Unit Test Pattern

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_description_of_behavior() {
        // Arrange
        let input = "test data";

        // Act
        let result = function_under_test(input);

        // Assert
        assert_eq!(result, expected_value);
    }

    #[test]
    fn test_error_case() {
        let result = function_under_test("invalid");
        assert!(result.is_err());
    }
}
```

**Rules:**
- One test per behavior (not per function)
- Use descriptive test names: `test_hash_determinism_for_same_line()`
- Test happy path + error cases
- No external dependencies in unit tests (use tempfile for I/O)

#### Integration Tests

```
tests/
├── integration/
│   ├── read_test.rs     # Full file read tests
│   └── edit_test.rs     # Full edit operation tests
└── fixtures/
    └── sample.rs        # Test data files
```

**Rules:**
- Use `tempfile::NamedTempFile` for file I/O tests
- Cleanup files automatically (RAII)
- Test realistic workflows, not just unit behaviors
- Name: `tests/integration/<feature>_test.rs`

### Code Style

#### Line Length
- **Soft limit:** 100 characters
- **Hard limit:** 120 characters
- Break long lines at logical boundaries (before operators)

#### Whitespace & Formatting
- Run `cargo fmt` before commit (enforced in CI)
- No trailing whitespace
- Blank lines: max 1 between items, 2 between sections

#### Comments

**Style: Rust documentation comments**

```rust
/// Compute a 2-character content ID for a line.
///
/// # Arguments
/// * `line` - The line content to hash
/// * `line_number` - Line number (used for blank lines)
///
/// # Returns
/// A 2-byte array representing the CID
pub fn compute_cid(line: &str, line_number: usize) -> [u8; 2] {
    // ...
}

/// Returns true if line is blank or contains only punctuation
fn classify_line(line: &str) -> bool { ... }
```

**Rules:**
- Use `///` for public functions (auto-docs)
- Use `//` for internal logic explanation
- Avoid obvious comments: ❌ `let x = 5; // set x to 5`
- Explain *why*, not *what*: ✅ `// Seed with line number for blank lines to differentiate adjacent blanks`

### Imports

**Organization:**
```rust
// 1. Standard library
use std::fs;
use std::io::{self, Read};
use std::path::PathBuf;

// 2. External crates
use anyhow::{bail, Context, Result};
use clap::Args;

// 3. Internal modules
use crate::hashline::format::annotate_line;
use crate::hashline::validate::parse_line_ref;
```

**Rules:**
- Group by category (std, external, internal)
- Sort within each group alphabetically
- Use `use crate::module::item` for internal paths
- Avoid `use crate::*` (wildcard imports)

### Security Considerations

#### No Unsafe Code
- Project is 100% safe Rust (no `unsafe` blocks)
- All dependencies must be reviewed before addition
- Deny unsafe in `lib.rs` if applicable

#### User Input Validation
- Always validate file paths before reading/writing
- Check that edit operations stay within file bounds
- Validate line refs format: `\d+#[A-Z]{2}`

#### Permissions
- Respect filesystem permissions (no chmod calls)
- File writes use atomic rename (POSIX safe)
- Backups inherit original file permissions

---

## Plugin Architecture (2 Plugins)

### Marketplace Registration

**File:** `.claude-plugin/marketplace.json`
- Registers both plugins in a single marketplace
- solon-cli: file operations (5 skills)
- solon-core: workflow operations (5 skills + agents + hooks)

### solon-cli Plugin Structure

```
plugins/solon-cli/.claude-plugin/
├── plugin.json                  # Plugin metadata
└── skills/                      # 5 Skills
    ├── hashline-read/          {SKILL.md, index.js}
    ├── hashline-edit/
    ├── ast-search/
    ├── ast-replace/
    └── lsp-tools/
```

### solon-core Plugin Structure

```
plugins/solon-core/.claude-plugin/
├── plugin.json                  # Plugin metadata
├── skills/                      # 5 Workflow skills
├── agents/                      # 9 Agent definitions
├── hooks/
│   └── hooks.json              # Lifecycle matchers (20 hooks → sl hook)
└── scripts/
    └── install.sh              # Installation script
```

### Skill Implementation

**File:** `skills/{skill-name}/index.js`

```javascript
/**
 * Hashline Read Skill
 * Executes: sl read <file> [--lines START:END] [--chunk-size N]
 */

module.exports = {
  name: 'hashline-read',
  description: 'Read file with hashline annotations',

  async execute(input) {
    const { file, lines, chunkSize } = input;

    // Validate input
    if (!file) throw new Error('Missing required parameter: file');

    // Build command
    const args = ['read', file];
    if (lines) args.push('--lines', lines);
    if (chunkSize) args.push('--chunk-size', chunkSize);

    // Execute sl binary
    const result = await execSl(args);

    return result;
  }
};
```

**Rules:**
- Each skill must export `{ name, description, execute }`
- `execute(input)` is async
- Throw `Error` for validation failures
- Return parsed/formatted output

### Hooks Subsystem (Rust)

**Location:** `crates/solon-core/src/hooks/` — Built into the `sl` binary

**Build:** `cargo build --workspace` (hooks compile as part of solon-core crate)

**Subcommand Pattern (Clap):**

Each hook handler in `hooks/` exports a `run()` function dispatched via `sl hook <name>`:

```rust
// hooks/privacy_block.rs
pub fn run() -> anyhow::Result<()> {
    let input = crate::hooks::read_hook_input()?;
    // Validate file paths against sensitive patterns
    // Return JSON output on stdout
    Ok(())
}
```

**Rules:**
- 20 subcommands dispatched via `cmd/hook.rs` Clap enum
- Each hook reads JSON from stdin + env vars (CLAUDE_CONTEXT_*, SOLON_*)
- Exit code 0 = allowed/success, non-zero = blocked/error
- Errors output to stderr; results to stdout

### Installation Script

**File:** `scripts/install.sh`

```bash
#!/bin/bash
set -euo pipefail

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map to release binary name
case "$OS-$ARCH" in
  linux-x86_64) BINARY="sl-linux-x64" ;;
  linux-aarch64) BINARY="sl-linux-arm64" ;;
  darwin-x86_64) BINARY="sl-darwin-x64" ;;
  darwin-arm64) BINARY="sl-darwin-arm64" ;;
  *) echo "Unsupported OS/arch: $OS-$ARCH"; exit 1 ;;
esac

# Download & verify
RELEASE_URL="https://github.com/solon-dev/solon/releases/latest/download/$BINARY"
curl -fsSL "$RELEASE_URL" -o ~/.local/bin/sl
chmod +x ~/.local/bin/sl

echo "Installed sl to ~/.local/bin/sl"
```

**Rules:**
- Detect OS and architecture
- Verify checksum after download
- Place binary in PATH location
- Report success/failure clearly

---

## Testing Standards

### Test File Naming

- **Unit:** Inline in source files under `#[cfg(test)]` module
- **Integration:** `tests/integration/<feature>_test.rs`
- **Fixtures:** `tests/fixtures/<data>.rs` or `.txt`

### Test Data

```
tests/fixtures/
├── sample.rs                # Rust code sample (100 lines)
├── utf8-with-bom.txt       # UTF-8 with BOM + CRLF
└── binary.bin              # Binary file test case
```

**Rules:**
- Keep fixtures small (<1 KB)
- Use realistic code snippets
- Test edge cases: empty files, very long lines, unusual encodings

### Coverage Requirements

- **Critical Paths:** >90% coverage (hash, edit, validation)
- **Commands:** >80% coverage (read, edit, ast, lsp)
- **Error Cases:** >70% coverage (edge cases, failures)

**Measure with:**
```bash
cargo tarpaulin --out Html
```

### Continuous Integration

**CI Pipeline (.github/workflows/ci.yml):**

```yaml
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    steps:
      - cargo fmt -- --check
      - cargo clippy -- -D warnings
      - cargo build --release
      - cargo test
```

**Rules:**
- Fail on formatting issues
- Fail on clippy warnings
- Test on all three platforms
- All tests must pass before merge

---

## Documentation Standards

### Code Comments

**Good:**
```rust
// Alphanumeric lines use seed=0 to ensure hash is content-based.
// Blank lines use line_number as seed to differentiate adjacent blanks.
let seed = if classify_line(line) { 0 } else { line_number as u32 };
```

**Bad:**
```rust
// Set seed
let seed = if classify_line(line) { 0 } else { line_number as u32 };
```

### Doc Comments

**Good:**
```rust
/// Compute a 2-character content ID (CID) for a line.
///
/// CIDs use a 16-character alphabet for compact representation.
/// For alphanumeric content, the CID is deterministic (seed=0).
/// For blank/punctuation lines, the CID includes line number (seed=line_num).
///
/// # Arguments
/// * `line` - Line content (string slice)
/// * `line_number` - 1-based line number for seed
///
/// # Returns
/// A 2-byte array `[high, low]` where each byte is from the alphabet
///
/// # Examples
/// ```
/// let cid = compute_cid("fn main() {", 1);
/// assert_eq!(cid[0], b'Z'); // First char from alphabet
/// ```
pub fn compute_cid(line: &str, line_number: usize) -> [u8; 2]
```

**Bad:**
```rust
/// Computes CID
pub fn compute_cid(line: &str, line_number: usize) -> [u8; 2]
```

### README Files

**Locations:**
- `./README.md` — Project overview (minimal, links to docs)
- `./docs/*` — Detailed documentation
- `./.claude-plugin/skills/*/SKILL.md` — Skill documentation

**Format:**
- Brief description (1-2 sentences)
- Quick start (copy-paste example)
- Link to detailed docs
- Installation instructions

---

## Dependency Management

### Allowed Dependencies

**Current Approved:**
- `clap` — CLI parsing (derive macros, excellent)
- `anyhow` — Error handling (minimal, idiomatic)
- `xxhash-rust` — Fast hashing (needed for CID)
- `lsp-types` — LSP protocol types (well-maintained)
- `tokio` — Async runtime (heavy but necessary for LSP)
- `serde`/`serde_json` — JSON (standard, minimal)

**Dev Only:**
- `tempfile` — Test file handling (RAII cleanup)

### Dependency Review Criteria

Before adding a new dependency:

1. **Necessity:** Can we implement it ourselves in <50 LOC?
2. **Maintenance:** Is it actively maintained? Check GitHub stars, recent commits
3. **Size:** Will it bloat the binary significantly?
4. **Security:** Any known vulnerabilities? Check `cargo audit`

**Process:**
```bash
cargo add <crate>
cargo audit  # Check for vulnerabilities
cargo tree --duplicates  # Check for duplication
```

---

## Performance Guidelines

### Profiling

```bash
# Measure startup time
time sl read large_file.rs > /dev/null

# Measure memory usage
/usr/bin/time -v sl edit file.rs 1#ZP 10#MQ "content"

# CPU profiling (requires flamegraph)
cargo flamegraph --bin sl -- read large_file.rs
```

### Optimization Priorities

1. **Startup Latency** — <100ms typical
   - Minimize allocations in main()
   - Lazy load config/data

2. **Memory** — No unnecessary copies
   - Use `&str` instead of `String` where possible
   - Process in chunks for large files

3. **I/O** — Reduce syscalls
   - Batch read/write operations
   - Use buffered readers

### Benchmarks

| Operation | Target | Current | Status |
|-----------|--------|---------|--------|
| Binary startup | <50ms | ~30ms | ✅ Good |
| Read 1 MB file | <500ms | ~50ms | ✅ Excellent |
| Hash validation | <1ms | <0.1ms | ✅ Excellent |
| Edit & diff | <100ms | ~10ms | ✅ Excellent |

---

## Git Workflow

### Commit Message Format

**Pattern:** `type(scope): description`

```
fix(hashline): validate hash before applying edit
feat(lsp): add hover information support
docs(architecture): document LSP subsystem
test(hashline): add edge case for blank lines
chore(deps): update serde to 1.0.200
```

**Types:**
- `feat` — New feature
- `fix` — Bug fix
- `docs` — Documentation only
- `test` — Test additions/updates
- `refactor` — Code reorganization
- `perf` — Performance improvement
- `chore` — Maintenance, dependencies

**Scopes:**
- `hashline`, `ast`, `lsp`, `plugin`, `ci`, `deps`, etc.

**Rules:**
- Lowercase
- Imperative mood: "add" not "added"
- No period at end
- Reference issues: `fixes #123`

### Branch Naming

```
feature/hashline-read-optimization
bugfix/edit-hash-validation
docs/system-architecture
chore/upgrade-dependencies
```

### Pull Requests

**Checklist:**
- [x] Passing tests (all platforms)
- [x] No clippy warnings
- [x] Code formatted (`cargo fmt`)
- [x] Documentation updated
- [x] Breaking changes documented
- [x] Commit messages follow convention

---

## Release Process

### Version Numbering

**Semantic Versioning: MAJOR.MINOR.PATCH**

- `MAJOR`: Breaking API changes
- `MINOR`: New features (backward compatible)
- `PATCH`: Bug fixes (backward compatible)

**Examples:**
- `0.1.0` → Initial release
- `0.2.0` → Add LSP support (new feature)
- `0.2.1` → Fix hash edge case (bug fix)

### Release Checklist

1. **Update version:**
   ```toml
   [package]
   version = "0.2.0"
   ```

2. **Update CHANGELOG.md:**
   ```markdown
   ## [0.2.0] - 2026-03-13
   ### Added
   - LSP diagnostics support
   - Hover information queries
   ```

3. **Commit & tag:**
   ```bash
   git add Cargo.toml docs/ CHANGELOG.md
   git commit -m "chore(release): v0.2.0"
   git tag v0.2.0
   git push origin main --tags
   ```

4. **GitHub Actions auto-builds:**
   - Cross-compiles for all platforms
   - Creates release with binaries + checksums
   - Publishes to GitHub Releases

---

## Code Review Checklist

When reviewing PRs:

- [ ] **Correctness** — Logic is sound, tests pass
- [ ] **Style** — Follows standards (names, formatting, docs)
- [ ] **Performance** — No unnecessary allocations or I/O
- [ ] **Security** — No unsafe code, input validated
- [ ] **Testing** — Adequate coverage of new code
- [ ] **Documentation** — Comments + doc strings are clear
- [ ] **Dependencies** — No unnecessary additions
- [ ] **Breaking Changes** — Documented and version-bumped

---

## Development Environment Setup

### Prerequisites

```bash
# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Required tools
rustup component add rustfmt clippy
cargo install cross  # For cross-compilation
```

### Build & Test

```bash
# Format code
cargo fmt

# Lint
cargo clippy -- -D warnings

# Build
cargo build --release

# Test
cargo test

# Test with coverage
cargo tarpaulin --out Html
```

### IDE Setup

**VS Code:**
```json
{
  "rust-analyzer.checkOnSave.command": "clippy",
  "[rust]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "rust-lang.rust-analyzer"
  }
}
```

---

## Troubleshooting Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Hash validation fails | File changed after read | Expected; re-read file to get new hashes |
| LSP server not found | Server not installed | Install: `cargo install rust-analyzer` |
| ast-grep no matches | Wrong pattern syntax | Check ast-grep docs; test pattern with `sg` directly |
| Edit creates .bak | Backup created | Check `.rs.bak` file for original; delete if not needed |
| Windows CRLF issues | Line ending mismatch | Edit preserves original; verify with `hexdump` |

---

**Last Updated:** 2026-03-17
**Standards Version:** 1.2 (Rust workspace + dual-plugin marketplace)
