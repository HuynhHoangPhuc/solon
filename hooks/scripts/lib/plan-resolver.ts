// Plan path resolution (session > branch matching) and related utilities
import * as fs from "node:fs";
import * as path from "node:path";
import type { SLConfig, PlanResolution } from "./types.ts";
import { readSessionState } from "./session-state.ts";
import { execGitSafe } from "./exec-utils.ts";
import { sanitizeSlug, normalizePath } from "./naming-utils.ts";

/** Extract feature slug from git branch name */
export function extractSlugFromBranch(branch: string | null, pattern?: string): string | null {
  if (!branch) return null;
  const defaultPattern = /(?:feat|fix|chore|refactor|docs)\/(?:[^/]+\/)?(.+)/;
  const regex = pattern ? new RegExp(pattern) : defaultPattern;
  const match = branch.match(regex);
  return match ? sanitizeSlug(match[1]) : null;
}

/** Find most recent plan folder by timestamp prefix */
export function findMostRecentPlan(plansDir: string): string | null {
  try {
    if (!fs.existsSync(plansDir)) return null;
    const entries = fs.readdirSync(plansDir, { withFileTypes: true });
    const planDirs = entries
      .filter((e) => e.isDirectory() && /^\d{6}/.test(e.name))
      .map((e) => e.name)
      .sort()
      .reverse();
    return planDirs.length > 0 ? path.join(plansDir, planDirs[0]) : null;
  } catch {
    return null;
  }
}

/**
 * Resolve active plan path using cascading resolution.
 * - 'session': Explicitly set → ACTIVE (directive)
 * - 'branch': Matched from git branch → SUGGESTED (hint only)
 */
export function resolvePlanPath(sessionId: string | undefined, config: SLConfig): PlanResolution {
  const plansDir = config?.paths?.plans || "plans";
  const resolution = config?.plan?.resolution || {};
  const order = resolution.order || ["session", "branch"];
  const branchPattern = resolution.branchPattern;

  for (const method of order) {
    switch (method) {
      case "session": {
        const state = readSessionState(sessionId || "");
        if (state?.activePlan) {
          let resolvedPath = state.activePlan;
          if (!path.isAbsolute(resolvedPath) && state.sessionOrigin) {
            resolvedPath = path.join(state.sessionOrigin, resolvedPath);
          }
          return { path: resolvedPath, resolvedBy: "session" };
        }
        break;
      }
      case "branch": {
        try {
          const branch = execGitSafe("git branch --show-current");
          const slug = extractSlugFromBranch(branch, branchPattern);
          if (slug && fs.existsSync(plansDir)) {
            const entries = fs
              .readdirSync(plansDir, { withFileTypes: true })
              .filter((e) => e.isDirectory() && e.name.includes(slug));
            if (entries.length > 0) {
              return {
                path: path.join(plansDir, entries[entries.length - 1].name),
                resolvedBy: "branch",
              };
            }
          }
        } catch {
          // Ignore errors reading plans dir
        }
        break;
      }
    }
  }
  return { path: null, resolvedBy: null };
}

/**
 * Get reports path based on plan resolution.
 * Only uses plan-specific path for 'session' resolved plans.
 */
export function getReportsPath(
  planPath: string | null,
  resolvedBy: "session" | "branch" | null,
  planConfig: SLConfig["plan"],
  pathsConfig: SLConfig["paths"],
  baseDir: string | null = null
): string {
  const reportsDir = normalizePath(planConfig?.reportsDir) || "reports";
  const plansDir = normalizePath(pathsConfig?.plans) || "plans";

  const normalizedPlanPath = planPath && resolvedBy === "session" ? normalizePath(planPath) : null;
  let reportPath: string;

  if (normalizedPlanPath) {
    reportPath = `${normalizedPlanPath}/${reportsDir}`;
  } else {
    reportPath = `${plansDir}/${reportsDir}`;
  }

  if (baseDir) return path.join(baseDir, reportPath);
  return reportPath + "/";
}

/** Extract task list ID from plan resolution (session-resolved only) */
export function extractTaskListId(resolved: PlanResolution): string | null {
  if (!resolved || resolved.resolvedBy !== "session" || !resolved.path) return null;
  return path.basename(resolved.path);
}
