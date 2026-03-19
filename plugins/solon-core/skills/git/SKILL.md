---
name: sl:git
description: "Git operations with conventional commits. Use for staging, committing, pushing, PRs. Auto-splits by scope. Security scans for secrets."
argument-hint: "cm|cp|pr [args]"
---

# Git — Smart Git Operations

Conventional commits with auto-split, security scanning, and PR creation.

## Usage

```
/sl:git cm          # commit
/sl:git cp          # commit + push
/sl:git pr          # create pull request
```

No args: use `AskUserQuestion` with cm/cp/pr options.

## Workflow

### Step 1 — Mode Selection

| Command | Action |
|---------|--------|
| `cm` | Stage, scan, commit |
| `cp` | Stage, scan, commit, push |
| `pr` | Stage, scan, commit, push, create PR via `gh` |

### Step 2 — Stage & Analyze

```bash
git status
git diff --stat
git diff
```

Review changes. Stage specific files by name (prefer over `git add -A` to avoid accidental inclusion of secrets or large binaries). Identify file scopes and change types.

### Step 3 — Security Scan

Scan cached diff for secrets before committing:
- API keys, tokens, passwords
- `.env` files, `credentials.json`
- Private keys, certificates
- Connection strings with embedded passwords

**If secrets found: STOP and warn user.** Do not proceed with commit.

### Step 4 — Commit Strategy

Analyze changed files for split decision:

| Condition | Strategy |
|-----------|----------|
| Same scope, same type, ≤5 files | Single commit |
| Different types (feat + fix) | Split by type |
| Different scopes (api + ui) | Split by scope |

Commit message format: `type(scope): description`

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `style`

### Step 5 — Execute

Delegate to `git-manager` agent via Agent tool with:

```
Project root: <path>
Operation: <cm|cp|pr>
Staged changes: <diff stat summary>
Commit strategy: <single or split details>
Commit messages: <drafted messages>

Instructions:
1. Create commits per strategy
2. Never skip hooks (--no-verify)
3. Never force-push without explicit user confirmation
4. For PR: use `gh pr create` with title + body
5. Return commit hashes and PR URL if applicable
```

### Step 6 — Report

Output:
- Commit hash(es) and messages
- Push status (if cp/pr)
- PR URL (if pr)

## Constraints

- Never skip pre-commit hooks (`--no-verify`)
- Never force-push without explicit user confirmation
- Never commit secrets — scan first, warn if found
- Delegate execution to `git-manager` agent (isolate verbose output)
- Clean, professional commit messages — no AI references
