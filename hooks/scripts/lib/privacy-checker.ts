// Privacy pattern matching logic for sensitive file detection (shared by privacy-block hook)
import * as path from "node:path";
import * as fs from "node:fs";

export const APPROVED_PREFIX = "APPROVED:";

// Safe file patterns — exempt from privacy checks
const SAFE_PATTERNS: RegExp[] = [
  /\.example$/i,
  /\.sample$/i,
  /\.template$/i,
];

// Privacy-sensitive patterns
const PRIVACY_PATTERNS: RegExp[] = [
  /^\.env$/,
  /^\.env\./,
  /\.env$/,
  /\/\.env\./,
  /credentials/i,
  /secrets?\.ya?ml$/i,
  /\.pem$/,
  /\.key$/,
  /id_rsa/,
  /id_ed25519/,
];

export interface PrivacyCheckResult {
  blocked: boolean;
  filePath?: string;
  reason?: string;
  approved?: boolean;
  isBash?: boolean;
  suspicious?: boolean;
  promptData?: PromptData;
}

export interface PromptData {
  type: string;
  file: string;
  basename: string;
  question: {
    header: string;
    text: string;
    options: { label: string; description: string }[];
  };
}

export function isSafeFile(testPath: string): boolean {
  if (!testPath) return false;
  const basename = path.basename(testPath);
  return SAFE_PATTERNS.some((p) => p.test(basename));
}

export function hasApprovalPrefix(testPath: string): boolean {
  return Boolean(testPath && testPath.startsWith(APPROVED_PREFIX));
}

export function stripApprovalPrefix(testPath: string): string {
  return hasApprovalPrefix(testPath) ? testPath.slice(APPROVED_PREFIX.length) : testPath;
}

export function isSuspiciousPath(strippedPath: string): boolean {
  return strippedPath.includes("..") || path.isAbsolute(strippedPath);
}

export function isPrivacySensitive(testPath: string): boolean {
  if (!testPath) return false;
  const cleanPath = stripApprovalPrefix(testPath);
  let normalized = cleanPath.replace(/\\/g, "/");
  try { normalized = decodeURIComponent(normalized); } catch { /* use as-is */ }
  if (isSafeFile(normalized)) return false;
  const basename = path.basename(normalized);
  return PRIVACY_PATTERNS.some((p) => p.test(basename) || p.test(normalized));
}

export function extractPaths(toolInput: Record<string, unknown>): { value: string; field: string }[] {
  const paths: { value: string; field: string }[] = [];
  if (!toolInput) return paths;

  if (toolInput.file_path) paths.push({ value: toolInput.file_path as string, field: "file_path" });
  if (toolInput.path) paths.push({ value: toolInput.path as string, field: "path" });
  if (toolInput.pattern) paths.push({ value: toolInput.pattern as string, field: "pattern" });

  if (toolInput.command && typeof toolInput.command === "string") {
    const cmd = toolInput.command;
    const approvedMatch = cmd.match(/APPROVED:[^\s]+/g) || [];
    approvedMatch.forEach((p) => paths.push({ value: p, field: "command" }));

    if (approvedMatch.length === 0) {
      const envMatch = cmd.match(/\.env[^\s]*/g) || [];
      envMatch.forEach((p) => paths.push({ value: p, field: "command" }));

      const varAssignments = cmd.match(/\w+=[^\s]*\.env[^\s]*/g) || [];
      varAssignments.forEach((a) => {
        const value = a.split("=")[1];
        if (value) paths.push({ value, field: "command" });
      });

      const cmdSubst = cmd.match(/\$\([^)]*?(\.env[^\s)]*)[^)]*\)/g) || [];
      for (const subst of cmdSubst) {
        const inner = subst.match(/\.env[^\s)]*/);
        if (inner) paths.push({ value: inner[0], field: "command" });
      }
    }
  }

  return paths.filter((p) => p.value);
}

export function isPrivacyBlockDisabled(configDir?: string): boolean {
  try {
    const configPath = configDir
      ? path.join(configDir, ".sl.json")
      : path.join(process.cwd(), ".claude", ".sl.json");
    const config = JSON.parse(fs.readFileSync(configPath, "utf8")) as Record<string, unknown>;
    return config.privacyBlock === false;
  } catch {
    return false;
  }
}

export function buildPromptData(filePath: string): PromptData {
  const basename = path.basename(filePath);
  return {
    type: "PRIVACY_PROMPT",
    file: filePath,
    basename,
    question: {
      header: "File Access",
      text: `I need to read "${basename}" which may contain sensitive data (API keys, passwords, tokens). Do you approve?`,
      options: [
        { label: "Yes, approve access", description: `Allow reading ${basename} this time` },
        { label: "No, skip this file", description: "Continue without accessing this file" },
      ],
    },
  };
}

/** Check if a tool call accesses privacy-sensitive files */
export function checkPrivacy(params: {
  toolName: string;
  toolInput: Record<string, unknown>;
  options?: { disabled?: boolean; configDir?: string; allowBash?: boolean };
}): PrivacyCheckResult {
  const { toolName, toolInput, options = {} } = params;
  const { disabled, configDir, allowBash = true } = options;

  if (disabled || isPrivacyBlockDisabled(configDir)) return { blocked: false };

  const isBashTool = toolName === "Bash";
  const paths = extractPaths(toolInput);

  for (const { value: testPath } of paths) {
    if (!isPrivacySensitive(testPath)) continue;

    if (hasApprovalPrefix(testPath)) {
      const strippedPath = stripApprovalPrefix(testPath);
      return { blocked: false, approved: true, filePath: strippedPath, suspicious: isSuspiciousPath(strippedPath) };
    }

    if (isBashTool && allowBash) {
      return { blocked: false, isBash: true, filePath: testPath, reason: `Bash command accesses sensitive file: ${testPath}` };
    }

    return {
      blocked: true,
      filePath: testPath,
      reason: "Sensitive file access requires user approval",
      promptData: buildPromptData(testPath),
    };
  }

  return { blocked: false };
}
