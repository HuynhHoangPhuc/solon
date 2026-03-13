// SubagentStart: Inject team context for Agent Team teammates (agent_id format: name@team-name)
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { isHookEnabled } from "../lib/config-loader.ts";
import { readInputSync, writeOutput } from "../lib/hook-io.ts";
import type { SubagentStartInput, SubagentStartOutput } from "../lib/types.ts";

if (!isHookEnabled("team-context-inject")) process.exit(0);

const TEAMS_DIR = path.join(os.homedir(), ".claude", "teams");
const TASKS_DIR = path.join(os.homedir(), ".claude", "tasks");

interface TeamConfig {
  name?: string;
  members?: { agentId?: string; name?: string; agentType?: string }[];
}

interface TaskFile {
  status?: string;
}

/** Extract team name from agent_id (format: "name@team-name"). Rejects path traversal. */
function extractTeamName(agentId: string): string | null {
  if (!agentId || typeof agentId !== "string") return null;
  const atIdx = agentId.indexOf("@");
  if (atIdx < 1) return null;
  const name = agentId.substring(atIdx + 1);
  if (name.includes("/") || name.includes("\\") || name.includes("..")) return null;
  return name;
}

function readJson<T>(filePath: string): T | null {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf-8")) as T;
  } catch {
    return null;
  }
}

function buildPeerList(config: TeamConfig, currentAgentId: string): string {
  if (!config?.members?.length) return "none";
  const peers = config.members
    .filter((m) => m.agentId !== currentAgentId)
    .map((m) => `${m.name} (${m.agentType})`)
    .join(", ");
  return peers || "none";
}

function buildCkContext(): string[] {
  const ctx: string[] = [];
  const env = process.env;
  if (env.SL_REPORTS_PATH) ctx.push(`Reports: ${env.SL_REPORTS_PATH}`);
  if (env.SL_PLANS_PATH) ctx.push(`Plans: ${env.SL_PLANS_PATH}`);
  if (env.SL_PROJECT_ROOT) ctx.push(`Project: ${env.SL_PROJECT_ROOT}`);
  if (env.SL_NAME_PATTERN) ctx.push(`Naming: ${env.SL_NAME_PATTERN}`);
  if (env.SL_GIT_BRANCH) ctx.push(`Branch: ${env.SL_GIT_BRANCH}`);
  if (env.SL_ACTIVE_PLAN) ctx.push(`Active plan: ${env.SL_ACTIVE_PLAN}`);
  ctx.push("Commits: conventional (feat:, fix:, docs:, refactor:, test:, chore:)");
  return ctx;
}

function summarizeTasks(teamName: string): { pending: number; inProgress: number; completed: number; total: number } | null {
  const taskDir = path.join(TASKS_DIR, teamName);
  try {
    if (!fs.existsSync(taskDir)) return null;
    const files = fs.readdirSync(taskDir).filter((f) => f.endsWith(".json"));
    let pending = 0, inProgress = 0, completed = 0;
    for (const file of files) {
      const task = readJson<TaskFile>(path.join(taskDir, file));
      if (!task?.status) continue;
      if (task.status === "pending") pending++;
      else if (task.status === "in_progress") inProgress++;
      else if (task.status === "completed") completed++;
    }
    return { pending, inProgress, completed, total: files.length };
  } catch {
    return null;
  }
}

try {
  const payload = readInputSync<SubagentStartInput>();
  const agentId = payload.agent_id || "";
  const teamName = extractTeamName(agentId);
  if (!teamName) process.exit(0); // Not a team agent

  const configPath = path.join(TEAMS_DIR, teamName, "config.json");
  const config = readJson<TeamConfig>(configPath);
  if (!config) process.exit(0);

  const peerList = buildPeerList(config, agentId);
  const tasks = summarizeTasks(teamName);
  const lines: string[] = [];

  lines.push("## Team Context");
  lines.push(`Team: ${config.name || teamName}`);
  lines.push(`Your peers: ${peerList}`);
  if (tasks) {
    lines.push(`Task summary: ${tasks.pending} pending, ${tasks.inProgress} in progress, ${tasks.completed} completed`);
  }

  const ckCtx = buildCkContext();
  if (ckCtx.length > 1) { // > 1 means env vars beyond the always-present commit convention line
    lines.push("");
    lines.push("## CK Context");
    lines.push(...ckCtx);
  }

  lines.push("");
  lines.push("Remember: Check TaskList, claim tasks, respect file ownership, use SendMessage to communicate.");

  writeOutput({
    hookSpecificOutput: {
      hookEventName: "SubagentStart",
      additionalContext: lines.join("\n"),
    } as SubagentStartOutput,
  });
  process.exit(0);
} catch {
  process.exit(0);
}
