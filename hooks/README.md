# solon hooks

Claude Code plugin providing 13 productivity hooks for session management, context injection, privacy protection, and notifications.

## Requirements

- **Bun** (recommended) or **Node v23+** with `--experimental-strip-types`
- Claude Code CLI

## Installation

```bash
claude --plugin-dir .
```

Or add to your global Claude Code settings:

```json
{
  "plugins": ["/absolute/path/to/solon"]
}
```

## Configuration

Copy example configs to activate:

```bash
# Hook configuration (enable/disable hooks, plan naming, locale)
cp.sl.json.example .claude/.sl.json

# Directory access control
cp.slignore.example .claude/.slignore

# Notification credentials (optional)
cp.env.example .claude/.env
```

### `.sl.json` Options

| Field | Default | Description |
|-------|---------|-------------|
| `plan.namingFormat` | `{date}-{issue}-{slug}` | Plan directory naming pattern |
| `plan.dateFormat` | `YYMMDD-HHmm` | Date format (`YYMMDD-HHmm`, `YYYYMMDD`, `YYYY-MM-DD`) |
| `plan.issuePrefix` | `""` | Issue tracker prefix (e.g. `SL-`, `PROJ-`) |
| `hooks.<name>` | `true` | Enable/disable individual hooks |
| `locale.responseLanguage` | `""` | Language for Claude responses |
| `codingLevel` | `5` | Coding guidance verbosity (1–10) |

## Hooks

| Hook | Event | Description |
|------|-------|-------------|
| `session-init` | SessionStart | Detect project type, write env vars, inject context |
| `subagent-init` | SubagentStart | Inject compact context (~200 tokens) to subagents |
| `team-context-inject` | SubagentStart | Inject team peer list and task summary for Agent Teams |
| `dev-rules-reminder` | UserPromptSubmit | Inject dev rules reminder into context |
| `scout-block` | PreToolUse | Block `.slignore` directories + broad Glob patterns |
| `privacy-block` | PreToolUse | Block sensitive files (.env, keys) with approval flow |
| `descriptive-name` | PreToolUse | Remind Claude to use descriptive file names |
| `post-edit-simplify-reminder` | PostToolUse | Remind to simplify after N file edits (default: 5) |
| `cook-after-plan-reminder` | SubagentStop | Remind cook workflow after Plan subagent completes |
| `usage-context-awareness` | UserPromptSubmit | Fetch and cache Claude API usage limits |
| `task-completed-handler` | PostToolUse | Log task completions to markdown file |
| `teammate-idle-handler` | SubagentStop | Inject available tasks when a teammate goes idle |

## Scout Block (`.slignore`)

Blocks Claude from accessing heavy directories (node_modules, dist, .git, etc.) to prevent context overflow. Uses gitignore-spec pattern matching.

```gitignore
# .claude/.slignore
node_modules
dist
.git
__pycache__

# Allow specific paths
!dist/assets
```

## Privacy Block

Blocks access to sensitive files (`.env`, credentials, private keys) until the user explicitly approves.

**Flow:**
1. Claude reads `.env` → **BLOCKED** with prompt
2. Claude uses `AskUserQuestion` tool to request approval
3. User approves → Claude retries with `APPROVED:.env` prefix
4. Access granted

## Notifications

Set credentials in `.claude/.env` (see `.env.example`). Supports Telegram, Discord, and Slack. Fires on `Stop`, `SubagentStop`, and `AskUserPrompt` events.

## Runtime

The `hooks/scripts/run.sh` shim selects the runtime automatically:

```bash
# Tries in order: bun → node --experimental-strip-types
hooks/scripts/run.sh hooks/hooks/session-init.ts
```

## Development

```bash
# Run tests
bun test hooks/scripts/__tests__/

# Compile-check all hooks
for f in hooks/scripts/hooks/*.ts; do echo '{}' | bun run --bun "$f" && echo "OK $f"; done
```

## Troubleshooting

- **Hook not firing**: Check `hooks.<name>` in `.sl.json` (must not be `false`)
- **scout-block blocking too much**: Add `!pattern` to `.claude/.slignore`
- **Privacy block always firing**: Use `APPROVED:` prefix or disable `privacy-block` in `.sl.json`
- **Notifications not sending**: Check credentials in `.claude/.env`, verify webhook URLs
