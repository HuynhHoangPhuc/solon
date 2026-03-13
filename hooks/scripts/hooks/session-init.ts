// SessionStart: Initialize session environment, detect project, write env vars, output context
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { loadConfig, writeEnv, isHookEnabled } from "../lib/config-loader.ts";
import { writeSessionState } from "../lib/session-state.ts";
import { resolvePlanPath, getReportsPath, extractTaskListId } from "../lib/plan-resolver.ts";
import { resolveNamingPattern } from "../lib/naming-utils.ts";
import { detectProjectType, detectPackageManager, detectFramework, getPythonVersion, getGitRemoteUrl, getGitBranch, getGitRoot, getCodingLevelStyleName, getCodingLevelGuidelines } from "../lib/project-detector.ts";
import { readInputSync, writeContext } from "../lib/hook-io.ts";
import type { SessionStartInput } from "../lib/types.ts";
import { cleanupOrphanedShadowedSkills, detectAgentTeam } from "./session-init-helpers.ts";

if (!isHookEnabled("session-init")) process.exit(0);

try {
  const shadowedCleanup = cleanupOrphanedShadowedSkills();
  const data = readInputSync<SessionStartInput>();
  const envFile = process.env.CLAUDE_ENV_FILE;
  const source = data.source || "unknown";
  const sessionId = data.session_id || null;
  const config = loadConfig();

  const detections = {
    type: detectProjectType(config.project?.type),
    pm: detectPackageManager(config.project?.packageManager),
    framework: detectFramework(config.project?.framework),
  };

  const resolved = resolvePlanPath(sessionId || undefined, config);

  if (sessionId) {
    writeSessionState(sessionId, {
      sessionOrigin: process.cwd(),
      activePlan: resolved.resolvedBy === "session" ? resolved.path : null,
      suggestedPlan: resolved.resolvedBy === "branch" ? resolved.path : null,
      timestamp: Date.now(),
      source,
    });
  }

  const reportsPath = getReportsPath(resolved.path, resolved.resolvedBy, config.plan, config.paths);
  const taskListId = extractTaskListId(resolved);

  const staticEnv = {
    nodeVersion: process.version,
    pythonVersion: getPythonVersion(),
    osPlatform: process.platform,
    gitUrl: getGitRemoteUrl(),
    gitBranch: getGitBranch(),
    gitRoot: getGitRoot(),
    user: process.env.USERNAME || process.env.USER || process.env.LOGNAME || os.userInfo().username,
    locale: process.env.LANG || "",
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    claudeSettingsDir: path.join(os.homedir(), ".claude"),
  };

  const baseDir = process.cwd();
  const namePattern = resolveNamingPattern(config.plan, staticEnv.gitBranch);

  if (envFile) {
    writeEnv(envFile, "SL_SESSION_ID", sessionId || "");
    writeEnv(envFile, "SL_PLAN_NAMING_FORMAT", config.plan.namingFormat);
    writeEnv(envFile, "SL_PLAN_DATE_FORMAT", config.plan.dateFormat);
    writeEnv(envFile, "SL_PLAN_ISSUE_PREFIX", config.plan.issuePrefix || "");
    writeEnv(envFile, "SL_PLAN_REPORTS_DIR", config.plan.reportsDir);
    writeEnv(envFile, "SL_NAME_PATTERN", namePattern);
    writeEnv(envFile, "SL_ACTIVE_PLAN", resolved.resolvedBy === "session" ? resolved.path : "");
    writeEnv(envFile, "SL_SUGGESTED_PLAN", resolved.resolvedBy === "branch" ? resolved.path : "");
    if (taskListId) writeEnv(envFile, "CLAUDE_CODE_TASK_LIST_ID", taskListId);
    writeEnv(envFile, "SL_GIT_ROOT", staticEnv.gitRoot || "");
    writeEnv(envFile, "SL_REPORTS_PATH", path.join(baseDir, reportsPath));
    writeEnv(envFile, "SL_DOCS_PATH", path.join(baseDir, config.paths.docs));
    writeEnv(envFile, "SL_PLANS_PATH", path.join(baseDir, config.paths.plans));
    writeEnv(envFile, "SL_PROJECT_ROOT", process.cwd());
    writeEnv(envFile, "SL_PROJECT_TYPE", detections.type || "");
    writeEnv(envFile, "SL_PACKAGE_MANAGER", detections.pm || "");
    writeEnv(envFile, "SL_FRAMEWORK", detections.framework || "");
    writeEnv(envFile, "SL_NODE_VERSION", staticEnv.nodeVersion);
    writeEnv(envFile, "SL_PYTHON_VERSION", staticEnv.pythonVersion || "");
    writeEnv(envFile, "SL_OS_PLATFORM", staticEnv.osPlatform);
    writeEnv(envFile, "SL_GIT_URL", staticEnv.gitUrl || "");
    writeEnv(envFile, "SL_GIT_BRANCH", staticEnv.gitBranch || "");
    writeEnv(envFile, "SL_USER", staticEnv.user);
    writeEnv(envFile, "SL_LOCALE", staticEnv.locale);
    writeEnv(envFile, "SL_TIMEZONE", staticEnv.timezone);
    writeEnv(envFile, "SL_CLAUDE_SETTINGS_DIR", staticEnv.claudeSettingsDir);
    if (config.locale?.thinkingLanguage) writeEnv(envFile, "SL_THINKING_LANGUAGE", config.locale.thinkingLanguage);
    if (config.locale?.responseLanguage) writeEnv(envFile, "SL_RESPONSE_LANGUAGE", config.locale.responseLanguage);
    const validation = config.plan?.validation || { mode: "prompt", minQuestions: 3, maxQuestions: 8, focusAreas: [] };
    writeEnv(envFile, "SL_VALIDATION_MODE", validation.mode || "prompt");
    writeEnv(envFile, "SL_VALIDATION_MIN_QUESTIONS", validation.minQuestions || 3);
    writeEnv(envFile, "SL_VALIDATION_MAX_QUESTIONS", validation.maxQuestions || 8);
    writeEnv(envFile, "SL_VALIDATION_FOCUS_AREAS", (validation.focusAreas || []).join(","));
    const codingLevel = config.codingLevel ?? 5;
    writeEnv(envFile, "SL_CODING_LEVEL", codingLevel);
    writeEnv(envFile, "SL_CODING_LEVEL_STYLE", getCodingLevelStyleName(codingLevel));
  }

  const teamInfo = detectAgentTeam();
  if (envFile && teamInfo) {
    writeEnv(envFile, "SL_AGENT_TEAM", teamInfo.teamName);
    writeEnv(envFile, "SL_AGENT_TEAM_MEMBERS", teamInfo.memberCount);
  }

  // Context output
  const contextParts = [`Project: ${detections.type || "unknown"}`];
  if (detections.pm) contextParts.push(`PM: ${detections.pm}`);
  contextParts.push(`Plan naming: ${config.plan.namingFormat}`);
  if (staticEnv.gitRoot && staticEnv.gitRoot !== process.cwd()) contextParts.push(`Root: ${staticEnv.gitRoot}`);
  if (resolved.path) contextParts.push(resolved.resolvedBy === "session" ? `Plan: ${resolved.path}` : `Suggested: ${resolved.path}`);
  writeContext(`Session ${source}. ${contextParts.join(" | ")}\n`);

  if (shadowedCleanup.restored.length > 0 || shadowedCleanup.kept.length > 0) {
    writeContext(`\n[!] SKILL-DEDUP CLEANUP (Issue #422): Recovered orphaned .shadowed/ directory.\n`);
    if (shadowedCleanup.restored.length > 0) writeContext(`Restored ${shadowedCleanup.restored.length} skill(s): ${shadowedCleanup.restored.join(", ")}\n`);
    if (shadowedCleanup.kept.length > 0) writeContext(`[!] Kept ${shadowedCleanup.kept.length} for review (content differs): ${shadowedCleanup.kept.join(", ")}\n`);
  }

  if (teamInfo) {
    writeContext(`[i] Agent Team detected: "${teamInfo.teamName}" (${teamInfo.memberCount} members)\n`);
  }

  if (staticEnv.gitRoot && staticEnv.gitRoot !== process.cwd()) {
    writeContext(`Subdirectory mode: Plans/docs will be created in current directory\n`);
    writeContext(`   Git root: ${staticEnv.gitRoot}\n`);
  }

  if (source === "compact") {
    writeContext(`\nCONTEXT COMPACTED - APPROVAL STATE CHECK:\nIf you were waiting for user approval via AskUserQuestion, you MUST re-confirm before proceeding.\n`);
  }

  const codingLevel = config.codingLevel ?? -1;
  const guidelines = getCodingLevelGuidelines(codingLevel);
  if (guidelines) writeContext(`\n${guidelines}\n`);

  if ((config.assertions || []).length > 0) {
    writeContext(`\nUser Assertions:\n`);
    config.assertions.forEach((a, i) => writeContext(`  ${i + 1}. ${a}\n`));
  }

  process.exit(0);
} catch (err) {
  process.stderr.write(`[session-init] Error: ${(err as Error).message}\n`);
  process.exit(0);
}
