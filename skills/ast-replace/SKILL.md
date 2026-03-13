---
name: ast-replace
description: Semantic code replacement using ast-grep patterns with safe preview mode
---

# AST Replace

Use `sl ast replace` for structural code transformations.

## Usage

```bash
# Preview changes (default — does NOT apply)
sl ast replace "<pattern>" "<replacement>" --lang <lang>

# Apply changes to all matches
sl ast replace "<pattern>" "<replacement>" --lang <lang> --update-all
```

## Pattern → Replacement

Captured variables (`$NAME`, `$$$ARGS`) are reused in the replacement.

### Examples

**Rename a function call:**
```bash
sl ast replace "console.log($$$ARGS)" "logger.info($$$ARGS)" --lang typescript
```

**Add error handling:**
```bash
sl ast replace "foo.unwrap()" "foo.expect(\"foo should be set\")" --lang rust
```

**Modernize syntax:**
```bash
sl ast replace "var $NAME = $VAL" "const $NAME = $VAL" --lang javascript --update-all
```

## Safety

- Always preview first (omit `--update-all`) to verify before applying
- Changes modify files in place — commit first or use `sl edit` for surgical edits
- Respects `.gitignore` by default

## Notes

- Requires `sg` binary (auto-downloaded on first use)
- Use `--path src/` to scope replacements to a directory
