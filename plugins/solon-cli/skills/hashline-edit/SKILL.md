---
name: sl:hashline-edit
description: Edit files using hashline line references for reliable, hash-validated edits
---

# Hashline Edit

Use `sl edit` to make changes to code files. Always run `sl read` first to get current hashes.

## Operations

### Replace a single line
```bash
sl edit <file> 5#HH "new content for line 5"
```

### Replace a range of lines
```bash
sl edit <file> 5#HH 10#QQ "replacement line 1\nreplacement line 2"
```

### Append after a line
```bash
sl edit <file> --after 3#MQ "new line inserted after line 3"
```

### Prepend before a line
```bash
sl edit <file> --before 1#ZP "header line added before line 1"
```

### Delete lines
```bash
sl edit <file> --delete 5#HH 10#QQ
```

### Multi-line content via stdin
```bash
echo -e "line1\nline2\nline3" | sl edit <file> 5#HH --stdin
```

## Hash Mismatch

If you see `Hash mismatch at line N`, the file changed since your last `sl read`. Re-read the file to get updated hashes before editing.

## Content Escapes

In content strings: `\n` = newline, `\t` = tab, `\\` = backslash.

## After Editing

Run `sl read <file>` again to verify changes and get updated hashes for further edits.
