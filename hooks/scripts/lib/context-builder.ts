// Main context builder entry points for session reminder injection
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { loadConfig } from "./config-loader.ts";
import { resolvePlanPath, getReportsPath } from "./plan-resolver.ts";
import { resolveNamingPattern, normalizePath } from "./naming-utils.ts";
import { execSafe } from "./exec-utils.ts";
import type { SLConfig } from "./types.ts";
import {
  buildLanguageSection,
  buildSessionSection,
  buildContextSection,
  buildUsageSection,
  buildRulesSection,
  buildModularizationSection,
  buildPathsSection,
  buildPlanContextSection,
  buildNamingSection,
} from "./context-sections.ts";

// ── Path Resolution Helpers ────────────────────────────────────────────────

export function resolveRulesPath(filename: string, configDirName = ".claude"): string | null {
  const localPath = path.join(process.cwd(), configDirName, "rules", filename);
  const globalPath = path.join(os.homedir(), ".claude", "rules", filename);
  if (fs.existsSync(localPath)) return `${configDirName}/rules/${filename}`;
  if (fs.existsSync(globalPath)) return `~/.claude/rules/${filename}`;

  // Backward compat: workflows/ (legacy location)
  const localLegacy = path.join(process.cwd(), configDirName, "workflows", filename);
  const globalLegacy = path.join(os.homedir(), ".claude", "workflows", filename);
  if (fs.existsSync(localLegacy)) return `${configDirName}/workflows/${filename}`;
  if (fs.existsSync(globalLegacy)) return `~/.claude/workflows/${filename}`;

  return null;
}

export function resolveScriptPath(filename: string, configDirName = ".claude"): string | null {
  const localPath = path.join(process.cwd(), configDirName, "scripts", filename);
  const globalPath = path.join(os.homedir(), ".claude", "scripts", filename);
  if (fs.existsSync(localPath)) return `${configDirName}/scripts/${filename}`;
  if (fs.existsSync(globalPath)) return `~/.claude/scripts/${filename}`;
  return null;
}

export function resolveSkillsVenv(configDirName = ".claude"): string | null {
  const isWindows = process.platform === "win32";
  const venvBin = isWindows ? "Scripts" : "bin";
  const pythonExe = isWindows ? "python.exe" : "python3";
  const localVenv = path.join(process.cwd(), configDirName, "skills", ".venv", venvBin, pythonExe);
  const globalVenv = path.join(os.homedir(), ".claude", "skills", ".venv", venvBin, pythonExe);
  if (fs.existsSync(localVenv)) {
    return isWindows ? `${configDirName}\\skills\\.venv\\Scripts\\python.exe` : `${configDirName}/skills/.venv/bin/python3`;
  }
  if (fs.existsSync(globalVenv)) {
    return isWindows ? `~\\.claude\\skills\\.venv\\Scripts\\python.exe` : `~/.claude/skills/.venv/bin/python3`;
  }
  return null;
}

// ── Plan Context Builder ───────────────────────────────────────────────────

interface PlanContext {
  reportsPath: string;
  gitBranch: string | null;
  planLine: string;
  namePattern: string;
  validationMode: string;
  validationMin: number;
  validationMax: number;
}

export function buildPlanContext(sessionId: string | null | undefined, config: SLConfig): PlanContext {
  const { plan, paths } = config;
  const gitBranch = execSafe("git branch --show-current");
  const resolved = resolvePlanPath(sessionId || undefined, config);
  const reportsPath = getReportsPath(resolved.path, resolved.resolvedBy, plan, paths);
  const namePattern = resolveNamingPattern(plan, gitBranch);

  const planLine =
    resolved.resolvedBy === "session"
      ? `- Plan: ${resolved.path}`
      : resolved.resolvedBy === "branch"
        ? `- Plan: none | Suggested: ${resolved.path}`
        : `- Plan: none`;

  const validation = plan.validation || { mode: "prompt", minQuestions: 3, maxQuestions: 8, focusAreas: [] };
  return {
    reportsPath,
    gitBranch,
    planLine,
    namePattern,
    validationMode: validation.mode || "prompt",
    validationMin: validation.minQuestions || 3,
    validationMax: validation.maxQuestions || 8,
  };
}

// ── Injection Guard ────────────────────────────────────────────────────────

/** Check if context was recently injected (prevent duplicate injection) */
export function wasRecentlyInjected(transcriptPath: string): boolean {
  try {
    if (!transcriptPath || !fs.existsSync(transcriptPath)) return false;
    const transcript = fs.readFileSync(transcriptPath, "utf-8");
    return transcript.split("\n").slice(-150).some((line) => line.includes("[IMPORTANT] Consider Modularization"));
  } catch {
    return false;
  }
}

// ── Main Entry Points ──────────────────────────────────────────────────────

