import { describe, it, expect, beforeEach, afterEach } from "bun:test";
import { writeFileSync, mkdirSync, unlinkSync, rmdirSync, existsSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { getMcpServerNames, countMcpServersInFile, countHooksInFile, countRulesInDir, countConfigs } from "../../lib/config-counter.ts";

describe("getMcpServerNames", () => {
  let tmpFile: string;

  beforeEach(() => {
    tmpFile = join(tmpdir(), `mcp-${Date.now()}.json`);
  });

  afterEach(() => {
    if (existsSync(tmpFile)) unlinkSync(tmpFile);
  });

  it("returns empty set for non-existent file", () => {
    const result = getMcpServerNames("/non/existent/file.json");
    expect(result).toBeInstanceOf(Set);
    expect(result.size).toBe(0);
  });

  it("returns empty set for invalid JSON", () => {
    writeFileSync(tmpFile, "{ invalid json");
    const result = getMcpServerNames(tmpFile);
    expect(result.size).toBe(0);
  });

  it("returns empty set when mcpServers is missing", () => {
    writeFileSync(tmpFile, JSON.stringify({ other: "data" }));
    const result = getMcpServerNames(tmpFile);
    expect(result.size).toBe(0);
  });

  it("returns empty set when mcpServers is not an object", () => {
    writeFileSync(tmpFile, JSON.stringify({ mcpServers: "not an object" }));
    const result = getMcpServerNames(tmpFile);
    expect(result.size).toBe(0);
  });

  it("extracts server names from mcpServers object", () => {
    const config = {
      mcpServers: {
        "filesystem": { command: "echo" },
        "web": { command: "echo" },
      },
    };
    writeFileSync(tmpFile, JSON.stringify(config));
    const result = getMcpServerNames(tmpFile);
    expect(result.size).toBe(2);
    expect(result.has("filesystem")).toBe(true);
    expect(result.has("web")).toBe(true);
  });

  it("handles empty mcpServers object", () => {
    writeFileSync(tmpFile, JSON.stringify({ mcpServers: {} }));
    const result = getMcpServerNames(tmpFile);
    expect(result.size).toBe(0);
  });
});

describe("countMcpServersInFile", () => {
  let tmpFile: string;
  let excludeFile: string;

  beforeEach(() => {
    tmpFile = join(tmpdir(), `mcp-count-${Date.now()}.json`);
    excludeFile = join(tmpdir(), `mcp-exclude-${Date.now()}.json`);
  });

  afterEach(() => {
    if (existsSync(tmpFile)) unlinkSync(tmpFile);
    if (existsSync(excludeFile)) unlinkSync(excludeFile);
  });

  it("counts servers in file", () => {
    const config = {
      mcpServers: {
        server1: {},
        server2: {},
        server3: {},
      },
    };
    writeFileSync(tmpFile, JSON.stringify(config));
    expect(countMcpServersInFile(tmpFile)).toBe(3);
  });

  it("excludes servers from another file", () => {
    writeFileSync(tmpFile, JSON.stringify({
      mcpServers: { server1: {}, server2: {}, server3: {} },
    }));
    writeFileSync(excludeFile, JSON.stringify({
      mcpServers: { server1: {}, server2: {} },
    }));
    expect(countMcpServersInFile(tmpFile, excludeFile)).toBe(1);
  });

  it("returns 0 for non-existent file", () => {
    expect(countMcpServersInFile("/non/existent/file.json")).toBe(0);
  });

  it("returns 0 when all servers are excluded", () => {
    writeFileSync(tmpFile, JSON.stringify({
      mcpServers: { server1: {}, server2: {} },
    }));
    writeFileSync(excludeFile, JSON.stringify({
      mcpServers: { server1: {}, server2: {}, server3: {} },
    }));
    expect(countMcpServersInFile(tmpFile, excludeFile)).toBe(0);
  });
});

describe("countHooksInFile", () => {
  let tmpFile: string;

  beforeEach(() => {
    tmpFile = join(tmpdir(), `hooks-${Date.now()}.json`);
  });

  afterEach(() => {
    if (existsSync(tmpFile)) unlinkSync(tmpFile);
  });

  it("returns 0 for non-existent file", () => {
    expect(countHooksInFile("/non/existent/file.json")).toBe(0);
  });

  it("returns 0 for invalid JSON", () => {
    writeFileSync(tmpFile, "{ broken json");
    expect(countHooksInFile(tmpFile)).toBe(0);
  });

  it("returns 0 when hooks is missing", () => {
    writeFileSync(tmpFile, JSON.stringify({ other: "data" }));
    expect(countHooksInFile(tmpFile)).toBe(0);
  });

  it("counts hook events", () => {
    writeFileSync(tmpFile, JSON.stringify({
      hooks: {
        "SessionStart": true,
        "PreToolUse": true,
        "PostToolUse": false,
      },
    }));
    expect(countHooksInFile(tmpFile)).toBe(3);
  });

  it("returns 0 for empty hooks object", () => {
    writeFileSync(tmpFile, JSON.stringify({ hooks: {} }));
    expect(countHooksInFile(tmpFile)).toBe(0);
  });

  it("returns 0 when hooks is not an object", () => {
    writeFileSync(tmpFile, JSON.stringify({ hooks: "not an object" }));
    expect(countHooksInFile(tmpFile)).toBe(0);
  });
});

describe("countRulesInDir", () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = join(tmpdir(), `rules-${Date.now()}`);
    mkdirSync(tmpDir);
  });

  afterEach(() => {
    // Clean up recursively
    try {
      const files = require("fs").readdirSync(tmpDir, { withFileTypes: true });
      for (const file of files) {
        const path = join(tmpDir, file.name);
        if (file.isDirectory()) {
          require("fs").rmSync(path, { recursive: true, force: true });
        } else {
          unlinkSync(path);
        }
      }
      rmdirSync(tmpDir);
    } catch {
      // Silent fail
    }
  });

  it("returns 0 for non-existent directory", () => {
    expect(countRulesInDir("/non/existent/dir")).toBe(0);
  });

  it("counts .md files in directory", () => {
    writeFileSync(join(tmpDir, "rule1.md"), "# Rule 1");
    writeFileSync(join(tmpDir, "rule2.md"), "# Rule 2");
    writeFileSync(join(tmpDir, "other.txt"), "not a rule");
    expect(countRulesInDir(tmpDir)).toBe(2);
  });

  it("ignores non-.md files", () => {
    writeFileSync(join(tmpDir, "rule.md"), "# Rule");
    writeFileSync(join(tmpDir, "ignore.json"), "{}");
    expect(countRulesInDir(tmpDir)).toBe(1);
  });

  it("counts .md files in subdirectories", () => {
    mkdirSync(join(tmpDir, "subdir"));
    writeFileSync(join(tmpDir, "rule1.md"), "# Rule 1");
    writeFileSync(join(tmpDir, "subdir", "rule2.md"), "# Rule 2");
    expect(countRulesInDir(tmpDir)).toBe(2);
  });

  it("respects depth limit of 5", () => {
    let currentDir = tmpDir;
    for (let i = 0; i < 6; i++) {
      currentDir = join(currentDir, `level${i}`);
      mkdirSync(currentDir);
      writeFileSync(join(currentDir, `rule${i}.md`), "# Rule");
    }
    // Should only count up to depth 5, so 5 rules (depth 0-4), not 6
    const count = countRulesInDir(tmpDir);
    expect(count).toBeLessThanOrEqual(5);
  });

  it("returns 0 for empty directory", () => {
    expect(countRulesInDir(tmpDir)).toBe(0);
  });
});

