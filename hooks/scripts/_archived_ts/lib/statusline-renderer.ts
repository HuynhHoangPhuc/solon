// Statusline Renderer - Mode-specific render functions (full, compact, minimal)
// BUG FIX #1: No \u00A0 replacement — output regular spaces
// BUG FIX #2: No RESET prefix on output lines

import { green, yellow, red, dim, coloredBar } from "./colors.ts";
import { visibleLength, formatElapsed, getTerminalWidth } from "./statusline-utils.ts";
import type { TranscriptData, TranscriptAgent } from "./types.ts";

export interface RenderContext {
  modelName: string;
  currentDir: string;
  gitBranch: string;
  gitUnstaged: number;
  gitStaged: number;
  gitAhead: number;
  gitBehind: number;
  contextPercent: number;
  sessionText: string;
  usagePercent: number | null;
  linesAdded: number;
  linesRemoved: number;
  transcript: TranscriptData;
}

/** Build usage time string with optional percentage */
function buildUsageString(ctx: RenderContext): string | null {
  if (!ctx.sessionText || ctx.sessionText === "N/A") return null;
  let str = ctx.sessionText.replace(" until reset", " left");
  if (ctx.usagePercent != null) str += ` (${Math.round(ctx.usagePercent)}%)`;
  return str;
}

/** Safe date → epoch ms (0 for invalid) */
function safeGetTime(dateValue: Date | string | null): number {
  if (!dateValue) return 0;
  const time = dateValue instanceof Date ? dateValue.getTime() : new Date(dateValue).getTime();
  return isNaN(time) ? 0 : time;
}

/** Render session lines with responsive wrapping based on terminal width */
function renderSessionLines(ctx: RenderContext): string[] {
  const lines: string[] = [];
  const termWidth = getTerminalWidth();
  const threshold = Math.floor(termWidth * 0.85);

  const dirPart = `📁 ${ctx.currentDir}`;

  let branchPart = "";
  if (ctx.gitBranch) {
    branchPart = `🌿 ${ctx.gitBranch}`;
    const indicators: string[] = [];
    if (ctx.gitUnstaged > 0) indicators.push(`${ctx.gitUnstaged}`);
    if (ctx.gitStaged > 0) indicators.push(`+${ctx.gitStaged}`);
    if (ctx.gitAhead > 0) indicators.push(`${ctx.gitAhead}↑`);
    if (ctx.gitBehind > 0) indicators.push(`${ctx.gitBehind}↓`);
    if (indicators.length > 0) {
      branchPart += ` ${yellow(`(${indicators.join(", ")})`)}`;
    }
  }

  let locationPart = branchPart ? `${dirPart}  ${branchPart}` : dirPart;

  // Session: model + context bar + usage
  let sessionPart = `🤖 ${ctx.modelName}`;
  if (ctx.contextPercent > 0) {
    sessionPart += `  ${coloredBar(ctx.contextPercent, 12)} ${ctx.contextPercent}%`;
  }
  const usageStr = buildUsageString(ctx);
  if (usageStr) {
    sessionPart += `  ⌛ ${usageStr.replace(/\)$/, " used)")}`;
  }

  // Stats (lines changed)
  let statsPart = "";
  if (ctx.linesAdded > 0 || ctx.linesRemoved > 0) {
    statsPart = `📝 ${green(`+${ctx.linesAdded}`)} ${red(`-${ctx.linesRemoved}`)}`;
  }

  const sessionLen = visibleLength(sessionPart);
  const statsLen = visibleLength(statsPart);

  // Layout: session first (Claude Code reads line 1)
  const allOneLine = `${sessionPart}  ${locationPart}  ${statsPart}`;
  const sessionLocation = `${sessionPart}  ${locationPart}`;

  if (visibleLength(allOneLine) <= threshold && statsLen > 0) {
    lines.push(allOneLine);
  } else if (visibleLength(sessionLocation) <= threshold) {
    lines.push(sessionLocation);
    if (statsLen > 0) lines.push(statsPart);
  } else if (sessionLen <= threshold) {
    lines.push(sessionPart);
    lines.push(locationPart);
    if (statsLen > 0) lines.push(statsPart);
  } else {
    lines.push(sessionPart);
    lines.push(dirPart);
    if (branchPart) lines.push(branchPart);
    if (statsLen > 0) lines.push(statsPart);
  }

  return lines;
}

