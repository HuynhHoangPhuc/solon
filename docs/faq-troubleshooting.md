# Solon: FAQ & Troubleshooting

## Frequently Asked Questions

### General Questions

#### Q: What is Solon?
**A:** Solon is a Rust CLI + Claude Code plugin that provides three core capabilities:
1. **Hashline** — Edit files via hash-validated line references (prevents off-by-one errors)
2. **AST** — Semantic code search/replace via ast-grep integration
3. **LSP** — Language server queries (diagnostics, goto-def, references, hover)

#### Q: Why use hashes instead of line numbers?
**A:** Line numbers shift as you edit. Hashes are content-based and don't change, preventing accidental edits to wrong lines if file changes between read and edit operations.

Example:
```
Read file:
5#MQ|    let x = 5;
6#VR|    let y = 10;

Someone adds a line at line 3, file changes:
3#XX|    // new line
4#YY|
5#ZZ|    let x = 5;   ← Now at line 5, but hash changed!
6#AA|    let y = 10;

If you tried to edit with old hashes:
sl edit file 5#MQ 6#VR "content"
Error: Hash validation failed at line 5

You must re-read to get new hashes.
```

#### Q: Is Solon production-ready?
**A:** Yes. Version 0.1.0 is production-ready:
- 27 unit tests + 11 integration tests (all passing)
- Tested on Linux, macOS, Windows
- Atomic writes prevent data loss
- Comprehensive error handling

#### Q: Can I use Solon outside Claude Code?
**A:** Yes! Solon is a standalone CLI. The Claude Code plugin is optional integration.

```bash
# Use directly from terminal
sl read src/main.rs
sl edit src/main.rs 5#MQ "new content"
sl ast search "fn main" --lang rust
sl lsp diagnostics src/main.rs
```

#### Q: Does Solon modify files?
**A:** Only the `sl edit` command modifies files. Commands like `sl read`, `sl ast`, and `sl lsp` are read-only.

---

### Installation & Setup

#### Q: How do I install Solon?
**A:** Quick start:

```bash
git clone https://github.com/solon-dev/solon.git
cd solon
./scripts/install.sh
```

Or build from source:
```bash
cargo build --release
cp target/release/sl ~/.local/bin/
```

#### Q: What if `./scripts/install.sh` fails?
**A:** Common causes and solutions:

1. **Binary download fails:**
   - Check internet connection
   - Verify GitHub is accessible
   - Try manual download: `https://github.com/solon-dev/solon/releases/latest`

2. **No write permission to `~/.local/bin/`:**
   - Create directory: `mkdir -p ~/.local/bin`
   - Verify PATH includes it: `echo $PATH | grep .local`

3. **Checksum mismatch:**
   - Indicates download corruption; try again
   - Or build from source instead

#### Q: Do I need to install `sg` (ast-grep)?
**A:** Only if you plan to use `sl ast` commands. Solon will error clearly if you try to use AST without `sg` installed.

```bash
# Install if needed
cargo install ast-grep

# Verify
which sg
```

#### Q: What language servers do I need?
**A:** Only install servers for languages you plan to use:

```bash
# Rust
cargo install rust-analyzer

# TypeScript/JavaScript
npm install -g typescript-language-server

# Python
pip install python-lsp-server

# List: https://microsoft.github.io/language-server-protocol/implementors/servers/
```

---

### Usage Questions

#### Q: Why do I get "Hash validation failed"?
**A:** The file changed since you read it. Solution:

```bash
# Original read
sl read src/main.rs --lines 5:10
# 5#MQ|old content

# File modified by something else (another editor, git, etc.)
# Now hash is different

# Solution: Re-read to get new hash
sl read src/main.rs --lines 5:5
# 5#XX|new content (different hash!)

# Use new hash
sl edit src/main.rs 5#XX "your content"
```

#### Q: Can I edit multiple files at once?
**A:** Not atomically. Each `sl edit` command modifies one file. For multi-file changes:

```bash
# Edit file 1
sl edit src/main.rs 5#MQ "change1"

# Edit file 2
sl edit src/lib.rs 10#VR "change2"

# If something fails mid-way, manually restore from .bak files
```

Future versions may support multi-file atomic edits.

#### Q: How do I handle merge conflicts?
**A:** Solon doesn't know about git. Use git to resolve conflicts, then use Solon:

```bash
# Resolve conflict in editor or with git tools
git add conflicted_file.rs

# Now use Solon to refine the resolved version
sl read conflicted_file.rs
sl edit conflicted_file.rs ...
```

#### Q: Can I undo edits?
**A:** Yes, via `.bak` file:

```bash
# After edit, if something went wrong
cp src/main.rs.bak src/main.rs

# Or use git
git checkout src/main.rs
```

#### Q: What if I need to edit the same file multiple times?
**A:** Re-read after each edit to get updated hashes:

```bash
# Edit 1
sl read src/main.rs --lines 5:10
sl edit src/main.rs 5#MQ "change1"

# Re-read to get new hashes
sl read src/main.rs --lines 5:10
# Hashes may have changed!

# Edit 2 with new hashes
sl edit src/main.rs 5#XX "change2"
```

---

### AST Questions

#### Q: What's the pattern syntax?
**A:** ast-grep patterns use `$NAME` and `$$$ARGS` placeholders:

```
$NAME       — Single node (function name, variable, etc.)
$$$ARGS     — Multiple nodes (parameters, arguments, etc.)
```

Examples:
```
fn $NAME($$$ARGS) { $$$BODY }           # Function definitions
const $VAR = $VALUE;                    # Variable declarations
match $EXPR { $$$CASES }                # Match expressions
```

Full syntax: https://ast-grep.github.io/

#### Q: How do I test a pattern?
**A:** Use `sg` directly:

```bash
# Test pattern
sg --pattern "fn $NAME($$$ARGS)" --lang rust --json src/

# If it works, use in Solon
sl ast search "fn $NAME($$$ARGS)" --lang rust --path src/
```

#### Q: Why does my pattern return no results?
**A:** Common causes:

1. **Wrong pattern syntax** — Check ast-grep docs
2. **Wrong language** — Use `--lang rust` not `--language rust`
3. **No matches exist** — Pattern is correct but no code matches
4. **Timeout** — Large codebases may timeout; increase with `--timeout 60`

#### Q: Can I use regex in patterns?
**A:** No. ast-grep patterns are semantic, not regex. Use `sg` with `--regex` if needed.

#### Q: Will `sl ast replace` modify my files?
**A:** No. Current version shows a **preview only**. No files are modified.

Future versions will have `--apply` flag to write changes.

---

### LSP Questions

#### Q: Which language servers are supported?
**A:** Any server following LSP spec v3.17:

```
Rust           → rust-analyzer
TypeScript     → typescript-language-server
JavaScript     → typescript-language-server
Python         → pylsp (python-lsp-server)
Go             → gopls
C/C++          → clangd
Java           → eclipse-jdt-ls
Ruby           → solargraph
PHP            → php-language-server
... and many more
```

#### Q: Why is the first LSP query slow?
**A:** Server cold-start. Solution:

```bash
# First query ~500ms (server startup)
sl lsp diagnostics src/main.rs

# Subsequent queries ~100ms (warm server)
sl lsp hover src/main.rs 10 5
```

Future: Daemon mode will keep server warm between queries.

#### Q: Can I specify which LSP server to use?
**A:** Currently auto-detected from file extension. Future versions may allow explicit selection.

#### Q: Why does LSP return empty results?
**A:** Common causes:

1. **Symbol not resolvable** — May be from external crate, macro-generated, etc.
2. **Server doesn't support operation** — Not all servers support all LSP features
3. **Position is wrong** — Ensure line/col are accurate

#### Q: How do I know if a language is supported?
**A:** Try it:

```bash
sl lsp diagnostics file.xyz
# If server found, it's supported. If not, install it.
```

---

### Plugin (Claude Code) Questions

#### Q: How do I install the Claude Code plugin?
**A:** After installing `sl` binary:

```bash
# The plugin is already included in the repo
# Copy to Claude Code's plugin directory
cp -r .claude-plugin ~/.claude-plugin/solon/
```

Or use the install script which handles this.

#### Q: Can I use skills without the plugin?
**A:** Yes, use the CLI directly:

```bash
# Instead of skill call
sl read src/main.rs

# Skills are just wrappers
```

#### Q: What do the safety hooks do?
**A:** Two hooks protect your data:

1. **Privacy Hook** — Blocks sensitive files (.env, .aws/, .ssh/, *.pem, secrets/)
2. **Scout Hook** — Respects Claude Code's file visibility rules

