// TaskCompleted: Log task completion and inject progress context
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { isHookEnabled } from "../lib/config-loader.ts";
import { readInputSync, writeOutput } from "../lib/hook-io.ts";
import type { TaskCompletedInput, TaskCompletedOutput } from "../lib/types.ts";

if (!isHookEnabled("task-completed-handler")) process.exit(0);

const TASKS_DIR = path.join(os.homedir(), ".claude", "tasks");

interface TaskFile {
  id: string;
  status: "pending" | "in_progress" | "completed";
  subject?: string;
}

function readJson<T>(filePath: string): T | null {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf-8")) as T;
  } catch {
    return null;
  }
}

function countTasks(teamName: string): { pending: number; inProgress: number; completed: number; total: number } | null {
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
    return { pending, inProgress, completed, total: pending + inProgress + completed };
  } catch {
    return null;
  }
}

function logCompletion(teamName: string, taskId: string, taskSubject: string, teammateName: string): void {
  const reportsPath = process.env.SL_REPORTS_PATH;
  if (!reportsPath) return;
  const logFile = path.join(reportsPath, `team-${teamName}-completions.md`);
  try {
    fs.mkdirSync(path.dirname(logFile), { recursive: true });
    const timestamp = new Date().toISOString().slice(0, 19).replace("T", " ");
    fs.appendFileSync(logFile, `- [${timestamp}] Task #${taskId} "${taskSubject}" completed by ${teammateName}\n`);
  } catch {}
}

try {
  const payload = readInputSync<TaskCompletedInput>();
  const { task_id, task_subject, teammate_name, team_name } = payload;
  if (!team_name) process.exit(0);

  logCompletion(team_name, task_id, task_subject, teammate_name);

  const counts = countTasks(team_name);
  const lines = [
    `## Task Completed`,
    `Task #${task_id} "${task_subject}" completed by ${teammate_name}.`,
  ];

  if (counts) {
    const remaining = counts.pending + counts.inProgress;
    lines.push(`Progress: ${counts.completed}/${counts.total} done. ${counts.pending} pending, ${counts.inProgress} in progress.`);
    if (remaining === 0) {
      lines.push("");
      lines.push("**All tasks completed.** Consider shutting down teammates and synthesizing results.");
    }
  }

  writeOutput({
    hookSpecificOutput: {
      hookEventName: "TaskCompleted",
      additionalContext: lines.join("\n"),
    } as TaskCompletedOutput,
  });
} catch {
  process.exit(0);
}