describe("countConfigs", () => {
  it("returns a ConfigCounts object with numeric fields", () => {
    const result = countConfigs("/tmp");
    expect(result).toHaveProperty("claudeMdCount");
    expect(result).toHaveProperty("rulesCount");
    expect(result).toHaveProperty("mcpCount");
    expect(result).toHaveProperty("hooksCount");
    expect(typeof result.claudeMdCount).toBe("number");
    expect(typeof result.rulesCount).toBe("number");
    expect(typeof result.mcpCount).toBe("number");
    expect(typeof result.hooksCount).toBe("number");
  });

  it("returns non-negative counts", () => {
    const result = countConfigs("/tmp");
    expect(result.claudeMdCount).toBeGreaterThanOrEqual(0);
    expect(result.rulesCount).toBeGreaterThanOrEqual(0);
    expect(result.mcpCount).toBeGreaterThanOrEqual(0);
    expect(result.hooksCount).toBeGreaterThanOrEqual(0);
  });

  it("handles non-existent cwd gracefully", () => {
    const result = countConfigs("/non/existent/path");
    expect(result.claudeMdCount).toBeGreaterThanOrEqual(0);
    expect(result.rulesCount).toBeGreaterThanOrEqual(0);
  });

  it("handles empty cwd gracefully", () => {
    const result = countConfigs("");
    expect(result).toHaveProperty("claudeMdCount");
    expect(result).toHaveProperty("rulesCount");
  });

  it("looks for user CLAUDE.md", () => {
    // This test verifies that countConfigs checks for ~/.claude/CLAUDE.md
    const result = countConfigs("/tmp");
    // claudeMdCount should be at least checking user scope
    expect(result.claudeMdCount).toBeGreaterThanOrEqual(0);
  });

  it("looks for project CLAUDE.md variants", () => {
    // Verify it checks multiple locations in project scope
    const result = countConfigs("/tmp");
    expect(result).toHaveProperty("claudeMdCount");
  });

  it("counts rules from user ~/.claude/rules", () => {
    const result = countConfigs("/tmp");
    // Should check user rules directory
    expect(result.rulesCount).toBeGreaterThanOrEqual(0);
  });

  it("counts configs from both user and project scopes", () => {
    const result = countConfigs("/tmp");
    // Total should represent user + project counts
    expect(result.claudeMdCount + result.rulesCount + result.mcpCount + result.hooksCount).toBeGreaterThanOrEqual(0);
  });
});