/** Render agents as compact chronological flow with duplicate collapsing */
function renderAgentsLines(transcript: TranscriptData): string[] {
  const { agents } = transcript;
  if (!agents || agents.length === 0) return [];

  const running = agents.filter((a) => a.status === "running");
  const completed = agents.filter((a) => a.status === "completed");

  const allAgents = [...running, ...completed];
  allAgents.sort((a, b) => safeGetTime(a.startTime) - safeGetTime(b.startTime));
  if (allAgents.length === 0) return [];

  // Collapse consecutive duplicate types
  const collapsed: { type: string; status: string; count: number; agents: TranscriptAgent[] }[] = [];
  for (const agent of allAgents) {
    const type = agent.type || "agent";
    const last = collapsed[collapsed.length - 1];
    if (last && last.type === type && last.status === agent.status) {
      last.count++;
      last.agents.push(agent);
    } else {
      collapsed.push({ type, status: agent.status, count: 1, agents: [agent] });
    }
  }

  const toShow = collapsed.slice(-4);
  const flowParts = toShow.map((group) => {
    const icon = group.status === "running" ? yellow("●") : dim("○");
    const suffix = group.count > 1 ? ` ×${group.count}` : "";
    return `${icon} ${group.type}${suffix}`;
  });

  const lines: string[] = [];
  const completedCount = completed.length;
  const flowSuffix = completedCount > 2 ? ` ${dim(`(${completedCount} done)`)}` : "";
  lines.push(flowParts.join(" → ") + flowSuffix);

  // Detail line for running (or last completed) agent
  const detailAgent = running[0] || completed[completed.length - 1];
  if (detailAgent?.description) {
    const desc = detailAgent.description.length > 50
      ? detailAgent.description.slice(0, 47) + "..."
      : detailAgent.description;
    const elapsed = formatElapsed(detailAgent.startTime, detailAgent.endTime);
    const icon = detailAgent.status === "running" ? yellow("▸") : dim("▸");
    lines.push(`   ${icon} ${desc} ${dim(`(${elapsed})`)}`);
  }

  return lines;
}

/** Render todos line (in-progress task with progress counts) */
function renderTodosLine(transcript: TranscriptData): string | null {
  const { todos } = transcript;
  if (!todos || todos.length === 0) return null;

  const inProgress = todos.find((t) => t.status === "in_progress");
  const completedCount = todos.filter((t) => t.status === "completed").length;
  const pendingCount = todos.filter((t) => t.status === "pending").length;
  const total = todos.length;

  if (!inProgress) {
    if (completedCount === total && total > 0) {
      return `${green("✓")} All ${total} todos complete`;
    }
    if (pendingCount > 0) {
      const nextPending = todos.find((t) => t.status === "pending");
      const nextTask = nextPending?.content || "Next task";
      const display = nextTask.length > 40 ? nextTask.slice(0, 37) + "..." : nextTask;
      return `${dim("○")} Next: ${display} ${dim(`(${completedCount} done, ${pendingCount} pending)`)}`;
    }
    return null;
  }

  const displayText = inProgress.activeForm || inProgress.content;
  const display = displayText.length > 50 ? displayText.slice(0, 47) + "..." : displayText;
  return `${yellow("▸")} ${display} ${dim(`(${completedCount} done, ${pendingCount} pending)`)}`;
}

/** Render minimal mode — single emoji-separated line */
export function renderMinimal(ctx: RenderContext): string[] {
  const parts = [`🤖 ${ctx.modelName}`];
  if (ctx.contextPercent > 0) {
    const batteryIcon = ctx.contextPercent > 70 ? red("🔋") : "🔋";
    parts.push(`${batteryIcon} ${ctx.contextPercent}%`);
  }
  const usageStr = buildUsageString(ctx);
  if (usageStr) parts.push(`⏰ ${usageStr}`);
  if (ctx.gitBranch) parts.push(`🌿 ${ctx.gitBranch}`);
  parts.push(`📁 ${ctx.currentDir}`);
  return [parts.join("  ")];
}

/** Render compact mode — 2 lines: session info + location */
export function renderCompact(ctx: RenderContext): string[] {
  let line1 = `🤖 ${ctx.modelName}`;
  if (ctx.contextPercent > 0) {
    line1 += `  ${coloredBar(ctx.contextPercent, 12)} ${ctx.contextPercent}%`;
  }
  const usageStr = buildUsageString(ctx);
  if (usageStr) line1 += `  ⌛ ${usageStr}`;

  let line2 = `📁 ${ctx.currentDir}`;
  if (ctx.gitBranch) line2 += `  🌿 ${ctx.gitBranch}`;

  // BUG FIX #1: No \u00A0 replacement — output regular spaces
  return [line1, line2];
}

/** Render full mode — multi-line with agents and todos */
export function renderFull(ctx: RenderContext): string[] {
  const lines: string[] = [];
  lines.push(...renderSessionLines(ctx));
  lines.push(...renderAgentsLines(ctx.transcript));
  const todosLine = renderTodosLine(ctx.transcript);
  if (todosLine) lines.push(todosLine);
  // BUG FIX #1: No \u00A0 replacement
  // BUG FIX #2: No RESET prefix on output lines
  return lines;
}
