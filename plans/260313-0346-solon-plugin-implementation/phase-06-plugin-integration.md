---
phase: 6
title: "Plugin Integration"
status: complete
priority: P1
effort: 4h
depends_on: [2, 3, 4, 5]
---

# Phase 6: Plugin Integration

## Context Links

- [Plan Overview](plan.md)
- [Claude Code Plugin Docs](https://docs.anthropic.com/en/docs/claude-code/plugins)

## Overview

Create the Claude Code plugin layer: skill files that teach Claude the `sl` commands, safety hooks (scout-block, privacy-block), install script, and plugin manifest. This connects the Rust binary to Claude's workflow.

## Key Insights

- Skills are markdown files — zero runtime overhead, loaded into context only when relevant
- Hooks run as JS scripts via plugin hooks system
- Install script must handle binary download + PATH setup
- Plugin manifest declares skills + hooks

## Requirements

### Functional
- 5 skill files teaching `sl` command usage
- Scout-block hook: prevent overly broad Glob/Grep patterns
- Privacy-block hook: block reading sensitive files (.env, credentials)
- Install script: download correct binary for platform, verify checksum
- Plugin manifest correctly references all skills and hooks

### Non-Functional
- Skill files < 100 lines each (token efficient)
- Hooks must respond in < 100ms
- Install works offline with pre-downloaded binary

## Architecture

```
.claude-plugin/plugin.json
├── skills: [hashline-read, hashline-edit, ast-search, ast-replace, lsp-tools]
├── hooks: hooks/hooks.json
└── install: scripts/install.sh
```

## Related Code Files

### Create
- `skills/hashline-read/SKILL.md` — teaches `sl read`
- `skills/hashline-edit/SKILL.md` — teaches `sl edit`
- `skills/ast-search/SKILL.md` — teaches `sl ast search`
- `skills/ast-replace/SKILL.md` — teaches `sl ast replace`
- `skills/lsp-tools/SKILL.md` — teaches `sl lsp` subcommands
- `hooks/scout-block.cjs` — broad pattern detection hook
- `hooks/privacy-block.cjs` — sensitive file detection hook

### Modify
- `.claude-plugin/plugin.json` — add skills + hooks references
- `hooks/hooks.json` — register hook scripts
- `scripts/install.sh` — finalize download + install logic

## Implementation Steps

1. **Create `skills/hashline-read/SKILL.md`**
   ```markdown
   ---
   name: hashline-read
   description: Read files with hashline annotations for reliable editing
   ---
   # Hashline Read

   Use `sl read` instead of the Read tool for reading code files.

   ## Usage
   - `sl read <file>` — read entire file with line hashes
   - `sl read <file> --lines 5:20` — read specific line range

   ## Output Format
   Each line: `LINE#HASH|CONTENT`
   Example: `1#ZP|fn main() {`

   ## When to Use
   - Always use for code files you plan to edit
   - Use native Read for non-code files (images, PDFs)
   ```

2. **Create `skills/hashline-edit/SKILL.md`**
   - Document all edit operations with examples
   - Emphasize: use hashes from most recent `sl read` output
   - Show: replace, range replace, append, prepend, delete
   - Warn: hash mismatch means re-read needed

3. **Create `skills/ast-search/SKILL.md`**
   - Document pattern syntax (tree-sitter patterns)
   - Common patterns by language
   - `$NAME` for single node, `$$$ARGS` for multiple nodes

4. **Create `skills/ast-replace/SKILL.md`**
   - Replace pattern syntax
   - Preview mode vs apply mode
   - Safety: always preview before applying

5. **Create `skills/lsp-tools/SKILL.md`**
   - All subcommands with examples
   - When to use each (diagnostics after edit, goto-def for understanding, refs for refactoring)

6. **Create `hooks/scout-block.cjs`**
   - PreToolUse hook on Glob, Grep
   - Block patterns like `**/*`, `*`, root-level globs
   - Return `{ blocked: true, reason: "..." }` with suggestions

7. **Create `hooks/privacy-block.cjs`**
   - PreToolUse hook on Read, Bash
   - Block access to: `.env*`, `*credentials*`, `*secret*`, `*.pem`, `*.key`
   - Return prompt for user approval (per existing hook protocol)

8. **Update `hooks/hooks.json`**
   ```json
   [
     {
       "hook": "PreToolUse",
       "tools": ["Glob", "Grep"],
       "script": "hooks/scout-block.cjs"
     },
     {
       "hook": "PreToolUse",
       "tools": ["Read", "Bash"],
       "script": "hooks/privacy-block.cjs"
     }
   ]
   ```

9. **Update `.claude-plugin/plugin.json`**
   - Add all 5 skills
   - Reference hooks.json
   - Set install command

10. **Finalize `scripts/install.sh`**
    - Detect OS (linux/darwin/windows) + arch (x64/arm64)
    - Download from GitHub Releases: `https://github.com/<owner>/solon/releases/latest/download/sl-{os}-{arch}`
    - Verify SHA256 checksum
    - Install to `~/.solon/bin/sl`
    - Add `~/.solon/bin` to PATH (append to .bashrc/.zshrc if not present)
    - Print success message with version

## Todo List

- [ ] Create skills/hashline-read/SKILL.md
- [ ] Create skills/hashline-edit/SKILL.md
- [ ] Create skills/ast-search/SKILL.md
- [ ] Create skills/ast-replace/SKILL.md
- [ ] Create skills/lsp-tools/SKILL.md
- [ ] Create hooks/scout-block.cjs
- [ ] Create hooks/privacy-block.cjs
- [ ] Update hooks/hooks.json
- [ ] Update .claude-plugin/plugin.json
- [ ] Finalize scripts/install.sh
- [ ] Test plugin detection by Claude Code
- [ ] Test hook blocking behavior
- [ ] Test install script on Linux

## Success Criteria

- `claude plugin install .` recognizes plugin
- Skills appear in Claude's context when relevant
- Scout-block prevents `**/*` glob patterns
- Privacy-block prompts for `.env` access
- Install script downloads and installs correct binary

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Claude ignores skills, uses native tools | Medium | Agent system prompt reinforcement; hooks as fallback |
| Hook script errors crash Claude | High | Try-catch in all hook scripts, return passthrough on error |
| Install script fails on exotic OS | Low | Provide `cargo install` fallback |

## Security Considerations

- Hooks must not leak file contents in error messages
- Install script verifies checksums
- Privacy hook blocks sensitive files by default
- Skill files don't contain executable code
