#!/usr/bin/env node
'use strict';
// PostToolUse(Bash|Grep|Glob) — add warning if output exceeds token cap
try {
  const fs = require('node:fs');
  const { loadConfig, isHookEnabled } = require('./lib/config-utils.cjs');

  const input = JSON.parse(fs.readFileSync('/dev/stdin', 'utf8'));
  const config = loadConfig(process.env.SOLON_PROJECT_DIR || process.cwd());

  if (!isHookEnabled(config, 'outputTruncation')) process.exit(0);

  const response = input.tool_response;
  const outputText = response?.output ?? response?.content ?? '';
  if (typeof outputText !== 'string' || outputText.length === 0) process.exit(0);

  const MAX_TOKENS = config.truncation?.maxTokens ?? 50000;
  const tokens = Math.ceil(outputText.length / 4);

  if (tokens <= MAX_TOKENS) process.exit(0);

  // Truncate and add warning
  const maxChars = MAX_TOKENS * 4;
  const lines = outputText.split('\n');
  const truncatedLines = [];
  let charCount = 0;
  const PRESERVE_HEADER = 3;

  for (let i = 0; i < Math.min(PRESERVE_HEADER, lines.length); i++) {
    truncatedLines.push(lines[i]);
    charCount += (lines[i]?.length ?? 0) + 1;
  }

  for (let i = PRESERVE_HEADER; i < lines.length; i++) {
    const lineLen = (lines[i]?.length ?? 0) + 1;
    if (charCount + lineLen > maxChars) {
      const remaining = lines.length - i;
      truncatedLines.push(`\n[${remaining} more lines truncated due to context window limit]`);
      break;
    }
    truncatedLines.push(lines[i]);
    charCount += lineLen;
  }

  process.stdout.write(JSON.stringify({
    hookSpecificOutput: {
      hookEventName: 'PostToolUse',
      additionalContext: `[Output truncated: ${tokens} estimated tokens exceeds ${MAX_TOKENS} cap]\n\n${truncatedLines.join('\n')}`,
    },
  }));
} catch {
  process.exit(0);
}
