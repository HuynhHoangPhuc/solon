# solon-cli

[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Rust 1.70+](https://img.shields.io/badge/rust-1.70+-orange.svg)](https://www.rust-lang.org)

**Solon v0.2.0** — A Rust CLI tool and workflow engine for precise, hash-validated file editing with semantic code search, language server protocol support, and full development workflow orchestration.

## Features

- **Hashline Editing** — Edit files by line reference with xxHash32 CIDs for validation
- **AST-based Search & Replace** — Semantic code search via ast-grep integration
- **Language Server Protocol (LSP)** — Code intelligence (diagnostics, goto-def, hover, references)
- **Claude Code Plugin** — 5 skills + 20 intelligent hooks
- **Token Efficiency Hooks** — Preemptive compaction, per-tool output truncation, semantic compression (20-40% reduction)
- **Agent Quality Hooks** — Intent gate classification, wisdom accumulation, todo enforcement, comment slop detection
- **Workflow Engine** — Full development workflow: brainstorm → plan → cook → test → review via `sc` binary

## Quick Install

```bash
# Clone and build
git clone https://github.com/yourusername/solon.git
cd solon
cargo install --path .

# Verify installation
sl --version
```

## Quick Start

### Read Files
```bash
# Read entire file
sl read src/main.rs

# Read specific lines
sl read src/main.rs --lines 10:20

# Read in chunks (useful for large files)
sl read data.json --chunk-size 50
```

### Edit Files
```bash
# Edit by line hash reference
sl edit src/main.rs 5#MQ 10#HW "fn new_function() {}"

# Append content after a line
sl edit src/main.rs 15#AQ "" "println!(\"new line\");" --after

# Delete lines
sl edit src/main.rs 20#XY 25#ZZ --delete

# Read from stdin
echo "new code" | sl edit src/main.rs 5#MQ --stdin
```

### Search & Replace (AST-based)
```bash
# Search for functions in Rust
sl ast search "fn \$NAME(\$\$\$ARGS)" --lang rust

# Search in specific directory
sl ast search "function \$NAME()" --lang typescript --path src/

# Replace (preview only)
sl ast replace "fn main() {}" "fn main() -> Result<()> {}" --lang rust
```

### Language Server Features
```bash
# Get diagnostics
sl lsp diagnostics src/main.rs

# Goto definition
sl lsp goto-def src/main.rs 25 10

# Find references
sl lsp references src/main.rs 15 5

# Hover information
sl lsp hover src/main.rs 20 0
```

### Workflow Commands
```bash
# Scaffold a new plan
sc plan scaffold --slug my-feature --mode fast

# Extract tasks from phases
sc task hydrate plans/YYMMDD-my-feature

# Check progress
sc workflow status plans/YYMMDD-my-feature
```

## Documentation

Full documentation is available in `/docs`:
- **[User Guide](docs/user-guide.md)** — Detailed CLI usage and examples
- **[API Reference](docs/api-reference.md)** — Command reference and options
- **[System Architecture](docs/system-architecture.md)** — Internal design and subsystems
- **[FAQ & Troubleshooting](docs/faq-troubleshooting.md)** — Common issues and solutions
- **[Code Standards](docs/code-standards.md)** — Development guidelines

## Architecture

Solon consists of 4 integrated subsystems:

1. **Hashline** — Line-ID editing with content validation via xxHash32
2. **AST-grep** — Semantic search/replace for code (auto-downloads `sg` binary)
3. **LSP Client** — Language server protocol integration (stdio JSON-RPC)
4. **Plugin System** — Claude Code skills + safety hooks

## Requirements

- Rust 1.70+
- For AST features: `sg` binary (auto-downloaded on first use)
- For LSP features: Appropriate language server installed (rust-analyzer, typescript-language-server, etc.)

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) file for details.

## Support

For issues, questions, or feature requests, please check [FAQ & Troubleshooting](docs/faq-troubleshooting.md) or open an issue on GitHub.
