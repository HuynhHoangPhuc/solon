// Context section builders for session reminder injection
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";

const USAGE_CACHE_FILE = path.join(os.tmpdir(), "sl-usage-limits-cache.json");
const WARN_THRESHOLD = 70;
const CRITICAL_THRESHOLD = 90;

export function buildLanguageSection(params: {
  thinkingLanguage?: string | null;
  responseLanguage?: string | null;
}): string[] {
  const { thinkingLanguage, responseLanguage } = params;
  const effectiveThinking = thinkingLanguage || (responseLanguage ? "en" : null);
  const hasThinking = effectiveThinking && effectiveThinking !== responseLanguage;
  const lines: string[] = [];

  if (hasThinking || responseLanguage) {
    lines.push("## Language");
    if (hasThinking) {
      lines.push(`- Thinking: Use ${effectiveThinking} for reasoning (logic, precision).`);
    }
    if (responseLanguage) {
      lines.push(`- Response: Respond in ${responseLanguage} (natural, fluent).`);
    }
    lines.push("");
  }

  return lines;
}

export function buildSessionSection(staticEnv: Record<string, string | undefined> = {}): string[] {
  const memUsed = Math.round(process.memoryUsage().heapUsed / 1024 / 1024);
  const memTotal = Math.round(os.totalmem() / 1024 / 1024);
  const memPercent = Math.round((memUsed / memTotal) * 100);
  const cpuUsage = Math.round((process.cpuUsage().user / 1000000) * 100);
  const cpuSystem = Math.round((process.cpuUsage().system / 1000000) * 100);

  return [
    "## Session",
    `- DateTime: ${new Date().toLocaleString()}`,
    `- CWD: ${staticEnv["cwd"] || process.cwd()}`,
    `- Timezone: ${staticEnv["timezone"] || Intl.DateTimeFormat().resolvedOptions().timeZone}`,
    `- Working directory: ${staticEnv["cwd"] || process.cwd()}`,
    `- OS: ${staticEnv["osPlatform"] || process.platform}`,
    `- User: ${staticEnv["user"] || process.env.USERNAME || process.env.USER}`,
    `- Locale: ${staticEnv["locale"] || process.env.LANG || ""}`,
    `- Memory usage: ${memUsed}MB/${memTotal}MB (${memPercent}%)`,
    `- CPU usage: ${cpuUsage}% user / ${cpuSystem}% system`,
    `- Spawning multiple subagents can cause performance issues, spawn and delegate tasks intelligently based on the available system resources.`,
    `- Remember that each subagent only has 200K tokens in context window, spawn and delegate tasks intelligently to make sure their context windows don't get bloated.`,
    `- IMPORTANT: Include these environment information when prompting subagents to perform tasks.`,
    "",
  ];
}

export function buildContextSection(sessionId?: string | null): string[] {
  if (!sessionId) return [];
  try {
    const contextPath = path.join(os.tmpdir(), `sl-context-${sessionId}.json`);
    if (!fs.existsSync(contextPath)) return [];
    const data = JSON.parse(fs.readFileSync(contextPath, "utf-8")) as {
      timestamp: number;
      tokens: number;
      size: number;
      percent: number;
    };
    if (Date.now() - data.timestamp > 300000) return [];

    const usedK = Math.round(data.tokens / 1000);
    const sizeK = Math.round(data.size / 1000);
    const lines = [
      "## Current Session's Context",
      `- Context: ${data.percent}% used (${usedK}K/${sizeK}K tokens)`,
      "- **NOTE:** Optimize the workflow for token efficiency",
    ];

    if (data.percent >= CRITICAL_THRESHOLD) {
      lines.push("- **CRITICAL:** Context nearly full - consider compaction or being concise, update current phase's status before the compaction.");
    } else if (data.percent >= WARN_THRESHOLD) {
      lines.push("- **WARNING:** Context usage moderate - being concise and optimize token efficiency.");
    }
    lines.push("");
    return lines;
  } catch {
    return [];
  }
}

interface UsageData {
  five_hour?: { utilization?: number; resets_at?: string };
  seven_day?: { utilization?: number };
}

export function readUsageCache(): UsageData | null {
  try {
    if (fs.existsSync(USAGE_CACHE_FILE)) {
      const cache = JSON.parse(fs.readFileSync(USAGE_CACHE_FILE, "utf-8")) as {
        timestamp: number;
        data: UsageData;
      };
      if (Date.now() - cache.timestamp < 300000 && cache.data) return cache.data;
    }
  } catch {}
  return null;
}

export function formatTimeUntilReset(resetAt?: string): string | null {
  if (!resetAt) return null;
  const remaining = Math.floor(new Date(resetAt).getTime() / 1000) - Math.floor(Date.now() / 1000);
  if (remaining <= 0 || remaining > 18000) return null;
  const hours = Math.floor(remaining / 3600);
  const mins = Math.floor((remaining % 3600) / 60);
  return `${hours}h ${mins}m`;
}

