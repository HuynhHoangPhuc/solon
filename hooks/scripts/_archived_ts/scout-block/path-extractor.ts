// Extract file paths from Claude Code tool inputs (Read, Glob, Bash, etc.)

// Flags indicating the following value should NOT be checked as a path
const EXCLUDE_FLAGS = [
  "--exclude", "--ignore", "--skip", "--prune",
  "-x", "-path", "--exclude-dir",
];

// Filesystem commands where bare directory names should be extracted as paths
export const FILESYSTEM_COMMANDS = [
  "cd", "ls", "cat", "head", "tail", "less", "more",
  "rm", "cp", "mv", "find", "touch", "mkdir", "rmdir",
  "stat", "file", "du", "tree", "chmod", "chown", "ln",
  "readlink", "realpath", "wc", "tee", "tar", "zip", "unzip",
  "open", "code", "vim", "nano", "bat", "rsync", "scp", "diff",
];

// Common blocked directory names — keep in sync with DEFAULT_PATTERNS in pattern-matcher.ts
export const BLOCKED_DIR_NAMES = [
  "node_modules", "__pycache__", ".git", "dist", "build",
  ".next", ".nuxt", ".venv", "venv", "vendor", "target", "coverage",
];

/** Extract all paths from a tool_input object */
export function extractFromToolInput(toolInput: Record<string, unknown>): string[] {
  const paths: string[] = [];
  if (!toolInput || typeof toolInput !== "object") return paths;

  for (const param of ["file_path", "path", "pattern"] as const) {
    if (toolInput[param] && typeof toolInput[param] === "string") {
      const normalized = normalizeExtractedPath(toolInput[param] as string);
      if (normalized) paths.push(normalized);
    }
  }

  if (toolInput.command && typeof toolInput.command === "string") {
    paths.push(...extractFromCommand(toolInput.command));
  }

  return paths.filter(Boolean);
}

/** Extract path-like segments from a Bash command string */
export function extractFromCommand(command: string): string[] {
  if (!command || typeof command !== "string") return [];

  const paths: string[] = [];

  // Extract quoted strings first (preserve spaces in paths)
  const quotedPattern = /["']([^"']+)["']/g;
  let match: RegExpExecArray | null;
  while ((match = quotedPattern.exec(command)) !== null) {
    const content = match[1];
    if (/^s[\/|@#,]/.test(content)) continue; // Skip sed/awk regex
    if (looksLikePath(content)) paths.push(normalizeExtractedPath(content));
  }

  const withoutQuotes = command.replace(/["'][^"']*["']/g, " ");
  const tokens = withoutQuotes.split(/\s+/).filter(Boolean);

  let commandName: string | null = null;
  let isFsCommand = false;
  let skipNextToken = false;
  let heredocDelimiter: string | null = null;
  let nextIsHeredocDelimiter = false;

  for (const token of tokens) {
    if (nextIsHeredocDelimiter) {
      heredocDelimiter = token.replace(/^['"]/, "").replace(/['"]$/, "");
      nextIsHeredocDelimiter = false;
      continue;
    }
    if (heredocDelimiter) {
      if (token === heredocDelimiter) heredocDelimiter = null;
      continue;
    }
    if (token.startsWith("<<") && token.length > 2) {
      heredocDelimiter = token.replace(/^<<-?['"]?/, "").replace(/['"]?$/, "");
      continue;
    }
    if (token === "<<" || token === "<<-") {
      nextIsHeredocDelimiter = true;
      continue;
    }
    if (skipNextToken) { skipNextToken = false; continue; }
    if (token === "&&" || token === ";" || token.startsWith("|")) {
      commandName = null; isFsCommand = false; continue;
    }
    if (isSkippableToken(token)) {
      if (EXCLUDE_FLAGS.includes(token)) skipNextToken = true;
      continue;
    }
    if (commandName === null) {
      commandName = token.toLowerCase();
      isFsCommand = FILESYSTEM_COMMANDS.includes(commandName);
      if (isCommandKeyword(token) || isFsCommand) continue;
    }
    if (isFsCommand && isBlockedDirName(token)) {
      paths.push(normalizeExtractedPath(token));
      continue;
    }
    if (isCommandKeyword(token)) continue;
    if (looksLikePath(token)) paths.push(normalizeExtractedPath(token));
  }

  return paths;
}

export function isBlockedDirName(token: string): boolean {
  return BLOCKED_DIR_NAMES.includes(token);
}

export function looksLikePath(str: string): boolean {
  if (!str || str.length < 2) return false;
  if (str.includes("/") || str.includes("\\")) return true;
  if (str.startsWith("./") || str.startsWith("../")) return true;
  if (/\.\w{1,6}$/.test(str)) return true;
  if (/^[a-zA-Z0-9_-]+\//.test(str)) return true;
  return false;
}

export function isSkippableToken(token: string): boolean {
  if (token.startsWith("-")) return true;
  if (["||", "&&", ">", ">>", "<", "<<", "&", ";"].includes(token)) return true;
  if (token.startsWith("|") || token.startsWith(">") || token.startsWith("<")) return true;
  if (token.startsWith("&")) return true;
  if (/^\d+$/.test(token)) return true;
  return false;
}

export function isCommandKeyword(token: string): boolean {
  const keywords = [
    "echo", "cat", "ls", "cd", "rm", "cp", "mv", "find", "grep", "head", "tail",
    "wc", "du", "tree", "touch", "mkdir", "rmdir", "pwd", "which", "env", "export",
    "source", "bash", "sh", "zsh", "true", "false", "test", "xargs", "tee", "sort",
    "uniq", "cut", "tr", "sed", "awk", "diff", "chmod", "chown", "ln", "file",
    "npm", "pnpm", "yarn", "bun", "npx", "pnpx", "bunx", "node",
    "run", "build", "test", "lint", "dev", "start", "install", "ci", "exec",
    "add", "remove", "update", "publish", "pack", "init", "create",
    "tsc", "esbuild", "vite", "webpack", "rollup", "turbo", "nx",
    "jest", "vitest", "mocha", "eslint", "prettier",
    "git", "commit", "push", "pull", "merge", "rebase", "checkout", "branch",
    "status", "log", "diff", "add", "reset", "stash", "fetch", "clone",
    "docker", "compose", "up", "down", "ps", "logs", "exec", "container", "image",
    "sudo", "time", "timeout", "watch", "make", "cargo", "python", "python3", "pip",
    "ruby", "gem", "go", "rust", "java", "javac", "mvn", "gradle",
  ];
  return keywords.includes(token.toLowerCase());
}

export function normalizeExtractedPath(p: string): string {
  if (!p) return "";
  let normalized = p.trim();
  if (
    (normalized.startsWith('"') && normalized.endsWith('"')) ||
    (normalized.startsWith("'") && normalized.endsWith("'"))
  ) {
    normalized = normalized.slice(1, -1);
  }
  normalized = normalized.replace(/^[`({\[]+/, "").replace(/[`)};\]]+$/, "");
  normalized = normalized.replace(/\\/g, "/");
  if (normalized.endsWith("/") && normalized.length > 1) {
    normalized = normalized.slice(0, -1);
  }
  return normalized;
}
