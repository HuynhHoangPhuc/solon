// PostToolUse(Edit/Write/MultiEdit): Track edits and remind to run code-simplifier
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { isHookEnabled } from "../lib/config-loader.ts";
import { invalidateCache } from "../lib/git-info-cache.ts";
import { readInputSync, writeOutput } from "../lib/hook-io.ts";
import type { PostToolUseInput } from "../lib/types.ts";

if (!isHookEnabled("post-edit-simplify-reminder")) process.exit(0);

const SESSION_TRACK_FILE = path.join(os.tmpdir(), "sl-simplify-session.json");
const EDIT_THRESHOLD = 5;

interface SessionData {
  startTime: number;
  editCount: number;
  modifiedFiles: string[];
  lastReminder: number;
  simplifierRun: boolean;
}

function initSessionData(): SessionData {
  return { startTime: Date.now(), editCount: 0, modifiedFiles: [], lastReminder: 0, simplifierRun: false };
}

function loadSessionData(): SessionData {
  try {
    if (fs.existsSync(SESSION_TRACK_FILE)) {
      const data = JSON.parse(fs.readFileSync(SESSION_TRACK_FILE, "utf8")) as SessionData;
      // Reset if session is older than 2 hours
      if (Date.now() - data.startTime > 2 * 60 * 60 * 1000) return initSessionData();
      return data;
    }
  } catch {}
  return initSessionData();
}

function saveSessionData(data: SessionData): void {
  try {
    fs.writeFileSync(SESSION_TRACK_FILE, JSON.stringify(data, null, 2));
  } catch {}
}

try {
  const hookData = readInputSync<PostToolUseInput>();
  const editTools = ["Edit", "Write", "MultiEdit"];

  if (!editTools.includes(hookData.tool_name)) {
    writeOutput({ continue: true });
    process.exit(0);
  }

  // Invalidate git cache so statusline shows fresh state
  invalidateCache(hookData.cwd || process.cwd());

  const session = loadSessionData();
  session.editCount++;

  const filePath = (hookData.tool_input as { file_path?: string; path?: string }).file_path
    || (hookData.tool_input as { file_path?: string; path?: string }).path || "";
  if (filePath && !session.modifiedFiles.includes(filePath)) {
    session.modifiedFiles.push(filePath);
  }

  const shouldRemind =
    session.editCount >= EDIT_THRESHOLD &&
    !session.simplifierRun &&
    Date.now() - session.lastReminder > 10 * 60 * 1000;

  let additionalContext: string | undefined;
  if (shouldRemind) {
    session.lastReminder = Date.now();
    additionalContext = `\n\n[Code Simplification Reminder] You have modified ${session.modifiedFiles.length} files in this session. Consider using the \`code-simplifier\` agent to refine recent changes before proceeding to code review. This is a MANDATORY step in the workflow.`;
  }

  saveSessionData(session);
  writeOutput({ continue: true, ...(additionalContext ? { additionalContext } : {}) });
} catch {
  writeOutput({ continue: true });
}
