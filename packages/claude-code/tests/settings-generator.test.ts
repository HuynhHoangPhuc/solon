import { describe, it, expect } from 'vitest';
import { generateSettings } from '../src/installer/settings-generator.js';

describe('generateSettings', () => {
  it('generates hooks and mcpServers from empty input', () => {
    const settings = generateSettings({});
    expect(settings.hooks).toBeDefined();
    expect(settings.hooks?.PreToolUse).toBeDefined();
    expect(settings.mcpServers?.solon).toBeDefined();
  });

  it('registers solon MCP server with stdio type', () => {
    const settings = generateSettings({});
    expect(settings.mcpServers?.solon?.type).toBe('stdio');
    expect(settings.mcpServers?.solon?.command).toBe('node');
  });

  it('preserves existing hooks when merging', () => {
    const existing = {
      hooks: {
        PreToolUse: [{ matcher: 'Edit', hooks: [{ type: 'command' as const, command: 'my-hook' }] }],
      },
    };
    const merged = generateSettings(existing);
    const preToolUse = merged.hooks?.PreToolUse ?? [];
    const hasExistingHook = preToolUse.some(b => b.hooks?.some(h => h.command === 'my-hook'));
    expect(hasExistingHook).toBe(true);
  });

  it('preserves existing mcpServers', () => {
    const existing = {
      mcpServers: { myServer: { type: 'stdio' as const, command: 'myserver' } },
    };
    const merged = generateSettings(existing);
    expect(merged.mcpServers?.myServer).toBeDefined();
    expect(merged.mcpServers?.solon).toBeDefined();
  });

  it('idempotent: re-install does not duplicate solon hooks', () => {
    const first = generateSettings({});
    const second = generateSettings(first);
    const countSolonHooks = (hooks: typeof first.hooks) =>
      (hooks?.PreToolUse ?? []).filter(b =>
        b.hooks?.some(h => h.command?.includes('solon'))
      ).length;
    expect(countSolonHooks(second.hooks)).toBe(countSolonHooks(first.hooks));
  });

  it('includes hashline-edit hook on PreToolUse Edit matcher', () => {
    const settings = generateSettings({});
    const preToolUse = settings.hooks?.PreToolUse ?? [];
    const hashlineHook = preToolUse.find(b =>
      b.matcher === 'Edit' && b.hooks?.some(h => h.command?.includes('hashline-edit'))
    );
    expect(hashlineHook).toBeDefined();
  });

  it('includes scout-block hook on PreToolUse', () => {
    const settings = generateSettings({});
    const preToolUse = settings.hooks?.PreToolUse ?? [];
    const scoutHook = preToolUse.find(b =>
      b.hooks?.some(h => h.command?.includes('scout-block'))
    );
    expect(scoutHook).toBeDefined();
  });

  it('includes output-truncation hook on PostToolUse', () => {
    const settings = generateSettings({});
    const postToolUse = settings.hooks?.PostToolUse ?? [];
    const truncHook = postToolUse.find(b =>
      b.hooks?.some(h => h.command?.includes('output-truncation'))
    );
    expect(truncHook).toBeDefined();
  });

  it('includes SubagentStart hook', () => {
    const settings = generateSettings({});
    expect(settings.hooks?.SubagentStart).toBeDefined();
    expect(settings.hooks?.SubagentStart?.length).toBeGreaterThan(0);
  });

  it('preserves extra top-level settings fields', () => {
    const existing = { theme: 'dark', someOtherField: 42 };
    const merged = generateSettings(existing);
    expect(merged.theme).toBe('dark');
    expect(merged.someOtherField).toBe(42);
  });
});
