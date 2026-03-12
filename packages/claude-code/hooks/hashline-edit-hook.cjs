#!/usr/bin/env node
'use strict';
// PreToolUse(Edit) — translates hashline refs in old_string to actual file content
try {
  const fs = require('node:fs');
  const { resolveHashlineEdit } = require('./lib/hashline-resolver.cjs');
  const { loadConfig, isHookEnabled } = require('./lib/config-utils.cjs');

  const input = JSON.parse(fs.readFileSync('/dev/stdin', 'utf8'));
  const config = loadConfig(process.env.SOLON_PROJECT_DIR || process.cwd());

  if (!isHookEnabled(config, 'hashlineEdit')) process.exit(0);

  const { tool_input } = input;
  const oldString = tool_input?.old_string ?? tool_input?.oldString;
  const filePath = tool_input?.path ?? tool_input?.file_path;

  if (!oldString || !filePath) process.exit(0);

  // Check if there are hashline refs
  const HASHLINE_REF_TEST = /(\d+)#([ZPMQVRWSNKTXJBYH]{2})/;
  if (!HASHLINE_REF_TEST.test(oldString)) process.exit(0);

  const result = resolveHashlineEdit(oldString, filePath);
  if (!result) process.exit(0); // no pure refs

  if (result.error) {
    // Hash mismatch — deny with explanation
    process.stdout.write(JSON.stringify({
      hookSpecificOutput: {
        hookEventName: 'PreToolUse',
        permissionDecision: 'deny',
        denyReason: `Hashline edit failed: ${result.error}. Re-read the file to get updated LINE#ID references.`,
      },
    }));
    process.exit(0);
  }

  // Success — return resolved content
  process.stdout.write(JSON.stringify({
    hookSpecificOutput: {
      hookEventName: 'PreToolUse',
      permissionDecision: 'allow',
      updatedInput: { old_string: result.resolved },
    },
  }));
} catch {
  process.exit(0); // fail-open: never block agent on hook crash
}
