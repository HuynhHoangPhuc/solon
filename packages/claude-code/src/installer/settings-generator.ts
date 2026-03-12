import type { ClaudeSettings, HookBinding, McpServerConfig } from './types.js';

const SOLON_HOOKS_PATH = '.claude/hooks/solon';

export function generateSettings(existingSettings: ClaudeSettings = {}): ClaudeSettings {
  const mergedHooks = mergeHooks(existingSettings.hooks ?? {}, {
    PreToolUse: [
      {
        matcher: 'Edit',
        hooks: [{ type: 'command', command: `node ${SOLON_HOOKS_PATH}/hashline-edit-hook.cjs` }],
      },
      {
        matcher: 'Bash|Glob|Grep|Read|Edit|Write',
        hooks: [{ type: 'command', command: `node ${SOLON_HOOKS_PATH}/scout-block-hook.cjs` }],
      },
    ],
    PostToolUse: [
      {
        matcher: 'Bash|Grep|Glob',
        hooks: [{ type: 'command', command: `node ${SOLON_HOOKS_PATH}/output-truncation-hook.cjs` }],
      },
      {
        matcher: 'Write',
        hooks: [{ type: 'command', command: `node ${SOLON_HOOKS_PATH}/write-suppression-hook.cjs` }],
      },
    ],
    SubagentStart: [
      {
        matcher: '*',
        hooks: [{ type: 'command', command: `node ${SOLON_HOOKS_PATH}/subagent-init-hook.cjs` }],
      },
    ],
    UserPromptSubmit: [
      {
        hooks: [{ type: 'command', command: `node ${SOLON_HOOKS_PATH}/dedup-guard-hook.cjs` }],
      },
    ],
  });

  const mcpServers: Record<string, McpServerConfig> = {
    ...(existingSettings.mcpServers ?? {}),
    solon: {
      type: 'stdio',
      command: 'node',
      args: [`.claude/hooks/solon/mcp-server.cjs`],
    },
  };

  return { ...existingSettings, hooks: mergedHooks, mcpServers };
}

function mergeHooks(
  existing: Record<string, HookBinding[]>,
  solonHooks: Record<string, HookBinding[]>
): Record<string, HookBinding[]> {
  const result: Record<string, HookBinding[]> = { ...existing };
  for (const [event, bindings] of Object.entries(solonHooks)) {
    const existingBindings = result[event] ?? [];
    // Filter out any existing solon hooks (for idempotent reinstall)
    const nonSolonBindings = existingBindings.filter(b =>
      !b.hooks?.some(h => h.command?.includes('solon'))
    );
    result[event] = [...nonSolonBindings, ...bindings];
  }
  return result;
}
