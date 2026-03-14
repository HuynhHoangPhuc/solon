// Naming utilities: date formatting, slug sanitization, pattern resolution
import type { PlanConfig } from "./types.ts";

/** Characters invalid in filenames across Windows, macOS, Linux */
const INVALID_FILENAME_CHARS = /[<>:"/\\|?*\x00-\x1f\x7f]/g;

/** Sanitize slug for safe filesystem usage */
export function sanitizeSlug(slug: string): string {
  if (!slug || typeof slug !== "string") return "";
  return slug
    .replace(INVALID_FILENAME_CHARS, "")
    .replace(/[^a-z0-9-]/gi, "-")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 100);
}

/** Format date according to dateFormat config (e.g. YYMMDD-HHmm) */
export function formatDate(format: string): string {
  const now = new Date();
  const pad = (n: number, len = 2) => String(n).padStart(len, "0");

  const tokens: Record<string, string | number> = {
    YYYY: now.getFullYear(),
    YY: String(now.getFullYear()).slice(-2),
    MM: pad(now.getMonth() + 1),
    DD: pad(now.getDate()),
    HH: pad(now.getHours()),
    mm: pad(now.getMinutes()),
    ss: pad(now.getSeconds()),
  };

  let result = format;
  for (const [token, value] of Object.entries(tokens)) {
    result = result.replace(token, String(value));
  }
  return result;
}

/** Normalize path value (trim, remove trailing slashes) */
export function normalizePath(pathValue: string): string | null {
  if (!pathValue || typeof pathValue !== "string") return null;
  let normalized = pathValue.trim();
  if (!normalized) return null;
  normalized = normalized.replace(/[/\\]+$/, "");
  if (!normalized) return null;
  return normalized;
}

/** Validate naming pattern — must contain {slug}, no unresolved placeholders */
export function validateNamingPattern(pattern: string): { valid: boolean; error?: string } {
  if (!pattern || typeof pattern !== "string") {
    return { valid: false, error: "Pattern is empty or not a string" };
  }

  const withoutSlug = pattern.replace(/\{slug\}/g, "").replace(/-+/g, "-").replace(/^-|-$/g, "");
  if (!withoutSlug) {
    return { valid: false, error: "Pattern resolves to empty after removing {slug}" };
  }

  const unresolvedMatch = withoutSlug.match(/\{[^}]+\}/);
  if (unresolvedMatch) {
    return { valid: false, error: `Unresolved placeholder: ${unresolvedMatch[0]}` };
  }

  if (!pattern.includes("{slug}")) {
    return { valid: false, error: "Pattern must contain {slug} placeholder" };
  }

  return { valid: true };
}

/** Extract issue ID from branch name */
export function extractIssueFromBranch(branch: string | null): string | null {
  if (!branch) return null;
  const patterns = [
    /(?:issue|gh|fix|feat|bug)[/-]?(\d+)/i,
    /[/-](\d+)[/-]/,
    /#(\d+)/,
  ];
  for (const pattern of patterns) {
    const match = branch.match(pattern);
    if (match) return match[1];
  }
  return null;
}

/** Format issue ID with prefix from config */
export function formatIssueId(issueId: string | null, planConfig: PlanConfig): string | null {
  if (!issueId) return null;
  return planConfig.issuePrefix ? `${planConfig.issuePrefix}${issueId}` : `#${issueId}`;
}

/**
 * Resolve naming pattern with date and optional issue prefix.
 * Keeps {slug} as placeholder for agents to substitute.
 * Example: "251212-1830-GH-88-{slug}"
 */
export function resolveNamingPattern(planConfig: PlanConfig, gitBranch: string | null): string {
  const { namingFormat, dateFormat, issuePrefix } = planConfig;
  const formattedDate = formatDate(dateFormat);

  const issueId = extractIssueFromBranch(gitBranch);
  const fullIssue = issueId && issuePrefix ? `${issuePrefix}${issueId}` : null;

  let pattern = namingFormat;
  pattern = pattern.replace("{date}", formattedDate);

  if (fullIssue) {
    pattern = pattern.replace("{issue}", fullIssue);
  } else {
    pattern = pattern.replace(/-?\{issue\}-?/, "-").replace(/--+/g, "-");
  }

  pattern = pattern
    .replace(/^-+/, "")
    .replace(/-+$/, "")
    .replace(/-+(\{slug\})/g, "-$1")
    .replace(/(\{slug\})-+/g, "$1-")
    .replace(/--+/g, "-");

  if (process.env.SL_DEBUG) {
    const validation = validateNamingPattern(pattern);
    if (!validation.valid) {
      process.stderr.write(`[naming-utils] Warning: ${validation.error}\n`);
    }
  }

  return pattern;
}
