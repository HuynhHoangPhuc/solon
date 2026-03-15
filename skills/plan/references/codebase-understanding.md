# Codebase Understanding

**When to skip:** If provided with scout reports, skip this phase.

## Essential Docs to Read First

Always read before writing any plan:

1. **`./docs/codebase-summary.md`** — project structure, current status, component relationships
2. **`./docs/code-standards.md`** — coding conventions, naming, patterns
3. **`./docs/system-architecture.md`** — architectural decisions, data flow, tech stack
4. **`./docs/development-rules.md`** — file size limits, naming rules, quality standards

If any of these are missing or stale, proceed with codebase scouting below.

## Scout Patterns

### Structure Discovery (Glob)

```bash
# Map top-level structure
Glob("**/*", path: "src/")

# Find specific file types
Glob("**/*.ts", path: "src/")
Glob("**/*service*", path: "src/")
Glob("**/*controller*", path: "src/")
```

### Pattern Search (Grep)

```bash
# Find how auth is currently implemented
Grep("authenticate|authorize", type: "ts")

# Find API route patterns
Grep("router\.(get|post|put|delete)", type: "ts")

# Find config patterns
Grep("export.*config|module\.exports", type: "ts")
```

### Semantic Code Search (AST)

```bash
# Find function definitions matching pattern
sl ast search "function authenticate" --lang typescript

# Find class definitions
sl ast search "class.*Service" --lang typescript

# Find interface definitions
sl ast search "interface.*Repository" --lang typescript
```

### Error & Diagnostics

```bash
# Check for existing LSP errors before planning
sl lsp diagnostics src/
```

## Pattern Recognition

Before writing implementation steps, identify:
- How existing features are structured (find a similar feature, read it)
- Error handling patterns in use
- Testing patterns (test file naming, assertion style)
- Import/module patterns
- Database access patterns (ORM vs raw, repository vs direct)

## Integration Planning

Map out before writing phase files:
- Which existing files get modified (not just created)
- Backward compatibility risks (public API changes)
- Database migration requirements
- Config/env variable additions

## Best Practices

- Read code before researching externally — answer may already be in codebase
- Note inconsistencies (existing tech debt) and document in Key Insights
- Find the most similar existing feature — follow its pattern unless there's good reason not to
- Check test files to understand expected behavior of existing code
