// ANSI Terminal Colors - Cross-platform color support
// Supports NO_COLOR, FORCE_COLOR, COLORTERM auto-detection

const RESET = "\x1b[0m";
const DIM = "\x1b[2m";
const RED = "\x1b[31m";
const GREEN = "\x1b[32m";
const YELLOW = "\x1b[33m";
const MAGENTA = "\x1b[35m";
const CYAN = "\x1b[36m";

// Detect color support at module load (cached)
// Claude Code statusline runs via pipe but output displays in TTY - default to true
export const shouldUseColor = (() => {
  if (process.env.NO_COLOR) return false;
  if (process.env.FORCE_COLOR) return true;
  return true; // Default true for statusline context
})();

// Detect 256-color support via COLORTERM
export const has256Color = (() => {
  const ct = process.env.COLORTERM;
  return ct === "truecolor" || ct === "24bit" || ct === "256color";
})();

function colorize(text: string | number, code: string): string {
  if (!shouldUseColor) return String(text);
  return `${code}${text}${RESET}`;
}

export function green(text: string): string { return colorize(text, GREEN); }
export function yellow(text: string): string { return colorize(text, YELLOW); }
export function red(text: string): string { return colorize(text, RED); }
export function cyan(text: string): string { return colorize(text, CYAN); }
export function magenta(text: string): string { return colorize(text, MAGENTA); }
export function dim(text: string): string { return colorize(text, DIM); }

/** Get ANSI color code based on context percentage threshold */
export function getContextColor(percent: number): string {
  if (percent >= 85) return RED;
  if (percent >= 70) return YELLOW;
  return GREEN;
}

/** Generate colored progress bar for context window usage */
export function coloredBar(percent: number, width = 12): string {
  const clamped = Math.max(0, Math.min(100, percent));
  const filled = Math.round((clamped / 100) * width);
  const empty = width - filled;

  if (!shouldUseColor) {
    return "▰".repeat(filled) + "▱".repeat(empty);
  }

  const color = getContextColor(percent);
  return `${color}${"▰".repeat(filled)}${DIM}${"▱".repeat(empty)}${RESET}`;
}

export { RESET };
