#!/usr/bin/env node
'use strict';
// UserPromptSubmit — prevent duplicate context injection
// Exits 0 (no output) to skip injection, or outputs context to inject
try {
  const fs = require('node:fs');
  const { loadConfig, isHookEnabled } = require('./lib/config-utils.cjs');

  const input = JSON.parse(fs.readFileSync('/dev/stdin', 'utf8'));
  const config = loadConfig(process.env.SOLON_PROJECT_DIR || process.cwd());

  if (!isHookEnabled(config, 'dedupGuard')) process.exit(0);

  // Check recent transcript for sentinel
  const SENTINEL = '<!-- solon:rules-injected -->';
  const transcript = input.transcript ?? [];
  const recentMessages = transcript.slice(-150);
  const alreadyInjected = recentMessages.some(msg =>
    typeof msg?.content === 'string' && msg.content.includes(SENTINEL)
  );

  if (alreadyInjected) process.exit(0);

  // Inject minimal dev rules reminder
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: {
      hookEventName: 'UserPromptSubmit',
      additionalContext: `${SENTINEL}\n## Dev Rules Reminder\n- YAGNI, KISS, DRY\n- No files > 200 LOC\n- Fail-open on hook errors`,
    },
  }));
} catch {
  process.exit(0);
}
