# Solon Documentation

Welcome to the Solon documentation hub. This directory contains comprehensive guides for understanding, using, and extending the Solon CLI and Claude Code plugin.

## Quick Navigation

### For New Users
1. **[User Guide](./user-guide.md)** — Installation, quick start, common workflows
2. **[FAQ & Troubleshooting](./faq-troubleshooting.md)** — Common questions and solutions

### For Developers
1. **[Project Overview & PDR](./project-overview-pdr.md)** — Vision, features, requirements, success criteria
2. **[System Architecture](./system-architecture.md)** — Design, data flows, module interactions
3. **[Code Standards](./code-standards.md)** — Naming conventions, testing, development practices
4. **[Codebase Summary](./codebase-summary.md)** — Module breakdown, dependencies, test structure

### For API Integration
1. **[API Reference](./api-reference.md)** — Complete CLI command documentation with examples

---

## Document Overview

| Document | Purpose | Audience | Length |
|----------|---------|----------|--------|
| **[project-overview-pdr.md](./project-overview-pdr.md)** | Project vision, features, requirements | Product managers, stakeholders | ~600 lines |
| **[system-architecture.md](./system-architecture.md)** | System design, subsystems, data flows | Architects, senior developers | ~850 lines |
| **[code-standards.md](./code-standards.md)** | Naming, testing, development workflows | Developers, maintainers | ~750 lines |
| **[codebase-summary.md](./codebase-summary.md)** | Module structure, dependencies, tests | Developers, contributors | ~800 lines |
| **[user-guide.md](./user-guide.md)** | Installation, usage, best practices | End users, Claude Code users | ~900 lines |
| **[api-reference.md](./api-reference.md)** | Command signatures, arguments, output formats | API consumers, integrators | ~1,200 lines |
| **[faq-troubleshooting.md](./faq-troubleshooting.md)** | Common questions, error solutions | Support, users | ~1,200 lines |

**Total:** ~5,400 lines of documentation

---

## Key Concepts

### Hashline Protocol
Lines are identified by 2-character content hashes (CIDs), not line numbers. This prevents edits from landing on wrong lines if file changes between read and edit operations.

**Format:** `LINE#CID|CONTENT`

Example:
```
1#ZP|fn main() {
2#MQ|    println!("hello");
3#HW|}
```

### Four Subsystems
1. **Hashline** — Line identification & editing via hashes
2. **AST-Grep** — Semantic code search/replace
3. **LSP Client** — Language server queries (diagnostics, goto-def, references, hover)
4. **Plugin** — Claude Code integration with safety hooks

### Safety First
- 100% safe Rust (no unsafe code)
- Privacy hook blocks sensitive files (.env, .aws/, .ssh/, etc.)
- Scout hook respects file visibility rules
- Atomic writes prevent data corruption

---

## Quick Start

### Install
```bash
git clone https://github.com/solon-dev/solon.git
cd solon
./scripts/install.sh
```

### Verify
```bash
sl --version
sl read src/main.rs --lines 1:5
```

### Use
```bash
# Read with hashes
sl read src/main.rs

# Edit via hash references
sl edit src/main.rs 5#MQ 10#HW "new content"

# Semantic search
sl ast search "fn $NAME($$$ARGS)" --lang rust

# Language intelligence
sl lsp diagnostics src/main.rs
```

---

## Common Use Cases

### Bug Fix
1. Read file to get line hashes
2. Identify buggy line(s)
3. Edit using hash references
4. Verify with `sl lsp diagnostics`

### Refactoring
1. Semantic search for pattern across codebase
2. Review all matches
3. Edit each occurrence with hash-validated edits
4. Test compilation

### Code Review
1. Run `sl lsp diagnostics` to find issues
2. Use hashes to fix problems precisely
3. Combine with `sl lsp hover` for context

---

## Documentation Standards

All documentation follows these principles:

- **Accuracy**: Information verified against actual codebase
- **Clarity**: Clear explanations with examples
- **Completeness**: All major features documented
- **Maintainability**: Easy to update as project evolves
- **User-Centric**: Organized by user workflows, not implementation details

---

## Keeping Documentation Updated

Documentation should be updated when:

1. **New features added** — Document in appropriate guide + API reference
2. **APIs changed** — Update API reference and examples
3. **Architecture evolved** — Update system-architecture.md
4. **Code standards updated** — Update code-standards.md
5. **Common issues discovered** — Add to FAQ & Troubleshooting

### Process

1. Make code change
2. Update relevant documentation
3. Verify all examples still work
4. Check links in all docs
5. Commit together with code changes

---

## File Organization

```
docs/
├── README.md                    # This file (navigation hub)
├── project-overview-pdr.md      # Vision, features, requirements
├── system-architecture.md       # System design, subsystems, data flows
├── code-standards.md            # Development standards and conventions
├── codebase-summary.md          # Module breakdown, dependencies
├── user-guide.md                # Installation, usage, workflows
├── api-reference.md             # Complete CLI command documentation
└── faq-troubleshooting.md       # Q&A, troubleshooting guides
```

---

## Related Resources

- **Source Code**: `./src/` directory
- **Tests**: `./tests/` directory
- **Plugin**: `./.claude-plugin/` directory
- **CI/CD**: `./.github/workflows/`
- **Git History**: `git log --oneline`

---

## Getting Help

### Questions Answered By

| Question | Document |
|----------|----------|
| "How do I install?" | [User Guide](./user-guide.md) → Installation |
| "What's the API?" | [API Reference](./api-reference.md) |
| "Why am I getting error X?" | [FAQ & Troubleshooting](./faq-troubleshooting.md) |
| "How does it work?" | [System Architecture](./system-architecture.md) |
| "What are the coding standards?" | [Code Standards](./code-standards.md) |
| "What features exist?" | [Project Overview](./project-overview-pdr.md) |
| "Where's the code?" | [Codebase Summary](./codebase-summary.md) |

### Still Stuck?

1. Search relevant documents for your question
2. Check [FAQ & Troubleshooting](./faq-troubleshooting.md)
3. Open an issue on GitHub
4. Consult source code directly

---

## Contributing to Documentation

Documentation contributions are welcome! When contributing:

1. Follow the structure and style of existing docs
2. Verify all code examples work
3. Check for typos and clarity
4. Ensure links are accurate
5. Keep line lengths reasonable (soft: 100, hard: 120)

---

## Document Statistics

| Metric | Value |
|--------|-------|
| Total Documents | 8 |
| Total Lines | ~5,400 |
| Avg. Doc Length | ~675 lines |
| Code Examples | 150+ |
| Links | 200+ |
| Last Updated | 2026-03-13 |

---

## Version

- **Documentation Version:** 1.0
- **Solon Version:** 0.1.7
- **Last Updated:** 2026-03-15

---

**Happy coding with Solon!** 🚀

For questions or feedback, open an issue at: https://github.com/solon-dev/solon/issues
