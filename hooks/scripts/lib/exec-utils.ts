// Consolidated safe exec helpers for hook scripts
// Provides whitelisted git commands + general safe execution
import { execSync, execFileSync } from "node:child_process";

const DEFAULT_TIMEOUT_MS = 5000;

// Whitelisted read-only git commands (execGitSafe only)
const GIT_ALLOWLIST = new Set([
  "git branch --show-current",
  "git rev-parse --abbrev-ref HEAD",
  "git rev-parse --show-toplevel",
]);

/** Execute a shell command safely with timeout */
export function execSafe(
  cmd: string,
  options: { cwd?: string; timeout?: number } = {}
): string | null {
  const { cwd, timeout = DEFAULT_TIMEOUT_MS } = options;
  try {
    return execSync(cmd, {
      encoding: "utf8",
      timeout,
      cwd,
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();
  } catch {
    return null;
  }
}

/** Execute a binary with args safely (avoids shell injection) */
export function execFileSafe(
  binary: string,
  args: string[],
  timeout = DEFAULT_TIMEOUT_MS
): string | null {
  try {
    return execFileSync(binary, args, {
      encoding: "utf8",
      timeout,
      stdio: ["pipe", "pipe", "pipe"],
    }).trim();
  } catch {
    return null;
  }
}

/** Execute a whitelisted git command safely */
export function execGitSafe(cmd: string, cwd?: string): string | null {
  if (!GIT_ALLOWLIST.has(cmd)) return null;
  return execSafe(cmd, { cwd });
}
