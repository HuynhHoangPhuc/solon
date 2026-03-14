// TeammateIdle: Inject available task context when teammate goes idle
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { isHookEnabled } from "../lib/config-loader.ts";
import { readInputSync, writeOutput } from "../lib/hook-io.ts";
import type { TeammateIdleInput, TeammateIdleOutput } from "../lib/types.ts";

if (!isHookEnabled("teammate-idle-handler")) process.exit(0);

const TASKS_DIR = path.join(os.homedir(), ".claude", "tasks");

interface TaskFile {
  id: string;
  status: "pending" | "in_progress" | "completed";
  subject?: string;
  blockedBy?: string[];
  owner?: string;
}

function readJson<T>(filePath: string): T | null {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf-8")) as T;
  } catch {
    return null;
  }
}

function getAvailableTasks(teamName: string): {
  pending: number; inProgress: number; completed: number; total: number;
  unblocked: { id: string; subject: string }[];
} | null {
  const taskDir = path.join(TASKS_DIR, teamName);
  try {
    if (!fs.existsSync(taskDir)) return null;
    const files = fs.readdirSync(taskDir).filter((f) => f.endsWith(".json"));
    const tasks = files.map((f) => readJson<TaskFile>(path.join(taskDir, f))).filter(Boolean) as TaskFile[];

    const completedIds = new Set(tasks.filter((t) => t.status === "completed").map((t) => t.id));
    let pending = 0, inProgress = 0, completed = 0;
    const unblocked: { id: string; subject: string }[] = [];

    for (const task of tasks) {
      if (task.status === "completed") { completed++; continue; }
      if (task.status === "in_progress") { inProgress++; continue; }
      if (task.status !== "pending") continue;
      pending++;
      const blockers = task.blockedBy || [];
      if (blockers.every((id) => completedIds.has(id)) && !task.owner) {
        unblocked.push({ id: task.id, subject: task.subject || "" });
      }
    }

    return { pending, inProgress, completed, total: pending + inProgress + completed, unblocked };
  } catch {
    return null;
  }
}

try {
  const payload = readInputSync<TeammateIdleInput>();
  const { teammate_name, team_name } = payload;
  if (!team_name) process.exit(0);

  const taskInfo = getAvailableTasks(team_name);
  const lines = [`## Teammate Idle`, `${teammate_name} is idle.`];

  if (taskInfo) {
    const remaining = taskInfo.pending + taskInfo.inProgress;
    lines.push(`Tasks: ${taskInfo.completed}/${taskInfo.total} done. ${remaining} remaining.`);

    if (taskInfo.unblocked.length > 0) {
      lines.push(`Unblocked & unassigned: ${taskInfo.unblocked.map((t) => `#${t.id} "${t.subject}"`).join(", ")}`);
      lines.push(`Consider assigning work to ${teammate_name} or waking them with a message.`);
    } else if (remaining === 0) {
      lines.push(`No remaining tasks. Consider shutting down ${teammate_name}.`);
    } else {
      lines.push(`All remaining tasks are blocked or assigned. ${teammate_name} may be waiting for dependencies.`);
    }
  }

  writeOutput({
    hookSpecificOutput: {
      hookEventName: "TeammateIdle",
      additionalContext: lines.join("\n"),
    } as TeammateIdleOutput,
  });
} catch {
  process.exit(0);
}
