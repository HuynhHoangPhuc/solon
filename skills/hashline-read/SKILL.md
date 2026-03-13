---
name: hashline-read
description: Read files with hashline annotations for reliable line-range editing
---

# Hashline Read

Use `sl read` instead of the Read tool when you plan to edit a code file.

## Usage

```bash
sl read <file>                    # read entire file with line hashes
sl read <file> --lines 5:20       # read lines 5-20 only
sl read <file> --lines 5:         # read from line 5 to EOF
sl read <file> --chunk-size 100   # limit output to 100 lines
```

## Output Format

Each line: `LINE#HASH|CONTENT`

Example:
```
1#ZP|fn main() {
2#MQ|    println!("hello");
3#HW|
4#RK|}
```

## When to Use

- Always before editing a code file with `sl edit`
- When you need exact line numbers + hashes for targeting edits
- Use native Read for non-code files (images, PDFs, configs you won't edit)

## Notes

- Large files are chunked at 200 lines by default — use `--lines` to navigate
- Binary files are rejected with a clear error
- BOM and CRLF line endings are handled automatically
