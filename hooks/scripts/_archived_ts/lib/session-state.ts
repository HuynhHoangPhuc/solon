// Session state persistence via temp files
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import type { SessionState } from "./types.ts";

export function getSessionTempPath(sessionId: string): string {
  return path.join(os.tmpdir(), `sl-session-${sessionId}.json`);
}

/** Read session state from temp file */
export function readSessionState(sessionId: string): SessionState | null {
  if (!sessionId) return null;
  const tempPath = getSessionTempPath(sessionId);
  try {
    if (!fs.existsSync(tempPath)) return null;
    return JSON.parse(fs.readFileSync(tempPath, "utf8")) as SessionState;
  } catch {
    return null;
  }
}

/** Write session state atomically to temp file */
export function writeSessionState(sessionId: string, state: SessionState): boolean {
  if (!sessionId) return false;
  const tempPath = getSessionTempPath(sessionId);
  const tmpFile = tempPath + "." + Math.random().toString(36).slice(2);
  try {
    fs.writeFileSync(tmpFile, JSON.stringify(state, null, 2));
    fs.renameSync(tmpFile, tempPath);
    return true;
  } catch {
    try { fs.unlinkSync(tmpFile); } catch {}
    return false;
  }
}