Both run automatically before any read/write.

#### Q: Can I disable safety hooks?
**A:** Not recommended, but you can:

Edit `.claude-plugin/plugin.json` and remove hooks, or modify hook files.

---

### Performance Questions

#### Q: How fast is Solon?
**A:** Benchmarks (approximate):

| Operation | Time |
|-----------|------|
| Binary startup | ~30ms |
| Read 1 MB file | ~50ms |
| Hash validation | <0.1ms per line |
| Edit + diff | ~10ms |
| AST search | 200-1000ms (dominated by `sg`) |
| LSP diagnostics (cold) | ~500ms |
| LSP diagnostics (warm) | ~100ms |

#### Q: How large a file can I edit?
**A:** Solon loads entire file into memory. Tested up to 100 MB successfully.

For files >1 GB, use `--chunk-size` to limit output:

```bash
sl read huge_file.rs --lines 1:1000 --chunk-size 500
```

#### Q: Why is AST search slow?
**A:** It spawns the `sg` binary, which has startup overhead. Typical timeline:

```
User command          : 0ms
Solon startup         : 10ms
Spawn sg process      : 50ms
sg searches file      : 500ms
Parse JSON results    : 5ms
Format output         : 10ms
Total                 : ~575ms
```

Most time is `sg` search. Use `--path` to narrow search scope.

---

### Security Questions

#### Q: Is Solon safe to use?
**A:** Yes. Solon is 100% safe Rust with:
- No `unsafe` code
- No code execution (no eval, exec, etc.)
- Input validation on all user input
- Atomic writes prevent data corruption
- Backup files for disaster recovery

#### Q: Can Solon read my `.env` file?
**A:** No. The privacy hook blocks it:

```bash
sl read .env
# Error: This file is protected for privacy/security reasons.
```

Blocked patterns:
- `.env`, `.env.*`
- `.aws/`, `.ssh/`
- `*.pem`, `*.key`
- `secrets/`, `credentials/`
- `.git/`

#### Q: Can I override the privacy hook?
**A:** Not recommended. Privacy hook exists to prevent accidental secret leaks.

If you must, edit the hook or use the CLI directly:

```bash
# NOT RECOMMENDED - don't do this in production!
solon_binary_path=$(which sl)
$solon_binary_path read .env
```

