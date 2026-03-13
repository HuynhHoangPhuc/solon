---
phase: 2
title: "Hashline Core"
status: complete
priority: P1
effort: 5h
depends_on: [1]
---

# Phase 2: Hashline Core

## Context Links

- [Plan Overview](plan.md)
- [Brainstorm — Hashline Format](../reports/brainstorm-260313-0346-solon-plugin-architecture.md)

## Overview

Implement the hashline computation engine and `sl read` command. This is the core primitive — every edit depends on correct hash generation and line annotation.

## Key Insights

- xxHash32 chosen for speed + small output (4 bytes)
- 2-char CID from 16-char alphabet `ZPMQVRWSNKTXJBYH` = 256 unique codes
- Seed selection: 0 for alphanumeric lines, `line_number` for blank/punctuation-only lines (avoids collisions on similar empty lines)
- Format: `LINE#HASH|CONTENT` — e.g., `1#ZP|fn main() {`

## Requirements

### Functional
- `sl read <file>` outputs every line with hashline annotation
- `sl read <file> --lines 5:20` outputs lines 5-20 only
- `sl read <file> --lines 5:` outputs lines 5 to EOF
- Hash is deterministic: same content + same line classification = same hash
- Handle UTF-8, binary files (reject gracefully), empty files

### Non-Functional
- Performance: read 100K-line file in < 500ms
- Output chunking: configurable max lines (default 200) for large files
- Memory: streaming, not full file in memory for giant files

## Architecture

```
sl read <file> [--lines N:M] [--chunk-size C]
        │
        ▼
  ┌──────────────┐
  │ File loader   │ ← canonicalize (BOM strip, CRLF→LF)
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Hash engine   │ ← xxHash32(content, seed) → mod 256 → CID
  └──────┬───────┘
         ▼
  ┌──────────────┐
  │ Formatter     │ ← "LINE#HASH|CONTENT"
  └──────────────┘
```

## Related Code Files

### Create
- `src/hashline/hash.rs` — xxHash32 computation, CID mapping
- `src/hashline/format.rs` — line formatting (LINE#HASH|CONTENT)
- `src/hashline/canonicalize.rs` — BOM stripping, line ending normalization
- `src/cmd/read.rs` — `sl read` command implementation

### Modify
- `Cargo.toml` — add `xxhash-rust` dependency
- `src/main.rs` — wire Read subcommand
- `src/hashline/mod.rs` — re-export submodules
- `src/cmd/mod.rs` — re-export read

## Implementation Steps

1. **Add dependency**: `xxhash-rust = { version = "0.8", features = ["xxh32"] }`

2. **Implement `src/hashline/hash.rs`**
   - CID alphabet constant: `ZPMQVRWSNKTXJBYH`
   - `fn classify_line(line: &str) -> bool` — true if line has alphanumeric chars
   - `fn compute_hash(line: &str, line_number: usize) -> [u8; 2]`
     - seed = 0 if alphanumeric content, else line_number as u32
     - hash = xxh32(line.as_bytes(), seed)
     - index = hash % 256
     - high = index / 16, low = index % 16
     - return `[ALPHABET[high], ALPHABET[low]]`
   - `fn hash_to_cid(line: &str, line_number: usize) -> String` — convenience wrapper

3. **Implement `src/hashline/canonicalize.rs`**
   - `fn strip_bom(content: &[u8]) -> &[u8]` — remove UTF-8 BOM (EF BB BF)
   - `fn normalize_line_endings(content: &str) -> String` — CRLF → LF
   - `fn detect_line_ending(content: &str) -> LineEnding` — for restoring on write

4. **Implement `src/hashline/format.rs`**
   - `fn format_line(line_number: usize, cid: &str, content: &str) -> String`
   - Returns `"{line_number}#{cid}|{content}"`
   - `fn parse_hashline(annotated: &str) -> Option<(usize, String, String)>` — inverse

5. **Implement `src/cmd/read.rs`**
   - Parse args: file path, optional `--lines N:M`, optional `--chunk-size`
   - Open file, canonicalize content
   - For each line in range: compute hash, format, print
   - Exit with error if file not found, is binary, or is directory
   - If line count exceeds chunk-size: print warning with total line count

6. **Wire into `main.rs`**: replace Read stub with actual command dispatch

7. **Unit tests** in `src/hashline/hash.rs`:
   - Known input → known CID (create reference vectors)
   - Blank line vs alphanumeric line use different seeds
   - BOM stripping works
   - CRLF normalization works

## Todo List

- [ ] Add xxhash-rust dependency
- [ ] Implement hash.rs (xxHash32, CID alphabet, seed logic)
- [ ] Implement canonicalize.rs (BOM, CRLF)
- [ ] Implement format.rs (LINE#HASH|CONTENT formatting + parsing)
- [ ] Implement cmd/read.rs (file reading, line range, chunking)
- [ ] Wire Read command into main.rs
- [ ] Write unit tests for hash computation
- [ ] Write unit tests for canonicalization
- [ ] Write integration test: read a sample file, verify output format
- [ ] Performance test: 100K-line file < 500ms

## Success Criteria

- `sl read src/main.rs` outputs hashline-annotated file
- `sl read src/main.rs --lines 1:5` outputs only lines 1-5
- Hash is deterministic across runs
- Binary files rejected with clear error
- Unit tests pass

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| xxHash32 mod 256 collisions | Low | Acceptable — CID is validation hint, not unique ID. Line number + CID together suffice |
| Large file memory pressure | Medium | Stream lines instead of loading entire file |
| Encoding issues (non-UTF8) | Medium | Detect encoding, reject non-UTF8 with error message |

## Security Considerations

- No file writes in this phase
- Respect filesystem permissions (don't bypass)
- Don't follow symlinks outside project root (optional, add if needed)
