// Statusline - Claude Code custom statusline entry point
// Reads JSON from stdin, renders based on config mode (full/compact/minimal/none)

import { stdin } from "node:process";
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { getGitInfo } from "./lib/git-info-cache.ts";
import { loadConfig } from "./lib/config-loader.ts";
import { parseTranscript } from "./lib/transcript-parser.ts";
import { collapseHome } from "./lib/statusline-utils.ts";
import { renderFull, renderCompact, renderMinimal, type RenderContext } from "./lib/statusline-renderer.ts";

// Buffer constant matching /context output (22.5% of 200k)
const AUTOCOMPACT_BUFFER = 45000;

/** Read all stdin as string */
async function readStdin(): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunks: string[] = [];
    stdin.setEncoding("utf8");
    stdin.on("data", (chunk: string) => chunks.push(chunk));
    stdin.on("end", () => resolve(chunks.join("")));
    stdin.on("error", reject);
  });
}

async function main(): Promise<void> {
  try {
    const input = await readStdin();
    if (!input.trim()) {
      console.error("No input provided");
      process.exit(1);
    }

    const data = JSON.parse(input);

    // Extract basic info
    const rawDir = data.workspace?.current_dir || data.cwd || process.cwd();
    const currentDir = collapseHome(rawDir);
    const modelName = data.model?.display_name || "Claude";

    // Git info (cached 30s)
    const gitInfo = getGitInfo(rawDir);

    // Context window calculation with autocompact buffer
    const usage = data.context_window?.current_usage || {};
    const contextSize = data.context_window?.context_window_size || 0;
    let contextPercent = 0;
    let totalTokens = 0;

    if (contextSize > 0 && contextSize > AUTOCOMPACT_BUFFER) {
      totalTokens = (usage.input_tokens ?? 0) +
        (usage.cache_creation_input_tokens ?? 0) +
        (usage.cache_read_input_tokens ?? 0);
      contextPercent = Math.min(100, Math.round(((totalTokens + AUTOCOMPACT_BUFFER) / contextSize) * 100));
    }

    // Write context data to temp file for hooks to read
    const sessionId = data.session_id;
    if (sessionId && contextSize > 0) {
      try {
        const contextDataPath = path.join(os.tmpdir(), `sl-context-${sessionId}.json`);
        fs.writeFileSync(contextDataPath, JSON.stringify({
          percent: contextPercent,
          tokens: totalTokens,
          size: contextSize,
          usage,
          timestamp: Date.now(),
        }));
      } catch { /* silent */ }
    }

    // Parse transcript for tools/agents/todos
    const transcriptPath = data.transcript_path;
    const transcript = transcriptPath
      ? await parseTranscript(transcriptPath)
      : { tools: [], agents: [], todos: [], sessionStart: null };

    // Read usage limits from cache (written by usage-context-awareness hook)
    let sessionText = "";
    let usagePercent: number | null = null;
    try {
      const usageCachePath = path.join(os.tmpdir(), "sl-usage-limits-cache.json");
      if (fs.existsSync(usageCachePath)) {
        const cache = JSON.parse(fs.readFileSync(usageCachePath, "utf8"));
        if (cache.status === "unavailable") {
          sessionText = "N/A";
        } else {
          const fiveHour = cache.data?.five_hour;
          usagePercent = fiveHour?.utilization ?? null;
          const resetAt = fiveHour?.resets_at;
          if (resetAt) {
            const resetTime = new Date(resetAt);
            const remaining = Math.floor(resetTime.getTime() / 1000) - Math.floor(Date.now() / 1000);
            if (remaining > 0 && remaining < 18000) {
              const rh = Math.floor(remaining / 3600);
              const rm = Math.floor((remaining % 3600) / 60);
              sessionText = `${rh}h ${rm}m until reset`;
            }
          }
        }
      }
    } catch { /* silent */ }

    // Lines changed
    const linesAdded = data.cost?.total_lines_added || 0;
    const linesRemoved = data.cost?.total_lines_removed || 0;

    // Build render context
    const ctx: RenderContext = {
      modelName,
      currentDir,
      gitBranch: gitInfo?.branch || "",
      gitUnstaged: gitInfo?.unstaged || 0,
      gitStaged: gitInfo?.staged || 0,
      gitAhead: gitInfo?.ahead || 0,
      gitBehind: gitInfo?.behind || 0,
      contextPercent,
      sessionText,
      usagePercent,
      linesAdded,
      linesRemoved,
      transcript,
    };

    // Load config for statusline mode
    const config = loadConfig({ includeProject: false, includeAssertions: false, includeLocale: false });
    const statuslineMode = config.statusline || "full";

    // Render and output
    let lines: string[];
    switch (statuslineMode) {
      case "none":
        lines = [""];
        break;
      case "minimal":
        lines = renderMinimal(ctx);
        break;
      case "compact":
        lines = renderCompact(ctx);
        break;
      case "full":
      default:
        lines = renderFull(ctx);
        break;
    }

    for (const line of lines) {
      console.log(line);
    }
  } catch {
    // Fallback: minimal dir line on any error
    console.log("📁 " + (process.cwd() || "unknown"));
  }
}

main().catch(() => {
  console.log("📁 error");
  process.exit(1);
});
