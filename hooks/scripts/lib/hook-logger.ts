// Zero-dependency structured logger for hooks
// Logs to scripts/.logs/hook-log.jsonl (JSON Lines format)
// Auto-creates .logs/ directory and handles rotation (1000 lines → keep last 500)
import * as fs from "node:fs";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { dirname } from "node:path";

const __dirname = dirname(fileURLToPath(import.meta.url));

const LOG_DIR = path.join(__dirname, "..", ".logs");
const LOG_FILE = path.join(LOG_DIR, "hook-log.jsonl");
const MAX_LINES = 1000;
const TRUNCATE_TO = 500;

interface LogData {
  tool?: string;
  dur?: number;
  status?: string;
  exit?: number;
  error?: string;
}

function ensureLogDir(): void {
  try {
    if (!fs.existsSync(LOG_DIR)) {
      fs.mkdirSync(LOG_DIR, { recursive: true });
    }
  } catch (_) {
    // Fail silently — never crash
  }
}

function rotateIfNeeded(): void {
  try {
    if (!fs.existsSync(LOG_FILE)) return;
    const lines = fs.readFileSync(LOG_FILE, "utf-8").split("\n").filter(Boolean);
    if (lines.length >= MAX_LINES) {
      const truncated = lines.slice(-TRUNCATE_TO).join("\n") + "\n";
      fs.writeFileSync(LOG_FILE, truncated, "utf-8");
    }
  } catch (_) {
    // Fail silently
  }
}

/** Log a hook event to JSONL file */
export function logHook(hookName: string, data: LogData): void {
  try {
    ensureLogDir();
    rotateIfNeeded();

    const entry = {
      ts: new Date().toISOString(),
      hook: hookName,
      tool: data.tool || "",
      dur: data.dur || 0,
      status: data.status || "ok",
      exit: data.exit !== undefined ? data.exit : 0,
      error: data.error || "",
    };

    fs.appendFileSync(LOG_FILE, JSON.stringify(entry) + "\n", "utf-8");
  } catch (_) {
    // Never crash — fail silently
  }
}

/** Create a duration timer for a hook */
export function createHookTimer(hookName: string): { end: (data?: LogData) => void } {
  const start = Date.now();
  return {
    end(data: LogData = {}) {
      const dur = Date.now() - start;
      logHook(hookName, { ...data, dur });
    },
  };
}
