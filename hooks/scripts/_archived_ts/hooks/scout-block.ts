// PreToolUse: Block access to directories listed in .slignore + overly broad glob patterns
import * as fs from "node:fs";
import * as path from "node:path";
import { fileURLToPath } from "node:url";
import { isHookEnabled } from "../lib/config-loader.ts";
import { createHookTimer } from "../lib/hook-logger.ts";
import { checkScoutBlock } from "../scout-block/scout-checker.ts";
import { formatBlockedError } from "../scout-block/error-formatter.ts";
import { formatBroadPatternError } from "../scout-block/broad-pattern-detector.ts";

if (!isHookEnabled("scout-block")) process.exit(0);

const __dirname = path.dirname(fileURLToPath(import.meta.url));

try {
  const timer = createHookTimer("scout-block");
  const hookInput = fs.readFileSync(0, "utf-8");

  if (!hookInput || hookInput.trim().length === 0) {
    process.stderr.write("ERROR: Empty input\n");
    timer.end({ status: "error", exit: 2 });
    process.exit(2);
  }

  let data: { tool_input?: Record<string, unknown>; tool_name?: string };
  try {
    data = JSON.parse(hookInput);
  } catch {
    process.stderr.write("WARN: JSON parse failed, allowing operation\n");
    timer.end({ status: "ok", exit: 0 });
    process.exit(0);
  }

  if (!data.tool_input || typeof data.tool_input !== "object") {
    process.stderr.write("WARN: Invalid JSON structure, allowing operation\n");
    timer.end({ status: "ok", exit: 0 });
    process.exit(0);
  }

  const toolInput = data.tool_input;
  const toolName = data.tool_name || "unknown";
  // Go up from hooks/scripts/hooks/ → hooks/scripts/ → hooks/ → project root with .slignore
  const claudeDir = path.resolve(__dirname, "..", "..", "..");

  const result = checkScoutBlock({
    toolName,
    toolInput,
    options: {
      claudeDir,
      ckignorePath: path.join(claudeDir, ".slignore"),
      checkBroadPatterns: true,
    },
  });

  if (result.isAllowedCommand) {
    timer.end({ tool: toolName, status: "ok", exit: 0 });
    process.exit(0);
  }

  if (result.blocked && result.isBroadPattern) {
    const errorMsg = formatBroadPatternError({
      blocked: true,
      reason: result.reason,
      pattern: result.pattern,
      suggestions: result.suggestions,
    });
    process.stderr.write(errorMsg);
    timer.end({ tool: toolName, status: "block", exit: 2 });
    process.exit(2);
  }

  if (result.blocked) {
    const errorMsg = formatBlockedError({
      path: result.path!,
      pattern: result.pattern!,
      tool: toolName,
      claudeDir,
    });
    process.stderr.write(errorMsg);
    timer.end({ tool: toolName, status: "block", exit: 2 });
    process.exit(2);
  }

  timer.end({ tool: toolName, status: "ok", exit: 0 });
  process.exit(0);
} catch (err) {
  process.stderr.write(`WARN: Hook error, allowing operation - ${(err as Error).message}\n`);
  process.exit(0);
}
