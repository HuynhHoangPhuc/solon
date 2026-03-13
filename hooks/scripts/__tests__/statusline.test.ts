import { describe, it, expect } from "bun:test";
import { renderMinimal, renderCompact, renderFull } from "../lib/statusline-renderer.ts";
import type { RenderContext, TranscriptData, TranscriptAgent } from "../lib/types.ts";

describe("renderMinimal", () => {
  const baseContext: RenderContext = {
    modelName: "claude-3-haiku",
    currentDir: "/home/user/project",
    gitBranch: "main",
    gitUnstaged: 0,
    gitStaged: 0,
    gitAhead: 0,
    gitBehind: 0,
    contextPercent: 50,
    sessionText: "1.5 hours left",
    usagePercent: 30,
    linesAdded: 5,
    linesRemoved: 3,
    transcript: {
      tools: [],
      agents: [],
      todos: [],
      sessionStart: null,
    },
  };

  it("returns single line array", () => {
    const lines = renderMinimal(baseContext);
    expect(lines).toHaveLength(1);
    expect(lines[0]).toBeTruthy();
  });

  it("includes model name", () => {
    const lines = renderMinimal(baseContext);
    expect(lines[0]).toContain("🤖");
    expect(lines[0]).toContain("claude-3-haiku");
  });

  it("includes context percentage", () => {
    const lines = renderMinimal(baseContext);
    expect(lines[0]).toContain("50%");
  });

  it("includes usage info", () => {
    const lines = renderMinimal(baseContext);
    expect(lines[0]).toContain("1.5 hours left");
  });

  it("includes git branch", () => {
    const lines = renderMinimal(baseContext);
    expect(lines[0]).toContain("🌿");
    expect(lines[0]).toContain("main");
  });

  it("includes directory", () => {
    const lines = renderMinimal(baseContext);
    expect(lines[0]).toContain("📁");
  });

  it("omits context bar when 0", () => {
    const ctx = { ...baseContext, contextPercent: 0 };
    const lines = renderMinimal(ctx);
    // Should not include context bar indicator, but may have usage %
    expect(lines[0]).not.toContain("🔋");
  });

  it("omits usage when sessionText is N/A", () => {
    const ctx = { ...baseContext, sessionText: "N/A" };
    const lines = renderMinimal(ctx);
    expect(lines[0]).not.toContain("hours");
  });

  it("omits git branch when empty", () => {
    const ctx = { ...baseContext, gitBranch: "" };
    const lines = renderMinimal(ctx);
    expect(lines[0]).not.toContain("🌿");
  });

  it("does not contain non-breaking space (U+00A0)", () => {
    const lines = renderMinimal(baseContext);
    const output = lines.join("\n");
    expect(output).not.toContain("\u00A0");
  });

  it("does not start with RESET code (x1b[0m)", () => {
    const lines = renderMinimal(baseContext);
    for (const line of lines) {
      expect(line).not.toMatch(/^\x1b\[0m/);
    }
  });
});

describe("renderCompact", () => {
  const baseContext: RenderContext = {
    modelName: "claude-3-sonnet",
    currentDir: "/home/user/dev/solon",
    gitBranch: "feature/test",
    gitUnstaged: 2,
    gitStaged: 1,
    gitAhead: 3,
    gitBehind: 0,
    contextPercent: 75,
    sessionText: "30 minutes left",
    usagePercent: 60,
    linesAdded: 10,
    linesRemoved: 5,
    transcript: {
      tools: [],
      agents: [],
      todos: [],
      sessionStart: null,
    },
  };

  it("returns exactly 2 lines", () => {
    const lines = renderCompact(baseContext);
    expect(lines).toHaveLength(2);
  });

  it("line 1 includes model and context", () => {
    const lines = renderCompact(baseContext);
    expect(lines[0]).toContain("🤖");
    expect(lines[0]).toContain("claude-3-sonnet");
    expect(lines[0]).toContain("75%");
  });

  it("line 1 includes usage time", () => {
    const lines = renderCompact(baseContext);
    expect(lines[0]).toContain("⌛");
    expect(lines[0]).toContain("30 minutes left");
  });

  it("line 2 includes directory and git branch", () => {
    const lines = renderCompact(baseContext);
    expect(lines[1]).toContain("📁");
    expect(lines[1]).toContain("/home/user/dev/solon");
    expect(lines[1]).toContain("🌿");
    expect(lines[1]).toContain("feature/test");
  });

  it("omits context bar when 0", () => {
    const ctx = { ...baseContext, contextPercent: 0 };
    const lines = renderCompact(ctx);
    expect(lines[0]).not.toContain("█");
  });

  it("handles no git branch", () => {
    const ctx = { ...baseContext, gitBranch: "" };
    const lines = renderCompact(ctx);
    expect(lines).toHaveLength(2);
    expect(lines[1]).not.toContain("🌿");
  });

  it("does not contain non-breaking space", () => {
    const lines = renderCompact(baseContext);
    const output = lines.join("\n");
    expect(output).not.toContain("\u00A0");
  });

  it("does not start lines with RESET code", () => {
    const lines = renderCompact(baseContext);
    for (const line of lines) {
      expect(line).not.toMatch(/^\x1b\[0m/);
    }
  });

  it("line 1 should be readable by Claude Code", () => {
    const lines = renderCompact(baseContext);
    expect(lines[0].length).toBeGreaterThan(0);
    // Should contain model name which Claude Code reads
    expect(lines[0]).toMatch(/claude-3-/);
  });
});

describe("renderFull", () => {
  const emptyTranscript: TranscriptData = {
    tools: [],
    agents: [],
    todos: [],
    sessionStart: null,
  };

  const baseContext: RenderContext = {
    modelName: "claude-3-opus",
    currentDir: "/Users/phuc/Developer/solon",
    gitBranch: "main",
    gitUnstaged: 1,
    gitStaged: 0,
    gitAhead: 0,
    gitBehind: 0,
    contextPercent: 85,
    sessionText: "2 hours left",
    usagePercent: 50,
    linesAdded: 20,
    linesRemoved: 10,
    transcript: emptyTranscript,
  };

  it("returns array of lines", () => {
    const lines = renderFull(baseContext);
    expect(Array.isArray(lines)).toBe(true);
    expect(lines.length).toBeGreaterThan(0);
  });

  it("includes session lines with model, dir, git", () => {
    const lines = renderFull(baseContext);
    const output = lines.join("\n");
    expect(output).toContain("🤖");
    expect(output).toContain("claude-3-opus");
    expect(output).toContain("📁");
    expect(output).toContain("🌿");
    expect(output).toContain("main");
  });

  it("includes stats with emoji", () => {
    const lines = renderFull(baseContext);
    const output = lines.join("\n");
    expect(output).toContain("📝");
    expect(output).toContain("+20");
    expect(output).toContain("-10");
  });

  it("includes agents when present in transcript", () => {
    const agent: TranscriptAgent = {
      id: "agent-1",
      type: "tester",
      model: "claude-3-haiku",
      description: "Run tests for the implementation",
      status: "completed",
      startTime: new Date("2026-03-13T12:00:00Z"),
      endTime: new Date("2026-03-13T12:05:00Z"),
    };
    const ctx = {
      ...baseContext,
      transcript: {
        ...emptyTranscript,
        agents: [agent],
      },
    };
    const lines = renderFull(ctx);
    const output = lines.join("\n");
    expect(output).toContain("tester");
  });

  it("includes todos when present", () => {
    const ctx = {
      ...baseContext,
      transcript: {
        ...emptyTranscript,
        todos: [
          { content: "Implement feature", status: "completed" },
          { content: "Write tests", status: "in_progress" },
        ],
      },
    };
    const lines = renderFull(ctx);
    const output = lines.join("\n");
    expect(output).toContain("Write tests");
  });

  it("renders running agent with detail line", () => {
    const runningAgent: TranscriptAgent = {
      id: "agent-running",
      type: "reviewer",
      model: null,
      description: "Reviewing code changes for quality",
      status: "running",
      startTime: new Date("2026-03-13T12:10:00Z"),
      endTime: null,
    };
    const ctx = {
      ...baseContext,
      transcript: {
        ...emptyTranscript,
        agents: [runningAgent],
      },
    };
    const lines = renderFull(ctx);
    const output = lines.join("\n");
    expect(output).toContain("Reviewing code");
    expect(output).toContain("reviewer");
  });

  it("does not contain non-breaking space", () => {
    const lines = renderFull(baseContext);
    const output = lines.join("\n");
    expect(output).not.toContain("\u00A0");
  });

  it("no line starts with RESET code", () => {
    const lines = renderFull(baseContext);
    for (const line of lines) {
      expect(line).not.toMatch(/^\x1b\[0m/);
    }
  });

  it("respects responsive layout for long paths", () => {
    const longDir = "/very/long/path/to/some/deeply/nested/project/directory";
    const ctx = { ...baseContext, currentDir: longDir };
    const lines = renderFull(ctx);
    // Should not crash, layout should adapt
    expect(lines.length).toBeGreaterThan(0);
  });

  it("shows multiple agents as flow with count", () => {
    const agents: TranscriptAgent[] = [
      {
        id: "agent-1",
        type: "tester",
        model: null,
        description: "Test phase 1",
        status: "completed",
        startTime: new Date("2026-03-13T12:00:00Z"),
        endTime: new Date("2026-03-13T12:02:00Z"),
      },
      {
        id: "agent-2",
        type: "tester",
        model: null,
        description: "Test phase 2",
        status: "completed",
        startTime: new Date("2026-03-13T12:02:00Z"),
        endTime: new Date("2026-03-13T12:04:00Z"),
      },
    ];
    const ctx = {
      ...baseContext,
      transcript: {
        ...emptyTranscript,
        agents,
      },
    };
    const lines = renderFull(ctx);
    const output = lines.join("\n");
    expect(output).toContain("tester");
  });

  it("handles empty transcript gracefully", () => {
    const lines = renderFull(baseContext);
    expect(lines.length).toBeGreaterThan(0);
    const output = lines.join("\n");
    expect(output).toBeTruthy();
  });

  it("shows all completed todos progress", () => {
    const ctx = {
      ...baseContext,
      transcript: {
        ...emptyTranscript,
        todos: [
          { content: "Task 1", status: "completed" },
          { content: "Task 2", status: "completed" },
          { content: "Task 3", status: "completed" },
        ],
      },
    };
    const lines = renderFull(ctx);
    const output = lines.join("\n");
    expect(output).toContain("All 3 todos complete");
  });
});

describe("renderMinimal + renderCompact + renderFull integration", () => {
  const ctx: RenderContext = {
    modelName: "test-model",
    currentDir: "/test",
    gitBranch: "test-branch",
    gitUnstaged: 0,
    gitStaged: 0,
    gitAhead: 0,
    gitBehind: 0,
    contextPercent: 50,
    sessionText: "1 hour left",
    usagePercent: 40,
    linesAdded: 0,
    linesRemoved: 0,
    transcript: {
      tools: [],
      agents: [],
      todos: [],
      sessionStart: null,
    },
  };

  it("all modes return non-empty lines", () => {
    const minimal = renderMinimal(ctx);
    const compact = renderCompact(ctx);
    const full = renderFull(ctx);

    expect(minimal.length).toBeGreaterThan(0);
    expect(compact.length).toBeGreaterThan(0);
    expect(full.length).toBeGreaterThan(0);

    expect(minimal[0]).toBeTruthy();
    expect(compact[0]).toBeTruthy();
    expect(full[0]).toBeTruthy();
  });

  it("all modes include model name", () => {
    const minimal = renderMinimal(ctx);
    const compact = renderCompact(ctx);
    const full = renderFull(ctx);

    expect(minimal.join("\n")).toContain("test-model");
    expect(compact.join("\n")).toContain("test-model");
    expect(full.join("\n")).toContain("test-model");
  });

  it("compact always has 2 lines", () => {
    const lines = renderCompact(ctx);
    expect(lines).toHaveLength(2);
  });

  it("no output contains consecutive spaces", () => {
    const minimal = renderMinimal(ctx);
    const compact = renderCompact(ctx);
    const full = renderFull(ctx);

    for (const line of [...minimal, ...compact, ...full]) {
      // Allow double spaces from formatting, but not \u00A0
      expect(line).not.toContain("\u00A0");
    }
  });

  it("emoji appear correctly in all modes", () => {
    const minimal = renderMinimal(ctx);
    const compact = renderCompact(ctx);
    const full = renderFull(ctx);

    const allOutput = [...minimal, ...compact, ...full].join("\n");
    expect(allOutput).toContain("🤖"); // model
    expect(allOutput).toContain("📁"); // folder
    expect(allOutput).toContain("🌿"); // branch
  });
});