interface ReminderParams {
  sessionId?: string | null;
  thinkingLanguage?: string | null;
  responseLanguage?: string | null;
  devRulesPath?: string | null;
  catalogScript?: string | null;
  skillsVenv?: string | null;
  reportsPath: string;
  plansPath: string;
  docsPath: string;
  docsMaxLoc?: number;
  planLine: string;
  gitBranch?: string | null;
  namePattern: string;
  validationMode: string;
  validationMin: number;
  validationMax: number;
  staticEnv?: Record<string, string | undefined>;
  hooks?: Record<string, boolean>;
}

export function buildReminder(params: ReminderParams): string[] {
  const hooksConfig = params.hooks || {};
  const contextEnabled = hooksConfig["context-tracking"] !== false;
  const usageEnabled = hooksConfig["usage-context-awareness"] !== false;

  return [
    ...buildLanguageSection({ thinkingLanguage: params.thinkingLanguage, responseLanguage: params.responseLanguage }),
    ...buildSessionSection(params.staticEnv),
    ...(contextEnabled ? buildContextSection(params.sessionId) : []),
    ...(usageEnabled ? buildUsageSection() : []),
    ...buildRulesSection({ devRulesPath: params.devRulesPath, catalogScript: params.catalogScript, skillsVenv: params.skillsVenv, plansPath: params.plansPath, docsPath: params.docsPath }),
    ...buildModularizationSection(),
    ...buildPathsSection({ reportsPath: params.reportsPath, plansPath: params.plansPath, docsPath: params.docsPath, docsMaxLoc: params.docsMaxLoc }),
    ...buildPlanContextSection({ planLine: params.planLine, reportsPath: params.reportsPath, gitBranch: params.gitBranch, validationMode: params.validationMode, validationMin: params.validationMin, validationMax: params.validationMax }),
    ...buildNamingSection({ reportsPath: params.reportsPath, plansPath: params.plansPath, namePattern: params.namePattern }),
  ];
}

interface BuildReminderContextOptions {
  sessionId?: string | null;
  config?: SLConfig;
  staticEnv?: Record<string, string | undefined>;
  configDirName?: string;
  baseDir?: string | null;
}

/** Build complete reminder context (unified entry point for hooks) */
export function buildReminderContext(options: BuildReminderContextOptions = {}): {
  content: string;
  lines: string[];
  sections: Record<string, string[]>;
} {
  const { sessionId, config, staticEnv, configDirName = ".claude", baseDir } = options;
  const cfg = config || loadConfig({ includeProject: false, includeAssertions: false });

  const devRulesPath = resolveRulesPath("development-rules.md", configDirName);
  const catalogScript = resolveScriptPath("generate_catalogs.py", configDirName);
  const skillsVenv = resolveSkillsVenv(configDirName);
  const planCtx = buildPlanContext(sessionId, cfg);

  const effectiveBaseDir = baseDir || null;
  const plansPathRel = normalizePath(cfg.paths?.plans) || "plans";
  const docsPathRel = normalizePath(cfg.paths?.docs) || "docs";

  const params: ReminderParams = {
    sessionId,
    thinkingLanguage: cfg.locale?.thinkingLanguage,
    responseLanguage: cfg.locale?.responseLanguage,
    devRulesPath,
    catalogScript,
    skillsVenv,
    reportsPath: effectiveBaseDir ? path.join(effectiveBaseDir, planCtx.reportsPath) : planCtx.reportsPath,
    plansPath: effectiveBaseDir ? path.join(effectiveBaseDir, plansPathRel) : plansPathRel,
    docsPath: effectiveBaseDir ? path.join(effectiveBaseDir, docsPathRel) : docsPathRel,
    docsMaxLoc: Math.max(1, parseInt(String(cfg.docs?.maxLoc), 10) || 800),
    planLine: planCtx.planLine,
    gitBranch: planCtx.gitBranch,
    namePattern: planCtx.namePattern,
    validationMode: planCtx.validationMode,
    validationMin: planCtx.validationMin,
    validationMax: planCtx.validationMax,
    staticEnv,
    hooks: cfg.hooks,
  };

  const lines = buildReminder(params);
  const hooksConfig = cfg.hooks || {};

  return {
    content: lines.join("\n"),
    lines,
    sections: {
      language: buildLanguageSection({ thinkingLanguage: params.thinkingLanguage, responseLanguage: params.responseLanguage }),
      session: buildSessionSection(staticEnv),
      context: hooksConfig["context-tracking"] !== false ? buildContextSection(sessionId) : [],
      usage: hooksConfig["usage-context-awareness"] !== false ? buildUsageSection() : [],
      rules: buildRulesSection({ devRulesPath, catalogScript, skillsVenv, plansPath: params.plansPath, docsPath: params.docsPath }),
      modularization: buildModularizationSection(),
      paths: buildPathsSection({ reportsPath: params.reportsPath, plansPath: params.plansPath, docsPath: params.docsPath, docsMaxLoc: params.docsMaxLoc }),
      planContext: buildPlanContextSection(planCtx),
      naming: buildNamingSection({ reportsPath: params.reportsPath, plansPath: params.plansPath, namePattern: params.namePattern }),
    },
  };
}
