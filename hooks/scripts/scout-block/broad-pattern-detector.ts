// Detect overly broad glob patterns that would fill context with too many files

const BROAD_PATTERN_REGEXES: RegExp[] = [
  /^\*\*$/,           // ** - all files everywhere
  /^\*$/,             // * - all files in root
  /^\*\*\/\*$/,       // **/* - all files everywhere
  /^\*\*\/\.\*$/,     // **/. - all dotfiles everywhere
  /^\*\.\w+$/,        // *.ext at root
  /^\*\.\{[^}]+\}$/,  // *.{ext,ext2} at root
  /^\*\*\/\*\.\w+$/,  // **/*.ext - all files of type everywhere
  /^\*\*\/\*\.\{[^}]+\}$/, // **/*.{ext,ext2}
];

const SPECIFIC_DIRS = [
  "src", "lib", "app", "apps", "packages", "components", "pages",
  "api", "server", "client", "web", "mobile", "shared", "common",
  "utils", "helpers", "services", "hooks", "store", "routes",
  "models", "controllers", "views", "tests", "__tests__", "spec",
];

const HIGH_RISK_INDICATORS: RegExp[] = [
  /\/worktrees\/[^/]+\/?$/,
  /^\.?\/?$/,
  /^[^/]+\/?$/,
];

export interface BroadPatternResult {
  blocked: boolean;
  reason?: string;
  pattern?: string;
  suggestions?: string[];
}

export function isBroadPattern(pattern: string): boolean {
  if (!pattern || typeof pattern !== "string") return false;
  const normalized = pattern.trim();
  return BROAD_PATTERN_REGEXES.some((re) => re.test(normalized));
}

export function hasSpecificDirectory(pattern: string): boolean {
  if (!pattern) return false;
  for (const dir of SPECIFIC_DIRS) {
    if (pattern.startsWith(`${dir}/`) || pattern.startsWith(`./${dir}/`)) return true;
  }
  const firstSegment = pattern.split("/")[0];
  if (firstSegment && !firstSegment.includes("*") && firstSegment !== ".") return true;
  return false;
}

export function isHighLevelPath(basePath: string | undefined): boolean {
  if (!basePath) return true;
  const normalized = basePath.replace(/\\/g, "/");
  if (HIGH_RISK_INDICATORS.some((re) => re.test(normalized))) return true;
  const segments = normalized.split("/").filter((s) => s && s !== ".");
  if (segments.length <= 1) return true;
  const hasSpecific = SPECIFIC_DIRS.some(
    (dir) =>
      normalized.includes(`/${dir}/`) || normalized.includes(`/${dir}`) ||
      normalized.startsWith(`${dir}/`) || normalized === dir
  );
  return !hasSpecific;
}

export function suggestSpecificPatterns(pattern: string): string[] {
  const suggestions: string[] = [];
  const extMatch = pattern.match(/\*\.(\{[^}]+\}|\w+)$/);
  const ext = extMatch ? extMatch[1] : "";
  const commonDirs = ["src", "lib", "app", "components"];

  if (pattern.includes(".ts") || pattern.includes("{ts")) {
    suggestions.push("src/**/*.ts", "src/**/*.tsx");
  }
  if (pattern.includes(".js") || pattern.includes("{js")) {
    suggestions.push("src/**/*.js", "lib/**/*.js");
  }
  for (const dir of commonDirs) {
    suggestions.push(ext ? `${dir}/**/*.${ext}` : `${dir}/**/*`);
  }
  return suggestions.slice(0, 4);
}

/** Check if a Glob tool call uses an overly broad pattern */
export function detectBroadPatternIssue(toolInput: { pattern?: string; path?: string }): BroadPatternResult {
  if (!toolInput || typeof toolInput !== "object") return { blocked: false };
  const { pattern, path: basePath } = toolInput;
  if (!pattern) return { blocked: false };
  if (hasSpecificDirectory(pattern)) return { blocked: false };
  if (!isBroadPattern(pattern)) return { blocked: false };
  if (!isHighLevelPath(basePath)) return { blocked: false };
  return {
    blocked: true,
    reason: `Pattern '${pattern}' is too broad for ${basePath || "project root"}`,
    pattern,
    suggestions: suggestSpecificPatterns(pattern),
  };
}

/** Format ANSI error message for broad pattern blocks */
export function formatBroadPatternError(result: BroadPatternResult): string {
  const { pattern, suggestions } = result;
  const lines = [
    "",
    "\x1b[36mNOTE:\x1b[0m This is not an error - this block is intentional to optimize context.",
    "",
    "\x1b[31mBLOCKED\x1b[0m: Overly broad glob pattern detected",
    "",
    `  \x1b[33mPattern:\x1b[0m  ${pattern}`,
    `  \x1b[33mReason:\x1b[0m   Would return ALL matching files, filling context`,
    "",
    "  \x1b[34mUse more specific patterns:\x1b[0m",
  ];
  for (const suggestion of suggestions || []) lines.push(`    • ${suggestion}`);
  lines.push("");
  lines.push("  \x1b[2mTip: Target specific directories to avoid context overflow\x1b[0m");
  lines.push("");
  return lines.join("\n");
}
