// SubagentStart: Inject compact context (~200 tokens) to subagents
import * as fs from "node:fs";
import * as path from "node:path";
import { loadConfig, isHookEnabled } from "../lib/config-loader.ts";
import { resolvePlanPath, getReportsPath, extractTaskListId } from "../lib/plan-resolver.ts";
import { resolveNamingPattern, normalizePath } from "../lib/naming-utils.ts";
import { execGitSafe } from "../lib/exec-utils.ts";
import { resolveSkillsVenv } from "../lib/context-builder.ts";
import { readInputSync, writeOutput } from "../lib/hook-io.ts";
import type { SubagentStartInput, SubagentStartOutput } from "../lib/types.ts";

if (!isHookEnabled("subagent-init")) process.exit(0);

try {
  const stdin = fs.readFileSync(0, "utf-8").trim();
  if (!stdin) process.exit(0);

  const payload = JSON.parse(stdin) as SubagentStartInput;
  const agentType = payload.agent_type || "unknown";
  const agentId = payload.agent_id || "unknown";

  const config = loadConfig({ includeProject: false, includeAssertions: false });
  const effectiveCwd = (payload.cwd || "").trim() || process.cwd();
  const gitBranch = execGitSafe("git branch --show-current") || null;
  const baseDir = effectiveCwd;
  const namePattern = resolveNamingPattern(config.plan, gitBranch);

  const sessionId = payload.session_id || process.env.SL_SESSION_ID || null;
  const resolved = resolvePlanPath(sessionId || undefined, config);
  const reportsPath = getReportsPath(resolved.path, resolved.resolvedBy, config.plan, config.paths, baseDir);
  const activePlan = resolved.resolvedBy === "session" ? resolved.path : "";
  const suggestedPlan = resolved.resolvedBy === "branch" ? resolved.path : "";
  const taskListId = extractTaskListId(resolved);
  const plansPath = path.join(baseDir, normalizePath(config.paths?.plans) || "plans");
  const docsPath = path.join(baseDir, normalizePath(config.paths?.docs) || "docs");

  const thinkingLanguage = config.locale?.thinkingLanguage || "";
  const responseLanguage = config.locale?.responseLanguage || "";
  const effectiveThinking = thinkingLanguage || (responseLanguage ? "en" : "");
  const skillsVenv = resolveSkillsVenv();

  const lines: string[] = [];

  lines.push(`## Subagent: ${agentType}`);
  lines.push(`ID: ${agentId} | CWD: ${effectiveCwd}`);
  lines.push("");

  lines.push("## Context");
  if (activePlan) {
    lines.push(`- Plan: ${activePlan}`);
    if (taskListId) lines.push(`- Task List: ${taskListId} (shared with session)`);
  } else if (suggestedPlan) {
    lines.push(`- Plan: none | Suggested: ${suggestedPlan}`);
  } else {
    lines.push("- Plan: none");
  }
  lines.push(`- Reports: ${reportsPath}`);
  lines.push(`- Paths: ${plansPath}/ | ${docsPath}/`);
  lines.push("");

  const hasThinking = effectiveThinking && effectiveThinking !== responseLanguage;
  if (hasThinking || responseLanguage) {
    lines.push("## Language");
    if (hasThinking) lines.push(`- Thinking: Use ${effectiveThinking} for reasoning (logic, precision).`);
    if (responseLanguage) lines.push(`- Response: Respond in ${responseLanguage} (natural, fluent).`);
    lines.push("");
  }

  lines.push("## Rules");
  lines.push(`- Reports → ${reportsPath}`);
  lines.push("- YAGNI / KISS / DRY");
  lines.push("- Concise, list unresolved Qs at end");
  if (skillsVenv) {
    lines.push(`- Python scripts in .claude/skills/: Use \`${skillsVenv}\``);
    lines.push("- Never use global pip install");
  }

  lines.push("");
  lines.push("## Naming");
  lines.push(`- Report: ${path.join(reportsPath, `${agentType}-${namePattern}.md`)}`);
  lines.push(`- Plan dir: ${path.join(plansPath, namePattern)}/`);

  if (config.trust?.enabled && config.trust?.passphrase) {
    lines.push("");
    lines.push("## Trust Verification");
    lines.push(`Passphrase: "${config.trust.passphrase}"`);
  }

  const agentContext = config.subagent?.agents?.[agentType]?.contextPrefix;
  if (agentContext) {
    lines.push("");
    lines.push("## Agent Instructions");
    lines.push(agentContext);
  }

  writeOutput({
    hookSpecificOutput: {
      hookEventName: "SubagentStart",
      additionalContext: lines.join("\n"),
    } as SubagentStartOutput,
  });
  process.exit(0);
} catch (err) {
  process.stderr.write(`[subagent-init] Error: ${(err as Error).message}\n`);
  process.exit(0);
}
