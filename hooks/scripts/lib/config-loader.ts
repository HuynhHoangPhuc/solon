// Config loading with cascade: DEFAULT -> global (~/.claude/.sl.json) -> local (.claude/.sl.json)
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import type { SLConfig } from "./types.ts";
import { normalizePath } from "./naming-utils.ts";

const LOCAL_CONFIG_PATH = ".claude/.sl.json";
const GLOBAL_CONFIG_PATH = path.join(os.homedir(), ".claude", ".sl.json");

export const DEFAULT_CONFIG: SLConfig = {
  plan: {
    namingFormat: "{date}-{issue}-{slug}",
    dateFormat: "YYMMDD-HHmm",
    issuePrefix: null,
    reportsDir: "reports",
    resolution: {
      order: ["session", "branch"],
      branchPattern: "(?:feat|fix|chore|refactor|docs)/(?:[^/]+/)?(.+)",
    },
    validation: {
      mode: "prompt",
      minQuestions: 3,
      maxQuestions: 8,
      focusAreas: ["assumptions", "risks", "tradeoffs", "architecture"],
    },
  },
  paths: { docs: "docs", plans: "plans" },
  docs: { maxLoc: 800 },
  locale: { thinkingLanguage: null, responseLanguage: null },
  trust: { passphrase: null, enabled: false },
  project: { type: "auto", packageManager: "auto", framework: "auto" },
  skills: { research: { useGemini: true } },
  assertions: [],
  statusline: "full",
  hooks: {
    "session-init": true,
    "subagent-init": true,
    "dev-rules-reminder": true,
    "usage-context-awareness": true,
    "scout-block": true,
    "privacy-block": true,
    "post-edit-simplify-reminder": true,
    "task-completed-handler": true,
    "teammate-idle-handler": true,
  },
  codingLevel: -1,
};

/**
 * Deep merge objects. Arrays replaced entirely.
 * IMPORTANT: Empty objects {} are treated as "inherit from parent" (no override).
 */
export function deepMerge<T extends object>(target: T, source: Partial<T>): T {
  if (!source || typeof source !== "object") return target;
  if (!target || typeof target !== "object") return source as T;

  const result = { ...target } as Record<string, unknown>;
  for (const key of Object.keys(source)) {
    const sourceVal = (source as Record<string, unknown>)[key];
    const targetVal = (target as Record<string, unknown>)[key];

    if (Array.isArray(sourceVal)) {
      result[key] = [...sourceVal];
    } else if (sourceVal !== null && typeof sourceVal === "object") {
      // Empty object = inherit (no override)
      if (Object.keys(sourceVal as object).length === 0) continue;
      result[key] = deepMerge(
        (targetVal as object) || {},
        sourceVal as object
      );
    } else {
      result[key] = sourceVal;
    }
  }
  return result as T;
}

export function loadConfigFromPath(configPath: string): Partial<SLConfig> | null {
  try {
    if (!fs.existsSync(configPath)) return null;
    return JSON.parse(fs.readFileSync(configPath, "utf8")) as Partial<SLConfig>;
  } catch {
    return null;
  }
}

/** Validate and sanitize config paths against project root */
export function sanitizeConfig(config: SLConfig, projectRoot: string): SLConfig {
  const result = { ...config };

  if (result.plan) {
    result.plan = { ...result.plan };
    if (!sanitizePath(result.plan.reportsDir, projectRoot)) {
      result.plan.reportsDir = DEFAULT_CONFIG.plan.reportsDir;
    }
    result.plan.resolution = { ...DEFAULT_CONFIG.plan.resolution, ...result.plan.resolution };
    result.plan.validation = { ...DEFAULT_CONFIG.plan.validation, ...result.plan.validation };
  }

  if (result.paths) {
    result.paths = { ...result.paths };
    if (!sanitizePath(result.paths.docs, projectRoot)) {
      result.paths.docs = DEFAULT_CONFIG.paths.docs;
    }
    if (!sanitizePath(result.paths.plans, projectRoot)) {
      result.paths.plans = DEFAULT_CONFIG.paths.plans;
    }
  }

  return result;
}

