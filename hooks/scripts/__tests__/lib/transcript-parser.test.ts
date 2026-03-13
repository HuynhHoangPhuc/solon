import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { writeFileSync, unlinkSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { extractTarget, parseTranscript } from "../../lib/transcript-parser.ts";

describe("extractTarget", () => {
  describe("Read tool", () => {
    it("extracts file_path for Read", () => {
      const input = { file_path: "/home/user/file.txt" };
      expect(extractTarget("Read", input)).toBe("/home/user/file.txt");
    });

    it("extracts path field if file_path missing", () => {
      const input = { path: "/home/user/file.txt" };
      expect(extractTarget("Read", input)).toBe("/home/user/file.txt");
    });

    it("returns null for Read without target", () => {
      expect(extractTarget("Read", {})).toBeNull();
      expect(extractTarget("Read", undefined)).toBeNull();
    });
  });

  describe("Write/Edit tools", () => {
    it("extracts file_path for Write", () => {
      const input = { file_path: "/home/user/new.ts" };
      expect(extractTarget("Write", input)).toBe("/home/user/new.ts");
    });

    it("extracts file_path for Edit", () => {
      const input = { file_path: "/home/user/edit.ts" };
      expect(extractTarget("Edit", input)).toBe("/home/user/edit.ts");
    });
  });

  describe("Glob/Grep tools", () => {
    it("extracts pattern for Glob", () => {
      const input = { pattern: "**/*.ts" };
      expect(extractTarget("Glob", input)).toBe("**/*.ts");
    });

    it("extracts pattern for Grep", () => {
      const input = { pattern: "function\\s+\\w+" };
      expect(extractTarget("Grep", input)).toBe("function\\s+\\w+");
    });

    it("returns null for pattern tools without pattern", () => {
      expect(extractTarget("Glob", {})).toBeNull();
      expect(extractTarget("Grep", undefined)).toBeNull();
    });
  });

  describe("Bash tool", () => {
    it("extracts command for Bash", () => {
      const input = { command: "git status" };
      expect(extractTarget("Bash", input)).toBe("git status");
    });

    it("truncates long commands to 30 chars + ellipsis", () => {
      const longCmd = "find . -type f -name '*.js' -exec grep -l 'import' {} \\;";
      const result = extractTarget("Bash", { command: longCmd });
      expect(result).toBe(longCmd.slice(0, 30) + "...");
      expect(result).toHaveLength(33);
    });

    it("returns short command as-is", () => {
      const cmd = "ls -la";
      expect(extractTarget("Bash", { command: cmd })).toBe(cmd);
    });
  });

  describe("Unknown tools", () => {
    it("returns null for unknown tool", () => {
      expect(extractTarget("UnknownTool", { foo: "bar" })).toBeNull();
    });
  });
});

describe("parseTranscript", () => {
  let tmpFile: string;

  beforeEach(() => {
    tmpFile = join(tmpdir(), `test-transcript-${Date.now()}.jsonl`);
  });

  afterEach(() => {
    if (existsSync(tmpFile)) unlinkSync(tmpFile);
  });

  it("returns empty result for non-existent file", async () => {
    const result = await parseTranscript("/non/existent/file.jsonl");
    expect(result.tools).toHaveLength(0);
    expect(result.agents).toHaveLength(0);
    expect(result.todos).toHaveLength(0);
    expect(result.sessionStart).toBeNull();
  });

  it("returns empty result for empty path", async () => {
    const result = await parseTranscript("");
    expect(result.tools).toHaveLength(0);
  });

  it("parses valid tool_use and tool_result entries", async () => {
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "tool-1",
              name: "Read",
              input: { file_path: "/home/user/file.txt" },
            },
            {
              type: "tool_result",
              tool_use_id: "tool-1",
              is_error: false,
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.tools).toHaveLength(1);
    expect(result.tools[0].id).toBe("tool-1");
    expect(result.tools[0].name).toBe("Read");
    expect(result.tools[0].status).toBe("completed");
    expect(result.tools[0].target).toBe("/home/user/file.txt");
  });

  it("tracks tool status from tool_result is_error", async () => {
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "err-tool",
              name: "Bash",
              input: { command: "false" },
            },
            {
              type: "tool_result",
              tool_use_id: "err-tool",
              is_error: true,
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.tools[0].status).toBe("error");
  });

  it("parses Task tool_use as agent", async () => {
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "agent-1",
              name: "Task",
              input: {
                subagent_type: "tester",
                model: "claude-3-haiku",
                description: "Test the implementation",
              },
            },
            {
              type: "tool_result",
              tool_use_id: "agent-1",
              is_error: false,
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.agents).toHaveLength(1);
    expect(result.agents[0].id).toBe("agent-1");
    expect(result.agents[0].type).toBe("tester");
    expect(result.agents[0].model).toBe("claude-3-haiku");
    expect(result.agents[0].status).toBe("completed");
  });

  it("tracks agent status from tool_result", async () => {
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "running-agent",
              name: "Task",
              input: { subagent_type: "reviewer" },
            },
          ],
        },
      }),
      JSON.stringify({
        timestamp: "2026-03-13T12:05:00Z",
        message: {
          content: [
            {
              type: "tool_result",
              tool_use_id: "running-agent",
              is_error: false,
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.agents[0].status).toBe("completed");
    expect(result.agents[0].endTime).not.toBeNull();
  });

  it("parses TodoWrite as full todo replacement", async () => {
    const todos = [
      { content: "Task 1", status: "pending" },
      { content: "Task 2", status: "pending" },
    ];
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "todo-1",
              name: "TodoWrite",
              input: { todos },
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.todos).toHaveLength(2);
    expect(result.todos[0].content).toBe("Task 1");
  });

  it("parses TaskCreate to add new todo", async () => {
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "task-create",
              name: "TaskCreate",
              input: { subject: "New task" },
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.todos).toHaveLength(1);
    expect(result.todos[0].content).toBe("New task");
    expect(result.todos[0].status).toBe("pending");
  });

  it("parses TaskUpdate to modify todo status", async () => {
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "todo-write",
              name: "TodoWrite",
              input: {
                todos: [{ id: "task-1", content: "Task", status: "pending" }],
              },
            },
          ],
        },
      }),
      JSON.stringify({
        timestamp: "2026-03-13T12:10:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "task-update",
              name: "TaskUpdate",
              input: { taskId: "task-1", status: "in_progress" },
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.todos[0].status).toBe("in_progress");
  });

  it("skips malformed JSONL lines without crashing", async () => {
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "valid",
              name: "Bash",
              input: { command: "echo test" },
            },
          ],
        },
      }),
      "this is not valid json",
      JSON.stringify({
        timestamp: "2026-03-13T12:01:00Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "another-valid",
              name: "Read",
              input: { file_path: "/file.txt" },
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.tools).toHaveLength(2);
    expect(result.tools[0].id).toBe("valid");
    expect(result.tools[1].id).toBe("another-valid");
  });

  it("keeps only last 20 tools", async () => {
    const lines = [];
    for (let i = 0; i < 25; i++) {
      lines.push(
        JSON.stringify({
          timestamp: "2026-03-13T12:00:00Z",
          message: {
            content: [
              {
                type: "tool_use",
                id: `tool-${i}`,
                name: "Read",
                input: { file_path: `/file-${i}.txt` },
              },
            ],
          },
        })
      );
    }
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.tools).toHaveLength(20);
    expect(result.tools[0].id).toBe("tool-5");
    expect(result.tools[19].id).toBe("tool-24");
  });

  it("keeps only last 10 agents", async () => {
    const lines = [];
    for (let i = 0; i < 15; i++) {
      lines.push(
        JSON.stringify({
          timestamp: "2026-03-13T12:00:00Z",
          message: {
            content: [
              {
                type: "tool_use",
                id: `agent-${i}`,
                name: "Task",
                input: { subagent_type: "tester" },
              },
            ],
          },
        })
      );
    }
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.agents).toHaveLength(10);
    expect(result.agents[0].id).toBe("agent-5");
    expect(result.agents[9].id).toBe("agent-14");
  });

  it("captures sessionStart from first timestamp", async () => {
    const startTime = "2026-03-13T12:00:00Z";
    const lines = [
      JSON.stringify({
        timestamp: startTime,
        message: {
          content: [
            {
              type: "tool_use",
              id: "tool-1",
              name: "Read",
              input: { file_path: "/file.txt" },
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.sessionStart).not.toBeNull();
    expect(result.sessionStart?.toISOString()).toMatch(/^2026-03-13T12:00:00/);
  });

  it("handles missing message.content gracefully", async () => {
    const lines = [
      JSON.stringify({
        timestamp: "2026-03-13T12:00:00Z",
        message: { content: null },
      }),
      JSON.stringify({
        timestamp: "2026-03-13T12:00:01Z",
        message: {
          content: [
            {
              type: "tool_use",
              id: "tool-1",
              name: "Read",
              input: { file_path: "/file.txt" },
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.tools).toHaveLength(1);
  });

  it("handles entries without timestamp", async () => {
    const lines = [
      JSON.stringify({
        message: {
          content: [
            {
              type: "tool_use",
              id: "tool-no-ts",
              name: "Read",
              input: { file_path: "/file.txt" },
            },
          ],
        },
      }),
    ];
    writeFileSync(tmpFile, lines.join("\n"));

    const result = await parseTranscript(tmpFile);
    expect(result.tools).toHaveLength(1);
    expect(result.tools[0].id).toBe("tool-no-ts");
  });
});
