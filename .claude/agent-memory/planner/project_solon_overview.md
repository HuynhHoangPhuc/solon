---
name: Solon Project Overview
description: Solon is a Rust sl CLI binary + Claude Code plugin providing hashline read/edit, AST-grep, LSP tools, and safety hooks
type: project
---

Solon = Rust `sl` binary + Claude Code plugin. Key features: hashline read/edit (line-range editing with xxHash32 validation), AST-grep (shell out to sg), LSP integration (on-demand), safety hooks (scout-block, privacy-block).

**Why:** Replaces string-match editing with line-range editing — ~60% success rate vs ~6.7% baseline. Zero MCP overhead (CLI via Bash tool).

**How to apply:** All implementation follows plan at `plans/260313-0346-solon-plugin-implementation/`. 7 phases: setup → hashline core → hashline edit → ast-grep → LSP → plugin integration → testing/release. Phases 4+5 can parallelize after Phase 1.
