# Solon: FAQ & Troubleshooting

## Frequently Asked Questions

### General Questions

#### Q: What is Solon?
**A:** Solon is a Rust CLI + Claude Code plugin providing:
1. **Hashline** — Edit files via hash-validated line references
2. **AST** — Semantic code search/replace via ast-grep
3. **LSP** — Language server queries (diagnostics, goto-def, references, hover)

#### Q: Why use hashes instead of line numbers?
**A:** Line numbers shift as you edit. Hashes are content-based and persist, preventing accidental edits to wrong lines.

#### Q: Is Solon production-ready?
**A:** Yes. Tested on Linux, macOS, Windows with 27 unit + 11 integration tests passing.

#### Q: Can I use Solon outside Claude Code?
**A:** Yes, it's a standalone CLI. Install `sl` and use directly from terminal.

---

### Installation & Setup

#### Q: How do I install Solon?
**A:**
```bash
git clone https://github.com/solon-dev/solon.git
cd solon
./scripts/install.sh
sl --version
```

Or build from source:
```bash
cargo build --release
cp target/release/sl ~/.local/bin/
```

#### Q: What if `./scripts/install.sh` fails?
**A:** Common causes:
1. **No internet** — Check connection, try manual download from GitHub Releases
2. **No write permission** — Create directory: `mkdir -p ~/.local/bin`
3. **Checksum mismatch** — Download corruption; try again or build from source

#### Q: Do I need to install `sg` (ast-grep)?
**A:** Only if using `sl ast` commands. Install: `cargo install ast-grep`

#### Q: What language servers do I need?
**A:** Only for languages you use:
```bash
cargo install rust-analyzer        # Rust
npm install -g typescript-language-server  # TypeScript/JavaScript
pip install python-lsp-server      # Python
```

---

### Usage Questions

#### Q: Why do I get "Hash validation failed"?
**A:** File changed since you read it. Solution:
```bash
sl read src/main.rs --lines 5:5    # Get new hash
# Use new hash in edit command
sl edit src/main.rs 5#XX "content"
```

#### Q: Can I edit multiple files at once?
**A:** No, each `sl edit` modifies one file. Edit files sequentially if needed.

#### Q: How do I undo edits?
**A:**
```bash
# Restore from .bak
cp src/main.rs.bak src/main.rs

# Or use git
git checkout src/main.rs
```

#### Q: What if I need to edit the same file multiple times?
**A:** Re-read after each edit to get updated hashes:
```bash
sl edit src/main.rs 5#MQ "change1"
sl read src/main.rs --lines 5:10   # Re-read for new hashes
sl edit src/main.rs 5#XX "change2" # Use new hash
```

---

### AST Questions

#### Q: What's the pattern syntax?
**A:** ast-grep patterns use `$NAME` (single) and `$$$ARGS` (multiple):
```
fn $NAME($$$ARGS) { $$$BODY }    # Function definitions
const $VAR = $VALUE;             # Variable declarations
```
Full syntax: https://ast-grep.github.io/

#### Q: How do I test a pattern?
**A:**
```bash
sg --pattern "fn $NAME($$$ARGS)" --lang rust --json src/
# If it works, use in Solon:
sl ast search "fn $NAME($$$ARGS)" --lang rust --path src/
```

#### Q: Will `sl ast replace` modify my files?
**A:** No, current version shows **preview only**. Future versions will have `--apply`.

---

### LSP Questions

#### Q: Which language servers are supported?
**A:** Any LSP v3.17 spec compliant server:
- Rust → rust-analyzer
- TypeScript/JavaScript → typescript-language-server
- Python → pylsp
- Go → gopls
- C/C++ → clangd

#### Q: Why is the first LSP query slow?
**A:** Server cold-start (~500ms). Subsequent queries are faster (~100ms).

#### Q: Can I specify which LSP server to use?
**A:** Currently auto-detected from file extension. Explicit selection planned for v2.

---

## Troubleshooting Guide

### Problem: `sl: command not found`

**Solutions:**
1. Verify: `which sl`
2. Reinstall: `./scripts/install.sh`
3. Add to PATH: `export PATH="$PATH:$HOME/.local/bin"`
4. Restart shell: `bash` or `source ~/.bashrc`

---

### Problem: `Hash validation failed at line N`

**Solutions:**
1. Re-read file: `sl read src/main.rs --lines 5:5`
2. Use new hash: `sl edit src/main.rs 5#XX "content"`
3. Merge conflicts with git/editor if needed

---

### Problem: `ast-grep not found in PATH`

**Solutions:**
1. Install: `cargo install ast-grep`
2. Verify: `which sg`
3. Add to PATH: `export PATH="$PATH:$HOME/.cargo/bin"`

---

### Problem: `Language server not found`

**Solutions:**
1. Install appropriate server (rust-analyzer, typescript-language-server, etc.)
2. Verify: `which rust-analyzer`
3. Check PATH if installed

---

### Problem: `Operation timed out after 30s`

**Solutions:**
1. Increase timeout: `sl ast search "pattern" --timeout 60`
2. Narrow scope: `sl ast search "pattern" --path src/core/`
3. Simplify pattern: Use less complex patterns

---

### Problem: `Permission denied` when editing

**Solutions:**
1. Check permissions: `ls -la src/main.rs`
2. Add write: `chmod u+w src/main.rs`
3. Check parent: `chmod u+w src/`

---

### Problem: Large file causes memory issues

**Solutions:**
1. Read in chunks: `sl read huge_file.rs --lines 1:1000`
2. Limit output: `sl read huge_file.rs --chunk-size 100`
3. Use specific range: `sl read huge_file.rs --lines 5000:5100`

---

### Problem: Binary file detected

**Solutions:**
Only use with text files. Check: `file image.png` (should say "ASCII text").

---

### Problem: LSP query returns empty

**Solutions:**
1. Verify position: `sl read src/main.rs --lines 10:10`
2. Try diagnostics: `sl lsp diagnostics src/main.rs`
3. Try hover: `sl lsp hover src/main.rs 10 5`
4. Try nearby positions with different columns

---

### Problem: Solon crashes or panics

**Solutions:**
1. Report issue with:
   - Command that failed
   - Error message
   - File (if safe)
   - OS/version
2. GitHub: https://github.com/solon-dev/solon/issues

---

## Performance

| Operation | Time |
|-----------|------|
| Startup | ~30ms |
| Read 1 MB | ~50ms |
| Hash validation | <0.1ms per line |
| Edit + diff | ~10ms |
| AST search | 200-1000ms |
| LSP cold start | ~500ms |
| LSP warm | ~100ms |

**Max file size:** Tested up to 100 MB. Use `--lines` for larger files.

---

## Security

- **100% safe Rust** — No `unsafe` code
- **Privacy hook** — Blocks `.env`, `.ssh/`, `*.pem`, `secrets/`, `.git/`
- **No code execution** — No eval, exec, dynamic code
- **Input validation** — All user input validated
- **Atomic writes** — Prevent data corruption
- **Backup files** — Disaster recovery with `.bak`

Cannot read: `.env`, `.aws/`, `.ssh/`, `*.pem`, `secrets/`, `.git/`

---

## Getting Help

**Documentation:** `./docs/` in repo
**API Reference:** `./docs/api-reference.md`
**Source Code:** `./src/` (Rust)
**Issues:** https://github.com/solon-dev/solon/issues

When reporting bugs, include:
```bash
sl --version
uname -a
# Command that failed + error message + minimal repro
```

---

**Last Updated:** 2026-03-20
**Version:** 1.1
