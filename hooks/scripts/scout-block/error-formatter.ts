// Rich, actionable ANSI error messages for scout-block (Problem + Reason + Solution)
import * as path from "node:path";

export interface BlockedErrorDetails {
  path: string;
  pattern: string;
  tool: string;
  claudeDir?: string;
}

function supportsColor(): boolean {
  if (process.env.NO_COLOR !== undefined) return false;
  if (process.env.FORCE_COLOR !== undefined) return true;
  return process.stderr.isTTY || false;
}

function colorize(text: string, code: string): string {
  if (!supportsColor()) return text;
  return `${code}${text}\x1b[0m`;
}

function formatConfigPath(claudeDir?: string): string {
  return claudeDir ? path.join(claudeDir, ".slignore") : ".claude/.slignore";
}

/** Format a blocked path error with actionable guidance */
export function formatBlockedError(details: BlockedErrorDetails): string {
  const { path: blockedPath, pattern, tool, claudeDir } = details;
  const configPath = formatConfigPath(claudeDir);
  const displayPath = blockedPath.length > 60 ? "..." + blockedPath.slice(-57) : blockedPath;

  const lines = [
    "",
    colorize("NOTE:", "\x1b[36m") + " This is not an error - this block is intentional to optimize context.",
    "",
    colorize("BLOCKED", "\x1b[31m") + `: Access to '${displayPath}' denied`,
    "",
    `  ${colorize("Pattern:", "\x1b[33m")}  ${pattern}`,
    `  ${colorize("Tool:", "\x1b[33m")}     ${tool || "unknown"}`,
    "",
    `  ${colorize("To allow, add to", "\x1b[34m")} ${configPath}:`,
    `    !${pattern}`,
    "",
    `  ${colorize("Config:", "\x1b[2m")} ${configPath}`,
    "",
  ];
  return lines.join("\n");
}
