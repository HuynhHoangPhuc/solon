'use strict';

/**
 * scout-block.cjs — PreToolUse hook for Glob and Grep
 * Blocks overly broad patterns that would scan the entire repo.
 * Returns a block with suggestions to use more specific patterns.
 */

const BROAD_PATTERNS = [
  /^\*\*\/\*$/,      // **/*
  /^\*$/,            // *
  /^\.\/*$/,         // ./*
  /^\*\*$/,          // **
  /^\.\*$/,          // .*
];

const MAX_ALLOWED_STARS = 3;

function isBroadPattern(pattern) {
  if (!pattern || typeof pattern !== 'string') return false;
  for (const re of BROAD_PATTERNS) {
    if (re.test(pattern)) return true;
  }
  // Count wildcards: if the pattern is only stars and slashes, it's too broad
  const stripped = pattern.replace(/[^*]/g, '');
  if (stripped.length > MAX_ALLOWED_STARS && !pattern.includes('.')) {
    return true;
  }
  return false;
}

function main() {
  let input = '';
  process.stdin.setEncoding('utf8');
  process.stdin.on('data', chunk => { input += chunk; });
  process.stdin.on('end', () => {
    try {
      const event = JSON.parse(input);
      const tool = event.tool_name || '';
      const input_data = event.tool_input || {};

      let pattern = '';
      if (tool === 'Glob') {
        pattern = input_data.pattern || '';
      } else if (tool === 'Grep') {
        // Check both pattern and glob filter
        pattern = input_data.glob || input_data.pattern || '';
      }

      if (isBroadPattern(pattern)) {
        const result = {
          decision: 'block',
          reason: `Pattern "${pattern}" is too broad and would scan the entire repository. ` +
            `Use a more specific pattern, e.g.:\n` +
            `  • "src/**/*.rs" to search Rust files in src/\n` +
            `  • "**/*.ts" to search TypeScript files\n` +
            `  • "src/module/specific-file.rs" for a single file\n` +
            `Or use "sl ast search" for semantic search across the codebase.`,
        };
        process.stdout.write(JSON.stringify(result));
      } else {
        // Allow
        process.stdout.write(JSON.stringify({ decision: 'allow' }));
      }
    } catch (e) {
      // On parse error, allow (fail open)
      process.stdout.write(JSON.stringify({ decision: 'allow' }));
    }
  });
}

main();
