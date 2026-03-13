// HTTP POST sender with 5-minute error throttling per provider
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

const THROTTLE_FILE = path.join(os.tmpdir(), "sl-noti-throttle.json");
const THROTTLE_DURATION_MS = 5 * 60 * 1000; // 5 minutes

export interface SendResult {
  success: boolean;
  error?: string;
  throttled?: boolean;
}

function loadThrottleState(): Record<string, number> {
  try {
    if (fs.existsSync(THROTTLE_FILE)) {
      return JSON.parse(fs.readFileSync(THROTTLE_FILE, "utf8")) as Record<string, number>;
    }
  } catch (err) {
    process.stderr.write(`[sender] Throttle file corrupted, resetting: ${(err as Error).message}\n`);
  }
  return {};
}

function saveThrottleState(state: Record<string, number>): void {
  try {
    fs.writeFileSync(THROTTLE_FILE, JSON.stringify(state, null, 2), "utf8");
  } catch (err) {
    process.stderr.write(`[sender] Failed to save throttle state: ${(err as Error).message}\n`);
  }
}

function isThrottled(provider: string): boolean {
  const state = loadThrottleState();
  const lastError = state[provider];
  if (!lastError) return false;
  return Date.now() - lastError < THROTTLE_DURATION_MS;
}

function recordError(provider: string): void {
  const state = loadThrottleState();
  state[provider] = Date.now();
  saveThrottleState(state);
}

function clearThrottle(provider: string): void {
  const state = loadThrottleState();
  if (state[provider]) {
    delete state[provider];
    saveThrottleState(state);
  }
}

/** Send HTTP POST request with error throttling */
export async function send(
  provider: string,
  url: string,
  body: unknown,
  headers: Record<string, string> = {}
): Promise<SendResult> {
  if (isThrottled(provider)) return { success: false, throttled: true };

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...headers },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const errorText = await response.text().catch(() => "Unknown error");
      const errorMsg = `HTTP ${response.status}: ${errorText.slice(0, 100)}`;
      recordError(provider);
      process.stderr.write(`[sender] ${provider} failed: ${errorMsg}\n`);
      return { success: false, error: errorMsg };
    }

    clearThrottle(provider);
    return { success: true };
  } catch (err) {
    recordError(provider);
    process.stderr.write(`[sender] ${provider} network error: ${(err as Error).message}\n`);
    return { success: false, error: (err as Error).message };
  }
}