export function formatUsagePercent(value: number, label: string): string {
  const pct = Math.round(value);
  if (pct >= CRITICAL_THRESHOLD) return `${label}: ${pct}% [CRITICAL]`;
  if (pct >= WARN_THRESHOLD) return `${label}: ${pct}% [WARNING]`;
  return `${label}: ${pct}%`;
}

export function buildUsageSection(): string[] {
  const usage = readUsageCache();
  if (!usage) return [];

  const parts: string[] = [];
  if (usage.five_hour) {
    if (typeof usage.five_hour.utilization === "number") {
      parts.push(formatUsagePercent(usage.five_hour.utilization, "5h"));
    }
    const timeLeft = formatTimeUntilReset(usage.five_hour.resets_at);
    if (timeLeft) parts.push(`resets in ${timeLeft}`);
  }
  if (usage.seven_day?.utilization != null) {
    parts.push(formatUsagePercent(usage.seven_day.utilization, "7d"));
  }

  if (parts.length === 0) return [];
  return ["## Usage Limits", `- ${parts.join(" | ")}`, ""];
}

export function buildRulesSection(params: {
  devRulesPath?: string | null;
  catalogScript?: string | null;
  skillsVenv?: string | null;
  plansPath?: string;
  docsPath?: string;
}): string[] {
  const { devRulesPath, catalogScript, skillsVenv, plansPath, docsPath } = params;
  const plansRef = plansPath || "plans";
  const docsRef = docsPath || "docs";
  const lines = ["## Rules"];

  if (devRulesPath) lines.push(`- Read and follow development rules: "${devRulesPath}"`);
  lines.push(`- Markdown files are organized in: Plans → "${plansRef}" directory, Docs → "${docsRef}" directory`);
  lines.push(`- **IMPORTANT:** DO NOT create markdown files outside of "${plansRef}" or "${docsRef}" UNLESS the user explicitly requests it.`);
  if (catalogScript) {
    lines.push(`- Activate skills: Run \`python ${catalogScript} --skills\` to generate a skills catalog and analyze it, then activate the relevant skills that are needed for the task during the process.`);
  }
  if (skillsVenv) lines.push(`- Python scripts in .claude/skills/: Use \`${skillsVenv}\``);
  lines.push("- When skills' scripts are failed to execute, always fix them and run again, repeat until success.");
  lines.push("- Follow **YAGNI (You Aren't Gonna Need It) - KISS (Keep It Simple, Stupid) - DRY (Don't Repeat Yourself)** principles");
  lines.push("- Sacrifice grammar for the sake of concision when writing reports.");
  lines.push("- In reports, list any unresolved questions at the end, if any.");
  lines.push("- IMPORTANT: Ensure token consumption efficiency while maintaining high quality.");
  lines.push("");
  return lines;
}

export function buildModularizationSection(): string[] {
  return [
    "## **[IMPORTANT] Consider Modularization:**",
    "- Check existing modules before creating new",
    "- Analyze logical separation boundaries (functions, classes, concerns)",
    "- Prefer kebab-case for JS/TS/Python/shell; respect language conventions (C#/Java use PascalCase, Go/Rust use snake_case)",
    "- Write descriptive code comments",
    "- After modularization, continue with main task",
    "- When not to modularize: Markdown files, plain text files, bash scripts, configuration files, environment variables files, etc.",
    "",
  ];
}

export function buildPathsSection(params: {
  reportsPath: string;
  plansPath: string;
  docsPath: string;
  docsMaxLoc?: number;
}): string[] {
  const { reportsPath, plansPath, docsPath, docsMaxLoc = 800 } = params;
  return [
    "## Paths",
    `Reports: ${reportsPath} | Plans: ${plansPath}/ | Docs: ${docsPath}/ | docs.maxLoc: ${docsMaxLoc}`,
    "",
  ];
}

export function buildPlanContextSection(params: {
  planLine: string;
  reportsPath: string;
  gitBranch?: string | null;
  validationMode: string;
  validationMin: number;
  validationMax: number;
}): string[] {
  const { planLine, reportsPath, gitBranch, validationMode, validationMin, validationMax } = params;
  const lines = ["## Plan Context", planLine, `- Reports: ${reportsPath}`];
  if (gitBranch) lines.push(`- Branch: ${gitBranch}`);
  lines.push(`- Validation: mode=${validationMode}, questions=${validationMin}-${validationMax}`);
  lines.push("");
  return lines;
}

export function buildNamingSection(params: {
  reportsPath: string;
  plansPath: string;
  namePattern: string;
}): string[] {
  const { reportsPath, plansPath, namePattern } = params;
  return [
    "## Naming",
    `- Report: \`${reportsPath}{type}-${namePattern}.md\``,
    `- Plan dir: \`${plansPath}/${namePattern}/\``,
    "- Replace `{type}` with: agent name, report type, or context",
    "- Replace `{slug}` in pattern with: descriptive-kebab-slug",
  ];
}
