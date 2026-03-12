#!/usr/bin/env node
'use strict';
// PreToolUse(Bash|Glob|Grep|Read|Edit|Write) — block broad patterns + blocked dirs
try {
  const fs = require('node:fs');
  const { checkScoutBlock } = require('./lib/scout-checker.cjs');
  const { loadConfig, isHookEnabled } = require('./lib/config-utils.cjs');

  const input = JSON.parse(fs.readFileSync('/dev/stdin', 'utf8'));
  const config = loadConfig(process.env.SOLON_PROJECT_DIR || process.cwd());

  if (!isHookEnabled(config, 'scoutBlock')) process.exit(0);

  const { tool_name, tool_input } = input;
  const projectDir = process.env.SOLON_PROJECT_DIR || process.cwd();
  const result = checkScoutBlock(tool_name, tool_input ?? {}, projectDir);

  if (!result.blocked) process.exit(0);

  process.stderr.write(`[solon:scout-block] ${result.reason}\n`);
  if (result.suggestion) process.stderr.write(`Suggestion: ${result.suggestion}\n`);
  process.exit(2); // block
} catch {
  process.exit(0); // fail-open
}
