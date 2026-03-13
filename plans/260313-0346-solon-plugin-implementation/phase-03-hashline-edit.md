---
phase: 3
title: "Hashline Edit"
status: complete
priority: P1
effort: 6h
depends_on: [2]
---

# Phase 3: Hashline Edit

## Context Links

- [Plan Overview](plan.md)
- [Phase 2 — Hashline Core](phase-02-hashline-core.md)

## Overview

Implement `sl edit` — line-range editing with hash validation. This is the key value proposition: Claude specifies line numbers + hashes instead of reproducing exact string content, dramatically improving edit reliability.

## Key Insights

- Hash validation catches stale edits (file changed since read)
- Line-range addressing eliminates string-match reproduction errors
- Must preserve original line endings (CRLF/LF) on write-back
- Diff output after edit helps Claude verify the change

## Requirements

### Functional
- **Single line replace**: `sl edit file.rs 1#ZP "new content"`
- **Range replace**: `sl edit file.rs 5#HH 10#QQ "replacement\nlines"`
- **Append after**: `sl edit file.rs --after 3#MQ "new line"`
- **Prepend before**: `sl edit file.rs --before 1#ZP "header"`
- **Delete lines**: `sl edit file.rs --delete 5#HH 10#QQ`
- Hash validation: reject edit if hash doesn't match current file state
- Output: show unified diff of changes after successful edit
- Atomic writes: write to temp file, then rename

### Non-Functional
- File backup before edit (optional `--no-backup` flag)
- Preserve file permissions
- Handle concurrent edits gracefully (file lock or hash check)

## Architecture

```
sl edit <file> [start#HASH] [end#HASH] [--after|--before|--delete] <content>
        │
        ▼
  ┌──────────────┐
  │ Parse args    │ ← determine operation type
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Read + hash   │ ← load file, compute hashes for target lines
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Validate hash │ ← compare provided hash vs computed hash
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Apply edit    │ ← splice lines: replace/insert/delete
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Write back    │ ← atomic write (temp + rename), restore line endings
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Diff output   │ ← show unified diff of changes
  └──────────────┘
```

## Related Code Files

### Create
- `src/hashline/validate.rs` — hash validation logic
- `src/hashline/edit.rs` — line splicing operations
- `src/cmd/edit.rs` — `sl edit` command implementation

### Modify
- `src/main.rs` — wire Edit subcommand
- `src/cmd/mod.rs` — re-export edit
- `src/hashline/mod.rs` — re-export validate, edit

## Implementation Steps

1. **Implement `src/hashline/validate.rs`**
   - `fn parse_line_ref(s: &str) -> Result<(usize, String)>` — parse `5#HH` into (5, "HH")
   - `fn validate_hash(file_lines: &[String], line_num: usize, expected_cid: &str) -> Result<()>`
     - Compute hash for line at line_num
     - Compare with expected CID
     - Error message: `"Hash mismatch at line {n}: expected {expected}, got {actual}. File may have changed since last read."`

2. **Implement `src/hashline/edit.rs`**
   - Enum `EditOp { Replace { start, end, content }, Append { after, content }, Prepend { before, content }, Delete { start, end } }`
   - `fn apply_edit(lines: &mut Vec<String>, op: EditOp) -> Result<()>`
     - Replace: splice lines[start-1..=end-1] with new content lines
     - Append: insert after target line
     - Prepend: insert before target line
     - Delete: remove lines[start-1..=end-1]
   - `fn generate_diff(original: &[String], modified: &[String], filename: &str) -> String`
     - Unified diff format (minimal, context=3)

3. **Implement `src/cmd/edit.rs`**
   - Parse CLI args via clap:
     ```
     sl edit <FILE> <START_REF> [END_REF] [CONTENT]
     sl edit <FILE> --after <REF> <CONTENT>
     sl edit <FILE> --before <REF> <CONTENT>
     sl edit <FILE> --delete <START_REF> [END_REF]
     ```
   - Content from positional arg or stdin (for multi-line)
   - Support `\n` in content string as literal newlines
   - Flow: read file → validate hashes → apply edit → write back → print diff
   - Atomic write: write to `{file}.tmp`, rename over original
   - Optionally create `.bak` backup

4. **Wire into `main.rs`**: replace Edit stub

5. **Handle edge cases**:
   - Edit at line 0 → error
   - Edit beyond EOF → error
   - Empty content for replace → effectively a delete
   - Content with trailing newline → don't add extra blank line

6. **Newline content parsing**: support escape sequences in content arg
   - `\n` → newline, `\t` → tab, `\\` → backslash
   - Alternative: `--stdin` flag to read content from stdin

## Todo List

- [ ] Implement validate.rs (parse line refs, hash comparison)
- [ ] Implement edit.rs (EditOp enum, apply_edit, generate_diff)
- [ ] Implement cmd/edit.rs (CLI args, orchestration, atomic write)
- [ ] Wire Edit command into main.rs
- [ ] Support escape sequences in content arg
- [ ] Implement --stdin flag for multi-line content
- [ ] Unit tests for validation logic
- [ ] Unit tests for each edit operation (replace, append, prepend, delete)
- [ ] Integration test: read → edit → read → verify changes
- [ ] Edge case tests (EOF, line 0, hash mismatch)

## Success Criteria

- Single-line replace works with hash validation
- Range replace works
- Append/prepend/delete operations work
- Hash mismatch produces clear error
- Diff output shows changes
- Atomic writes prevent corruption
- Round-trip: `sl read` → `sl edit` → `sl read` shows expected changes

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Content arg parsing ambiguity | Medium | Use `--stdin` for complex multi-line content, escape sequences for simple cases |
| Race condition on concurrent edits | Low | Hash validation catches stale state; atomic rename prevents partial writes |
| Large file edit performance | Low | Only load affected line range + context |

## Security Considerations

- Validate file path (no path traversal)
- Atomic writes prevent partial file corruption
- Backup files don't leak into git (add to .gitignore template)
