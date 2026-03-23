---
name: sl:release
description: "Manage releases with semantic versioning, changelog generation, and tag management"
argument-hint: "[major|minor|patch] [--dry-run] [--no-push]"
---

# Release — Semantic Version & Changelog Manager

Orchestrate releases: bump version, generate changelog, create git tag, optionally push.

**See also:** `../../references/shared/skill-decision-tree.md` for when to use `/sl:release` vs other skills.

## Usage

```
/sl:release [major|minor|patch] [--dry-run] [--no-push]
```

| Arg | Default | Effect |
|-----|---------|--------|
| `major` / `minor` / `patch` | auto-detect | Version bump type. Auto-detect analyzes conventional commits since last tag. |
| `--dry-run` | off | Preview version bump and changelog without modifying files |
| `--no-push` | off | Create tag locally but don't push to remote |

## Workflow

### Step 1 — Detect Current Version

Read version from root `Cargo.toml` (workspace version):

```bash
grep '^version' Cargo.toml | head -1 | sed 's/.*"\(.*\)".*/\1/'
```

Also check `marketplace.json` and `plugin.json` files for version consistency.

### Step 2 — Analyze Commits Since Last Tag

```bash
git log $(git describe --tags --abbrev=0 2>/dev/null || echo "")..HEAD --pretty=format:"%s" --no-merges
```

Parse conventional commits to determine bump type if not explicitly provided:
- `feat!:` or `BREAKING CHANGE:` → **major**
- `feat:` → **minor**
- `fix:`, `docs:`, `chore:`, `refactor:`, `test:`, `perf:` → **patch**

### Step 3 — Bump Version

Update version in these files:
1. Root `Cargo.toml` — workspace version
2. All `Cargo.toml` files in workspace members that reference workspace version
3. `.claude-plugin/marketplace.json` — if present
4. `plugins/*/. claude-plugin/plugin.json` — all plugin manifests

### Step 4 — Generate Changelog

Group commits by type and append to `CHANGELOG.md`:

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Features
- feat: description (#PR)

### Bug Fixes
- fix: description (#PR)

### Other Changes
- chore/docs/refactor entries
```

### Step 5 — Create Git Tag

```bash
git add -A
git commit -m "chore: release vX.Y.Z"
git tag vX.Y.Z
```

### Step 6 — Push (unless `--no-push`)

```bash
git push origin main --tags
```

This triggers CI/CD release workflow via GitHub Actions.

### Dry Run Mode

When `--dry-run` is specified:
- Show current version → proposed version
- Show changelog preview
- Show files that would be modified
- Do NOT modify any files, create tags, or push

## Integration

- Uses existing `scripts/release.sh` for binary artifact builds (triggered by CI on tag push)
- Compatible with GitHub Actions release workflow
- Follows conventional commit format enforced by Solon hooks

## Prerequisites

- Clean working directory (no uncommitted changes)
- On `main` branch
- `gh` CLI authenticated (for release creation)
