#!/usr/bin/env node
'use strict';
// PostToolUse(Write) — suppress echoed file content, show brief summary
try {
  const fs = require('node:fs');
  const { loadConfig, isHookEnabled } = require('./lib/config-utils.cjs');

  const input = JSON.parse(fs.readFileSync('/dev/stdin', 'utf8'));
  const config = loadConfig(process.env.SOLON_PROJECT_DIR || process.cwd());

  if (!isHookEnabled(config, 'writeSuppression')) process.exit(0);

  const filePath = input.tool_input?.path ?? input.tool_input?.file_path;
  if (!filePath) process.exit(0);

  let lineCount = 0;
  try {
    const content = fs.readFileSync(filePath, 'utf8');
    lineCount = content.split('\n').length;
    if (content.endsWith('\n')) lineCount--;
  } catch {
    lineCount = -1;
  }

  const summary = lineCount >= 0
    ? `File written successfully. ${lineCount} lines.`
    : `File written successfully.`;

  process.stdout.write(JSON.stringify({
    hookSpecificOutput: {
      hookEventName: 'PostToolUse',
      additionalContext: summary,
    },
  }));
} catch {
  process.exit(0);
}
