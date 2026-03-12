/** Glob patterns considered overly broad (would match too many files) */
const BROAD_PATTERNS: Array<{ regex: RegExp; suggestion: string }> = [
  { regex: /^\*\*\/\*$/, suggestion: "Use a specific directory like src/**/*" },
  {
    regex: /^\*\*\/\*\.\w+$/,
    suggestion: "Prefix with a directory, e.g. src/**/*.ts",
  },
  { regex: /^\*$/, suggestion: "Use a specific directory glob like src/*" },
  {
    regex: /^\.\/\*$/,
    suggestion: "Use ./{dir}/* to target a specific folder",
  },
  { regex: /^\*\*$/, suggestion: "Use a specific path prefix" },
  { regex: /^\.$/, suggestion: "Use a specific directory like ./src" },
];

export interface BroadPatternResult {
  blocked: boolean;
  pattern?: string;
  suggestion?: string;
}

/**
 * Check whether a glob pattern is too broad for safe use in a tool call.
 * Returns blocked=true with a suggestion if the pattern matches a broad rule.
 */
export function checkBroadPattern(
  pattern: string,
  _toolName: string,
): BroadPatternResult {
  const trimmed = pattern.trim();
  for (const { regex, suggestion } of BROAD_PATTERNS) {
    if (regex.test(trimmed)) {
      return { blocked: true, pattern: trimmed, suggestion };
    }
  }
  return { blocked: false };
}