#### Q: Does Solon phone home or send data?
**A:** No. Solon is entirely local. No network calls except:
- Initial `./scripts/install.sh` downloads binary from GitHub
- Language servers make network requests (LSP servers may, Solon doesn't)

#### Q: What about file permissions?
**A:** Solon respects filesystem permissions:

```bash
# If you don't have read permission, it fails
sl read /root/secret.txt
# Error: Permission denied

# If you don't have write permission, edit fails
sl edit /root/secret.txt 5#MQ "content"
# Error: Permission denied
```

---

## Troubleshooting Guide

### Problem: `sl: command not found`

**Symptoms:**
```
$ sl read file.rs
bash: sl: command not found
```

**Diagnosis:**
- Binary not installed or not in PATH

**Solutions:**

1. **Verify installation:**
   ```bash
   which sl
   # Should show path like /home/user/.local/bin/sl
   ```

2. **If not found, reinstall:**
   ```bash
   ./scripts/install.sh
   ```

3. **Add to PATH if needed:**
   ```bash
   export PATH="$PATH:$HOME/.local/bin"
   # Add to ~/.bashrc or ~/.zshrc to persist
   ```

4. **Check shell is using updated PATH:**
   ```bash
   # Restart shell or source profile
   bash
   # or
   source ~/.bashrc
   ```

---

### Problem: `Hash validation failed at line N`

**Symptoms:**
```
$ sl edit src/main.rs 5#MQ "content"
Error: Hash validation failed at line 5 (file changed?)
```

**Diagnosis:**
- File was modified since you last read it
- Hash at line 5 no longer matches `MQ`

**Solutions:**

1. **Re-read the file:**
   ```bash
   sl read src/main.rs --lines 5:5
   # Check the new hash
   ```

2. **Use the new hash:**
   ```bash
   # Assuming new hash is XX
   sl edit src/main.rs 5#XX "content"
   ```

3. **Merge changes if needed:**
   ```bash
   # Use git/editor to merge changes from both sides
   # Then re-read to get hashes
   ```

---

### Problem: `ast-grep not found in PATH`

**Symptoms:**
```
$ sl ast search "fn main" --lang rust
Error: ast-grep not found in PATH. Install via: cargo install ast-grep
```

**Diagnosis:**
- `sg` binary not installed

**Solutions:**

1. **Install ast-grep:**
   ```bash
   cargo install ast-grep
   ```

2. **Verify installation:**
   ```bash
   which sg
   sg --version
   ```

3. **If not in PATH:**
   ```bash
   # Check where it was installed
   find ~/.cargo -name sg -type f

   # Add to PATH
   export PATH="$PATH:$HOME/.cargo/bin"
   ```

---

### Problem: `Language server not found`

**Symptoms:**
```
$ sl lsp diagnostics src/main.rs
Error: Language server not found for .rs files. Install rust-analyzer
```

**Diagnosis:**
- Language server for file type not installed

**Solutions:**

1. **Install appropriate server:**
   ```bash
   # For Rust
   cargo install rust-analyzer

   # For TypeScript
   npm install -g typescript-language-server

   # For Python
   pip install python-lsp-server
   ```

2. **Verify installation:**
   ```bash
   which rust-analyzer
   rust-analyzer --version
   ```

3. **If installed but not found:**
   ```bash
   # Check PATH includes installation directory
   echo $PATH

   # Add if needed (varies by installation method)
   export PATH="$PATH:/path/to/server/bin"
   ```

---

### Problem: `Operation timed out after 30s`

**Symptoms:**
```
$ sl ast search "some_pattern" --lang rust --path /huge/codebase
Error: Operation timed out after 30s
```

**Diagnosis:**
- Search took longer than 30 second timeout
- Likely on large codebase or complex pattern

**Solutions:**

1. **Increase timeout:**
   ```bash
   sl ast search "pattern" --lang rust --path src/ --timeout 60
   ```

2. **Narrow search scope:**
   ```bash
   # Instead of searching entire codebase
   sl ast search "pattern" --lang rust --path src/core/

   # Search specific file
   sl ast search "pattern" --lang rust --path src/main.rs
   ```

3. **Simplify pattern:**
   ```bash
   # Complex pattern
   sl ast search "fn $NAME($$$ARGS) -> $TYPE { $$$BODY }"

   # Simpler pattern
   sl ast search "fn $NAME"
   ```

---

### Problem: `Invalid range 'X:Y': expected format N:M or N:`

**Symptoms:**
```
$ sl read src/main.rs --lines 5-10
Error: Invalid range '5-10': expected format N:M or N:
```

**Diagnosis:**
- Wrong range syntax (used `-` instead of `:`)

**Solutions:**

1. **Use correct syntax:**
   ```bash
   # Wrong
   sl read file --lines 5-10

   # Correct
   sl read file --lines 5:10
   ```

2. **Valid formats:**
   ```bash
   sl read file --lines 5:10    # Lines 5 to 10
   sl read file --lines 5:      # Lines 5 to EOF
   sl read file --lines :10     # Lines 1 to 10
   ```

---

### Problem: `Permission denied` when editing

**Symptoms:**
```
$ sl edit src/main.rs 5#MQ "content"
Error: Permission denied
```

**Diagnosis:**
- No write permission on file

**Solutions:**

1. **Check permissions:**
   ```bash
   ls -la src/main.rs
   # Look for 'w' in permissions
   ```

2. **Add write permission:**
   ```bash
   chmod u+w src/main.rs
   ```

3. **Check directory permissions:**
   ```bash
   # Parent directory must be writable (for .bak file)
   ls -la src/
   chmod u+w src/
   ```

---

### Problem: `.bak` file not created

**Symptoms:**
```
$ sl edit src/main.rs 5#MQ "content"
# Edit succeeds, but no src/main.rs.bak created
```

**Diagnosis:**
- `--no-backup` flag was used, or backup creation failed

**Solutions:**

1. **Don't skip backup during development:**
   ```bash
   # This creates .bak
   sl edit src/main.rs 5#MQ "content"

   # This skips .bak (advanced users only)
   sl edit src/main.rs 5#MQ "content" --no-backup
   ```

2. **Use git if backup fails:**
   ```bash
   git add src/main.rs  # Commit before edit
   sl edit src/main.rs 5#MQ "content"
   git diff src/main.rs  # View changes
   ```

---

### Problem: Large file causes memory issues

**Symptoms:**
```
$ sl read huge_file.rs
# Process hangs or crashes
```

**Diagnosis:**
- File too large to load entirely, or output too large

**Solutions:**

1. **Read in chunks:**
   ```bash
   # Instead of reading entire file
   sl read huge_file.rs

   # Read a line range
   sl read huge_file.rs --lines 1:1000
   sl read huge_file.rs --lines 1001:2000
   ```

2. **Limit output:**
   ```bash
   sl read huge_file.rs --chunk-size 100
   ```

3. **Use line range directly:**
   ```bash
   # If you know what you need
   sl read huge_file.rs --lines 5000:5100
   ```

---

### Problem: Binary file detected

**Symptoms:**
```
$ sl read image.png
Error: Binary file detected
```

**Diagnosis:**
- File is binary (image, executable, archive, etc.)

**Solutions:**

1. **Only use with text files:**
   ```bash
   sl read image.png      # ❌ Binary
   sl read src/main.rs    # ✅ Text
   ```

2. **Check file type:**
   ```bash
   file image.png
   # Should output something like "ASCII text" for Solon to work
   ```

---

### Problem: Pattern matches but replacement is empty

**Symptoms:**
```
$ sl ast replace "fn $NAME" "async fn $NAME" --lang rust
# No output, no results
```

**Diagnosis:**
- Pattern syntax or replacement may be incorrect
- No actual matches in the search path

**Solutions:**

1. **Test pattern separately:**
   ```bash
   sl ast search "fn $NAME" --lang rust --path src/
   # Should show matches
   ```

2. **Check replacement syntax:**
   ```bash
   # Captures must match between pattern and replacement
   # Pattern: fn $NAME($$$ARGS)
   # Replacement: async fn $NAME($$$ARGS)  # ✅ $NAME matches

   # Wrong
   # Replacement: async fn $OTHERNAME($$$ARGS)  # ❌ $OTHERNAME not in pattern
   ```

---

### Problem: LSP query returns empty

**Symptoms:**
```
$ sl lsp goto-def src/main.rs 10 5
# No output
```

**Diagnosis:**
- Symbol at position not resolvable
- Server doesn't support this operation
- Position might be slightly off

**Solutions:**

1. **Verify position:**
   ```bash
   # Read the file to see exact position
   sl read src/main.rs --lines 10:10
   # Count characters to column 5
   ```

2. **Try diagnostics first:**
   ```bash
   sl lsp diagnostics src/main.rs
   # If this works, server is running
   ```

3. **Check if symbol is resolvable:**
   ```bash
   # Hover should work for resolvable symbols
   sl lsp hover src/main.rs 10 5
   # If empty, symbol may not be resolvable
   ```

4. **Try nearby positions:**
   ```bash
   # Maybe column 5 is in whitespace
   sl lsp goto-def src/main.rs 10 1   # Try column 1
   sl lsp goto-def src/main.rs 10 10  # Try column 10
   ```

---

### Problem: Solon crashes or panics

**Symptoms:**
```
$ sl read src/main.rs
thread 'main' panicked at ...
```

**Diagnosis:**
- Unexpected input or edge case triggered a panic

**Solutions:**

1. **Report the issue:**
   ```bash
   # Include:
   # - Command that failed
   # - Error message
   # - File (if safe to share)
   # - OS and version

   # GitHub: https://github.com/solon-dev/solon/issues
   ```

2. **Workaround:**
   - Try similar command with different arguments
   - Use different file or edit approach

---

## Getting Help

### Resources

- **Documentation:** `./docs/` directory in repo
- **API Reference:** `./docs/api-reference.md`
- **User Guide:** `./docs/user-guide.md`
- **Source Code:** `./src/` directory (Rust)

### Community

- **GitHub Issues:** https://github.com/solon-dev/solon/issues
- **Discussions:** GitHub Discussions (if enabled)

### Debug Information

When reporting issues, include:

```bash
# Version
sl --version

# OS and architecture
uname -a

# Which command failed
# Full error message
# Minimal example to reproduce

# Optional: Enable debug logging (if available)
export RUST_LOG=debug
sl read file.rs
```

---

**Last Updated:** 2026-03-13
**Version:** 1.0
