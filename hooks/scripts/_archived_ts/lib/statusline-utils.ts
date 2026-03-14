// Statusline utility functions - visible length, elapsed time, home expansion
import * as os from "node:os";

/**
 * Calculate visible string length (strip ANSI, emoji-aware column width)
 * BUG FIX #3: Iterate codepoints, not UTF-16 code units. Each emoji = 2 cols.
 */
export function visibleLength(str: string): number {
  if (!str || typeof str !== "string") return 0;
  // Strip ANSI escape codes
  const noAnsi = str.replace(/\x1b\[[0-9;]*m/g, "");
  let width = 0;
  for (const char of noAnsi) {
    const cp = char.codePointAt(0)!;
    // SMP emoji ranges (U+1F300–U+1F9FF) and misc symbols (U+2600–U+27BF)
    if (
      (cp >= 0x1f300 && cp <= 0x1f9ff) ||
      (cp >= 0x2600 && cp <= 0x26ff) ||
      (cp >= 0x2700 && cp <= 0x27bf)
    ) {
      width += 2;
    } else {
      width += 1;
    }
  }
  return width;
}

/** Format elapsed time from start to end (or now) */
export function formatElapsed(startTime: Date | string | null, endTime: Date | string | null): string {
  if (!startTime) return "0s";
  const start = startTime instanceof Date ? startTime.getTime() : new Date(startTime).getTime();
  if (isNaN(start)) return "0s";
  const end = endTime
    ? (endTime instanceof Date ? endTime.getTime() : new Date(endTime).getTime())
    : Date.now();
  if (isNaN(end)) return "0s";
  const ms = end - start;
  if (ms < 0 || ms < 1000) return "<1s";
  if (ms < 60000) return `${Math.round(ms / 1000)}s`;
  const mins = Math.floor(ms / 60000);
  const secs = Math.round((ms % 60000) / 1000);
  return `${mins}m ${secs}s`;
}

/** Collapse home directory to ~ */
export function collapseHome(filePath: string): string {
  const homeDir = os.homedir();
  return filePath.startsWith(homeDir) ? filePath.replace(homeDir, "~") : filePath;
}

/** Get terminal width with fallback chain */
export function getTerminalWidth(): number {
  if (process.stderr.columns) return process.stderr.columns;
  if (process.env.COLUMNS) {
    const parsed = parseInt(process.env.COLUMNS, 10);
    if (!isNaN(parsed) && parsed > 0) return parsed;
  }
  return 120;
}
