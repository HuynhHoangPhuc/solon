// Facade for scout-block modules — unified interface for checking blocked paths and broad patterns
import * as path from "node:path";
import { loadPatterns, createMatcher, matchPath } from "./pattern-matcher.ts";
import { extractFromToolInput } from "./path-extractor.ts";
import { detectBroadPatternIssue } from "./broad-pattern-detector.ts";

// Build command allowlist — allowed even if they reference blocked paths
const BUILD_COMMAND_PATTERN =
  /^(npm|pnpm|yarn|bun)\s+([^\s]+\s+)*(run\s+)?(build|test|lint|dev|start|install|ci|add|remove|update|publish|pack|init|create|exec)/;

// Tool commands — JS/TS, Go, Rust, Java, .NET, containers, IaC, Python, Ruby, PHP, Deno, Elixir
const TOOL_COMMAND_PATTERN =
  /^(\.\/)?(npx|pnpx|bunx|tsc|esbuild|vite|webpack|rollup|turbo|nx|jest|vitest|mocha|eslint|prettier|go|cargo|make|mvn|mvnw|gradle|gradlew|dotnet|docker|podman|kubectl|helm|terraform|ansible|bazel|cmake|sbt|flutter|swift|ant|ninja|meson|python3?|pip|uv|deno|bundle|rake|gem|php|composer|ruby|mix|elixir)/;

const VENV_EXECUTABLE_PATTERN = /(^|[\/\\])\.?venv[\/\\](bin|Scripts)[\/\\]/;

const VENV_CREATION_PATTERN =
  /^(python3?|py)\s+(-[\w.]+\s+)*-m\s+venv\s+|^uv\s+venv(\s|$)|^virtualenv\s+/;

export interface ScoutCheckResult {
  blocked: boolean;
  path?: string;
  pattern?: string;
  reason?: string;
  isBroadPattern?: boolean;
  suggestions?: string[];
  isAllowedCommand?: boolean;
}

export interface ScoutCheckOptions {
  ckignorePath?: string;
  claudeDir?: string;
  checkBroadPatterns?: boolean;
}

function stripCommandPrefix(command: string): string {
  if (!command) return command;
  let stripped = command.trim();
  stripped = stripped.replace(/^(\w+=\S+\s+)+/, "");
  stripped = stripped.replace(/^(sudo|env|nice|nohup|time|timeout)\s+/, "");
  stripped = stripped.replace(/^(\w+=\S+\s+)+/, "");
  return stripped.trim();
}

export function isBuildCommand(command: string): boolean {
  if (!command) return false;
  const trimmed = command.trim();
  return BUILD_COMMAND_PATTERN.test(trimmed) || TOOL_COMMAND_PATTERN.test(trimmed);
}

export function isVenvExecutable(command: string): boolean {
  if (!command) return false;
  return VENV_EXECUTABLE_PATTERN.test(command);
}

export function isVenvCreationCommand(command: string): boolean {
  if (!command) return false;
  return VENV_CREATION_PATTERN.test(command.trim());
}

export function isAllowedCommand(command: string): boolean {
  const stripped = stripCommandPrefix(command);
  return isBuildCommand(stripped) || isVenvExecutable(stripped) || isVenvCreationCommand(stripped);
}

export function splitCompoundCommand(command: string): string[] {
  if (!command) return [];
  return command.split(/\s*(?:&&|\|\||;)\s*/).filter((cmd) => cmd && cmd.trim().length > 0);
}

export function unwrapShellExecutor(command: string): string {
  if (!command) return command;
  const match = command.trim().match(/^(?:(?:bash|sh|zsh)\s+-c|eval)\s+["'](.+)["']\s*$/);
  return match ? match[1] : command;
}

/** Check if a tool call accesses blocked directories or uses overly broad patterns */
export function checkScoutBlock(params: {
  toolName: string;
  toolInput: Record<string, unknown>;
  options?: ScoutCheckOptions;
}): ScoutCheckResult {
  const { toolName, options = {} } = params;
  let { toolInput } = params;
  const {
    claudeDir = path.join(process.cwd(), ".claude"),
    checkBroadPatterns = true,
  } = options;

  // Unwrap shell executor wrappers
  if (toolInput.command && typeof toolInput.command === "string") {
    const unwrapped = unwrapShellExecutor(toolInput.command);
    if (unwrapped !== toolInput.command) toolInput = { ...toolInput, command: unwrapped };
  }

  // Split compound commands — check each sub-command independently
  if (toolInput.command && typeof toolInput.command === "string") {
    const subCommands = splitCompoundCommand(toolInput.command);
    const nonAllowed = subCommands.filter((cmd) => !isAllowedCommand(cmd.trim()));
    if (nonAllowed.length === 0) return { blocked: false, isAllowedCommand: true };
    if (nonAllowed.length < subCommands.length) {
      toolInput = { ...toolInput, command: nonAllowed.join(" ; ") };
    }
  }

  // Check for overly broad glob patterns
  if (checkBroadPatterns && (toolName === "Glob" || toolInput.pattern)) {
    const broadResult = detectBroadPatternIssue(
      toolInput as { pattern?: string; path?: string }
    );
    if (broadResult.blocked) {
      return {
        blocked: true,
        isBroadPattern: true,
        pattern: toolInput.pattern as string | undefined,
        reason: broadResult.reason || "Pattern too broad - may fill context with too many files",
        suggestions: broadResult.suggestions || [],
      };
    }
  }

  const resolvedCkignorePath =
    options.slignorePath || path.join(claudeDir, ".slignore");
  const patterns = loadPatterns(resolvedCkignorePath);
  const matcher = createMatcher(patterns);
  const extractedPaths = extractFromToolInput(toolInput);

  if (extractedPaths.length === 0) return { blocked: false };

  for (const extractedPath of extractedPaths) {
    const result = matchPath(matcher, extractedPath);
    if (result.blocked) {
      return {
        blocked: true,
        path: extractedPath,
        pattern: result.pattern,
        reason: `Path matches blocked pattern: ${result.pattern}`,
      };
    }
  }

  return { blocked: false };
}