/** Sanitize path value — prevents traversal, allows absolute paths */
export function sanitizePath(pathValue: string, projectRoot: string): string | null {
  const normalized = normalizePath(pathValue);
  if (!normalized) return null;
  if (/[\x00]/.test(normalized)) return null;
  if (path.isAbsolute(normalized)) return normalized;

  const resolved = path.resolve(projectRoot, normalized);
  if (!resolved.startsWith(projectRoot + path.sep) && resolved !== projectRoot) return null;
  return normalized;
}

/** Escape shell special characters for env file values */
export function escapeShellValue(str: string): string {
  if (typeof str !== "string") return str;
  return str
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\$/g, "\\$")
    .replace(/`/g, "\\`");
}

/** Append an exported env var to CLAUDE_ENV_FILE */
export function writeEnv(envFile: string | undefined, key: string, value: unknown): void {
  if (envFile && value !== null && value !== undefined) {
    const escaped = escapeShellValue(String(value));
    fs.appendFileSync(envFile, `export ${key}="${escaped}"\n`);
  }
}

interface LoadConfigOptions {
  includeProject?: boolean;
  includeAssertions?: boolean;
  includeLocale?: boolean;
}

/** Load config with cascade: DEFAULT -> global -> local */
export function loadConfig(options: LoadConfigOptions = {}): SLConfig {
  const { includeProject = true, includeAssertions = true, includeLocale = true } = options;
  const projectRoot = process.cwd();

  const globalConfig = loadConfigFromPath(GLOBAL_CONFIG_PATH);
  const localConfig = loadConfigFromPath(LOCAL_CONFIG_PATH);

  if (!globalConfig && !localConfig) {
    return getDefaultConfig(includeProject, includeAssertions, includeLocale);
  }

  try {
    let merged = deepMerge({} as SLConfig, DEFAULT_CONFIG);
    if (globalConfig) merged = deepMerge(merged, globalConfig as SLConfig);
    if (localConfig) merged = deepMerge(merged, localConfig as SLConfig);

    const result: SLConfig = {
      plan: merged.plan || DEFAULT_CONFIG.plan,
      paths: merged.paths || DEFAULT_CONFIG.paths,
      docs: merged.docs || DEFAULT_CONFIG.docs,
      trust: merged.trust || DEFAULT_CONFIG.trust,
      locale: includeLocale ? (merged.locale || DEFAULT_CONFIG.locale) : DEFAULT_CONFIG.locale,
      project: includeProject ? (merged.project || DEFAULT_CONFIG.project) : DEFAULT_CONFIG.project,
      assertions: includeAssertions ? (merged.assertions || []) : [],
      codingLevel: merged.codingLevel ?? -1,
      skills: merged.skills || DEFAULT_CONFIG.skills,
      hooks: merged.hooks || DEFAULT_CONFIG.hooks,
      statusline: merged.statusline || "full",
    };

    return sanitizeConfig(result, projectRoot);
  } catch {
    return getDefaultConfig(includeProject, includeAssertions, includeLocale);
  }
}

export function getDefaultConfig(
  includeProject = true,
  includeAssertions = true,
  includeLocale = true
): SLConfig {
  return {
    plan: { ...DEFAULT_CONFIG.plan },
    paths: { ...DEFAULT_CONFIG.paths },
    docs: { ...DEFAULT_CONFIG.docs },
    trust: { ...DEFAULT_CONFIG.trust },
    locale: includeLocale ? { ...DEFAULT_CONFIG.locale } : DEFAULT_CONFIG.locale,
    project: includeProject ? { ...DEFAULT_CONFIG.project } : DEFAULT_CONFIG.project,
    assertions: includeAssertions ? [] : [],
    codingLevel: -1,
    skills: { ...DEFAULT_CONFIG.skills },
    hooks: { ...DEFAULT_CONFIG.hooks },
    statusline: "full",
  };
}

/** Check if a hook is enabled in config (default: enabled) */
export function isHookEnabled(hookName: string): boolean {
  const config = loadConfig({ includeProject: false, includeAssertions: false, includeLocale: false });
  const hooks = config.hooks || {};
  return hooks[hookName] !== false;
}
