---
title: "Solon Plugin Implementation"
description: "Full implementation plan for Solon — a Rust CLI + Claude Code plugin for hashline read/edit, AST-grep, LSP tools, and safety hooks"
status: complete
priority: P1
effort: 32h
branch: main
tags: [rust, claude-code, plugin, hashline, ast-grep, lsp]
created: 2026-03-13
---

# Solon Plugin Implementation Plan

## Context

- [Brainstorm Report](../reports/brainstorm-260313-0346-solon-plugin-architecture.md)
- License: Apache 2.0

## Summary

Solon = Rust `sl` CLI binary + Claude Code plugin. Provides hashline read/edit (line-range editing with content-hash validation), AST-grep semantic search/replace, LSP integration, and safety hooks. Zero MCP overhead — skills teach Claude to call `sl` via Bash.

## Architecture Overview

```
User → Claude Code → Bash("sl read/edit/ast/lsp") → Rust binary
                   → Plugin hooks (safety) → Block/allow tool calls
                   → Plugin skills (docs) → Teach sl syntax
```

## Phases

| # | Phase | Effort | Status | File |
|---|-------|--------|--------|------|
| 1 | Project Setup | 3h | complete | [phase-01-project-setup.md](phase-01-project-setup.md) |
| 2 | Hashline Core | 5h | complete | [phase-02-hashline-core.md](phase-02-hashline-core.md) |
| 3 | Hashline Edit | 6h | complete | [phase-03-hashline-edit.md](phase-03-hashline-edit.md) |
| 4 | AST-grep Integration | 4h | complete | [phase-04-ast-grep-integration.md](phase-04-ast-grep-integration.md) |
| 5 | LSP Integration | 5h | complete | [phase-05-lsp-integration.md](phase-05-lsp-integration.md) |
| 6 | Plugin Integration | 4h | complete | [phase-06-plugin-integration.md](phase-06-plugin-integration.md) |
| 7 | Testing & Release | 5h | complete | [phase-07-testing-and-release.md](phase-07-testing-and-release.md) |

## Dependencies

- Phase 2 depends on Phase 1
- Phase 3 depends on Phase 2
- Phase 4, 5 can run in parallel after Phase 1
- Phase 6 depends on Phases 2-5
- Phase 7 depends on all prior phases

## Key Decisions

1. **CLI via Bash over MCP** — zero context overhead, same tool-call count
2. **Line-range edit over string-match** — eliminates reproduction errors (~60% vs ~6.7% success)
3. **ast-grep: shell out to `sg` binary** — avoids embedding 25 grammars, simpler maintenance
4. **LSP: lazy on-demand connection** — no persistent server process

## Unresolved Questions

1. ast-grep: embed tree-sitter grammars or require `sg` binary? → **Decision: shell out to `sg`** (KISS)
2. LSP: multiple servers simultaneously? → **Decision: one at a time** (YAGNI, add later)
3. Default agent to enforce `sl` usage? → **Decision: defer** (skills sufficient for v1)
4. `sl edit` operation set? → **Decision: replace/append/prepend/delete** (full set, small incremental cost)
