// UserPromptSubmit + PostToolUse: Fetch Claude usage limits from API and write to cache
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { execSync } from "node:child_process";
import { isHookEnabled } from "../lib/config-loader.ts";

if (!isHookEnabled("usage-context-awareness")) process.exit(0);

const USAGE_CACHE_FILE = path.join(os.tmpdir(), "sl-usage-limits-cache.json");
const CACHE_TTL_MS = 60000; // 60s
const FETCH_INTERVAL_MS = 300000; // 5min for PostToolUse
const FETCH_INTERVAL_PROMPT_MS = 60000; // 1min for UserPromptSubmit

interface CacheFile {
  timestamp: number;
  status: string;
  data?: unknown;
}

function getClaudeCredentials(): string | null {
  // macOS: Try Keychain first
  if (os.platform() === "darwin") {
    try {
      const result = execSync('security find-generic-password -s "Claude Code-credentials" -w', {
        timeout: 5000,
        encoding: "utf-8",
        stdio: ["pipe", "pipe", "ignore"],
      }).trim();
      const parsed = JSON.parse(result) as { claudeAiOauth?: { accessToken?: string } };
      if (parsed.claudeAiOauth?.accessToken) return parsed.claudeAiOauth.accessToken;
    } catch {}
  }

  // File-based credentials (Linux/Windows, or macOS fallback)
  const credPath = path.join(os.homedir(), ".claude", ".credentials.json");
  try {
    const parsed = JSON.parse(fs.readFileSync(credPath, "utf-8")) as { claudeAiOauth?: { accessToken?: string } };
    return parsed.claudeAiOauth?.accessToken || null;
  } catch {
    return null;
  }
}

function shouldFetch(isUserPrompt: boolean): boolean {
  const interval = isUserPrompt ? FETCH_INTERVAL_PROMPT_MS : FETCH_INTERVAL_MS;
  try {
    if (fs.existsSync(USAGE_CACHE_FILE)) {
      const cache = JSON.parse(fs.readFileSync(USAGE_CACHE_FILE, "utf-8")) as CacheFile;
      if (Date.now() - cache.timestamp < interval) return false;
    }
  } catch {}
  return true;
}

function writeCache(status: string, data: unknown = null): void {
  fs.writeFileSync(USAGE_CACHE_FILE, JSON.stringify({ timestamp: Date.now(), status, data }));
}

async function fetchAndCacheUsageLimits(): Promise<void> {
  const token = getClaudeCredentials();
  if (!token) {
    writeCache("unavailable");
    return;
  }

  try {
    const response = await fetch("https://api.anthropic.com/api/oauth/usage", {
      method: "GET",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
        "anthropic-beta": "oauth-2025-04-20",
        "User-Agent": "solon/1.0",
      },
    });

    if (!response.ok) {
      writeCache("unavailable");
      return;
    }

    const data = await response.json();
    writeCache("available", data);
  } catch {
    writeCache("unavailable");
  }
}

async function main(): Promise<void> {
  try {
    let inputStr = "";
    try {
      inputStr = fs.readFileSync(0, "utf-8");
    } catch {}

    const input = JSON.parse(inputStr || "{}") as { prompt?: string };
    const isUserPrompt = typeof input.prompt === "string";

    if (shouldFetch(isUserPrompt)) {
      await fetchAndCacheUsageLimits();
    }
  } catch {}

  process.stdout.write(JSON.stringify({ continue: true }));
}

main().catch(() => {
  process.stdout.write(JSON.stringify({ continue: true }));
  process.exit(0);
});
