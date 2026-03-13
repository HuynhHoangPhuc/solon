// Git Info Cache - Cross-platform git information batching
// Caches git query results with 30s TTL to reduce process spawns
import { execSync } from "node:child_process";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { createHash } from "node:crypto";

const CACHE_TTL = 30000;
const CACHE_MISS = Symbol("cache_miss");

interface GitInfo {
  branch: string;
  unstaged: number;
  staged: number;
  ahead: number;
  behind: number;
}

interface CacheFile {
  timestamp: number;
  data: GitInfo | null;
}

function execIn(cmd: string, cwd?: string): string {
  try {
    return execSync(cmd, {
      encoding: "utf8",
      stdio: ["pipe", "pipe", "ignore"],
      windowsHide: true,
      cwd: cwd || undefined,
    } as Parameters<typeof execSync>[1]).trim();
  } catch {
    return "";
  }
}

function getCachePath(cwd: string): string {
  const hash = createHash("md5").update(cwd).digest("hex").slice(0, 8);
  return path.join(os.tmpdir(), `sl-git-cache-${hash}.json`);
}

function readCache(cachePath: string): GitInfo | null | typeof CACHE_MISS {
  try {
    const cache = JSON.parse(fs.readFileSync(cachePath, "utf8")) as CacheFile;
    if (Date.now() - cache.timestamp < CACHE_TTL) {
      return cache.data;
    }
  } catch {
    // File missing, corrupted, or expired — cache miss
  }
  return CACHE_MISS;
}

function writeCache(cachePath: string, data: GitInfo | null): void {
  const tmpPath = cachePath + ".tmp";
  try {
    fs.writeFileSync(tmpPath, JSON.stringify({ timestamp: Date.now(), data }));
    fs.renameSync(tmpPath, cachePath);
  } catch {
    try { fs.unlinkSync(tmpPath); } catch {}
  }
}

function countLines(str: string): number {
  if (!str) return 0;
  return str.split("\n").filter((l) => l.trim()).length;
}

function fetchGitInfo(cwd: string): GitInfo | null {
  if (!execIn("git rev-parse --git-dir", cwd)) return null;

  const branch = execIn("git branch --show-current", cwd) || execIn("git rev-parse --short HEAD", cwd);
  const unstaged = countLines(execIn("git diff --name-only", cwd));
  const staged = countLines(execIn("git diff --cached --name-only", cwd));

  let ahead = 0, behind = 0;
  const aheadBehind = execIn("git rev-list --left-right --count @{u}...HEAD", cwd);
  if (aheadBehind) {
    const parts = aheadBehind.split(/\s+/);
    behind = parseInt(parts[0], 10) || 0;
    ahead = parseInt(parts[1], 10) || 0;
  }

  return { branch, unstaged, staged, ahead, behind };
}

/** Get git info with caching (30s TTL) */
export function getGitInfo(cwd = process.cwd()): GitInfo | null {
  const cachePath = getCachePath(cwd);
  const cached = readCache(cachePath);
  if (cached !== CACHE_MISS) return cached as GitInfo | null;

  const data = fetchGitInfo(cwd);
  writeCache(cachePath, data);
  return data;
}

/** Invalidate cache for a directory (call after file changes) */
export function invalidateCache(cwd = process.cwd()): void {
  try { fs.unlinkSync(getCachePath(cwd)); } catch {}
}
