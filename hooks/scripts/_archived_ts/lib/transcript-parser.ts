// Transcript Parser - Extract tool/agent/todo state from session JSONL
// Streams JSONL line-by-line for memory efficiency, keeps last 20 tools / 10 agents

import * as fs from "node:fs";
import * as readline from "node:readline";
import type { TranscriptTool, TranscriptAgent, TranscriptTodo, TranscriptData } from "./types.ts";

/** Extract target path/pattern from tool input */
export function extractTarget(toolName: string, input: Record<string, unknown> | undefined): string | null {
  if (!input) return null;
  switch (toolName) {
    case "Read":
    case "Write":
    case "Edit":
      return (input.file_path as string) ?? (input.path as string) ?? null;
    case "Glob":
    case "Grep":
      return (input.pattern as string) ?? null;
    case "Bash": {
      const cmd = input.command as string | undefined;
      if (!cmd) return null;
      return cmd.length > 30 ? cmd.slice(0, 30) + "..." : cmd;
    }
    default:
      return null;
  }
}

/** Process a single JSONL entry into tool/agent/todo maps */
function processEntry(
  entry: Record<string, unknown>,
  toolMap: Map<string, TranscriptTool>,
  agentMap: Map<string, TranscriptAgent>,
  latestTodos: TranscriptTodo[],
  result: TranscriptData,
): void {
  const timestamp = entry.timestamp ? new Date(entry.timestamp as string) : new Date();

  if (!result.sessionStart && entry.timestamp) {
    result.sessionStart = timestamp;
  }

  const message = entry.message as Record<string, unknown> | undefined;
  const content = message?.content;
  if (!content || !Array.isArray(content)) return;

  for (const block of content) {
    if (block.type === "tool_use" && block.id && block.name) {
      const input = block.input as Record<string, unknown> | undefined;

      if (block.name === "Task") {
        agentMap.set(block.id, {
          id: block.id,
          type: (input?.subagent_type as string) ?? "unknown",
          model: (input?.model as string) ?? null,
          description: (input?.description as string) ?? null,
          status: "running",
          startTime: timestamp,
          endTime: null,
        });
      } else if (block.name === "TodoWrite") {
        if (input?.todos && Array.isArray(input.todos)) {
          latestTodos.length = 0;
          latestTodos.push(...(input.todos as TranscriptTodo[]));
        }
      } else if (block.name === "TaskCreate") {
        if (input?.subject) {
          latestTodos.push({
            content: input.subject as string,
            status: "pending",
            activeForm: (input.activeForm as string) || null,
          });
        }
      } else if (block.name === "TaskUpdate") {
        if (input?.taskId && input?.status) {
          const task = latestTodos.find((t) => t.id === (input.taskId as string));
          if (task) task.status = input.status as string;
        }
      } else {
        toolMap.set(block.id, {
          id: block.id,
          name: block.name,
          target: extractTarget(block.name, input),
          status: "running",
          startTime: timestamp,
          endTime: null,
        });
      }
    }

    if (block.type === "tool_result" && block.tool_use_id) {
      const tool = toolMap.get(block.tool_use_id);
      if (tool) {
        tool.status = block.is_error ? "error" : "completed";
        tool.endTime = timestamp;
      }
      const agent = agentMap.get(block.tool_use_id);
      if (agent) {
        agent.status = "completed";
        agent.endTime = timestamp;
      }
    }
  }
}

/** Parse transcript JSONL file, returning tool/agent/todo state */
export async function parseTranscript(transcriptPath: string): Promise<TranscriptData> {
  const result: TranscriptData = { tools: [], agents: [], todos: [], sessionStart: null };

  if (!transcriptPath || !fs.existsSync(transcriptPath)) return result;

  const toolMap = new Map<string, TranscriptTool>();
  const agentMap = new Map<string, TranscriptAgent>();
  const latestTodos: TranscriptTodo[] = [];

  try {
    const rl = readline.createInterface({
      input: fs.createReadStream(transcriptPath),
      crlfDelay: Infinity,
    });

    for await (const line of rl) {
      if (!line.trim()) continue;
      try {
        processEntry(JSON.parse(line), toolMap, agentMap, latestTodos, result);
      } catch {
        // Skip malformed lines
      }
    }
  } catch {
    // Return partial results on error
  }

  result.tools = Array.from(toolMap.values()).slice(-20);
  result.agents = Array.from(agentMap.values()).slice(-10);
  result.todos = latestTodos;
  return result;
}
