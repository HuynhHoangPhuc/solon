import { describe, it, expect } from "bun:test";
import {
  extractFromToolInput, extractFromCommand, looksLikePath,
  isCommandKeyword, isBlockedDirName, normalizeExtractedPath,
} from "../../scout-block/path-extractor.ts";

describe("extractFromToolInput", () => {
  it("extracts file_path", () => {
    const result = extractFromToolInput({ file_path: "src/main.ts" });
    expect(result).toContain("src/main.ts");
  });

  it("extracts path param", () => {
    const result = extractFromToolInput({ path: "node_modules" });
    expect(result).toContain("node_modules");
  });

  it("extracts pattern param", () => {
    const result = extractFromToolInput({ pattern: "**/*.ts" });
    expect(result).toContain("**/*.ts");
  });

  it("extracts from command", () => {
    const result = extractFromToolInput({ command: "ls node_modules" });
    expect(result.some((p) => p.includes("node_modules"))).toBe(true);
  });

  it("returns empty array for empty input", () => {
    expect(extractFromToolInput({})).toEqual([]);
  });
});

describe("extractFromCommand", () => {
  it("extracts path from cd command", () => {
    const result = extractFromCommand("cd node_modules");
    expect(result.some((p) => p.includes("node_modules"))).toBe(true);
  });

  it("extracts path from ls command", () => {
    const result = extractFromCommand("ls dist/");
    expect(result.some((p) => p.includes("dist"))).toBe(true);
  });

  it("extracts quoted path with spaces", () => {
    const result = extractFromCommand('cat "my dir/file.ts"');
    expect(result.some((p) => p.includes("my dir/file.ts"))).toBe(true);
  });

  it("does not extract non-path strings from grep", () => {
    const result = extractFromCommand("grep -r 'hello world' src/");
    // 'hello world' is not a path
    expect(result.every((p) => !p.includes("hello world"))).toBe(true);
  });

  it("extracts path with extension", () => {
    const result = extractFromCommand("cat src/config.json");
    expect(result.some((p) => p.includes("src/config.json"))).toBe(true);
  });

  it("handles compound commands — extracts paths from each segment", () => {
    const result = extractFromCommand("ls dist && echo done");
    expect(result.some((p) => p.includes("dist"))).toBe(true);
  });

  it("skips flags", () => {
    const result = extractFromCommand("rm -rf node_modules");
    // -rf is a flag, node_modules is a blocked dir name for rm (filesystem command)
    expect(result.some((p) => p.includes("node_modules"))).toBe(true);
  });

  it("does not extract bare dir names from non-fs commands like echo", () => {
    const result = extractFromCommand("echo build");
    // 'build' is a bare name extracted only for fs commands — echo is not fs
    expect(result.some((p) => p === "build")).toBe(false);
  });

  it("skips heredoc body content", () => {
    const result = extractFromCommand("cat <<EOF\nsome content with dist/file\nEOF");
    // Content inside heredoc should not be treated as paths
    expect(result.length).toBeLessThanOrEqual(1);
  });
});

describe("looksLikePath", () => {
  it("detects path with /", () => expect(looksLikePath("src/main.ts")).toBe(true));
  it("detects ./ prefix", () => expect(looksLikePath("./file.ts")).toBe(true));
  it("detects ../ prefix", () => expect(looksLikePath("../parent")).toBe(true));
  it("detects file extension", () => expect(looksLikePath("file.json")).toBe(true));
  it("rejects bare word", () => expect(looksLikePath("hello")).toBe(false));
  it("rejects single char", () => expect(looksLikePath("x")).toBe(false));
});

describe("isCommandKeyword", () => {
  it("recognizes shell commands", () => {
    expect(isCommandKeyword("echo")).toBe(true);
    expect(isCommandKeyword("cat")).toBe(true);
    expect(isCommandKeyword("npm")).toBe(true);
    expect(isCommandKeyword("git")).toBe(true);
  });

  it("rejects non-keyword", () => {
    expect(isCommandKeyword("myapp")).toBe(false);
    expect(isCommandKeyword("solon-hooks")).toBe(false);
  });

  it("is case-insensitive", () => {
    expect(isCommandKeyword("NPM")).toBe(true);
    expect(isCommandKeyword("GIT")).toBe(true);
  });
});

describe("isBlockedDirName", () => {
  it("recognizes blocked dir names", () => {
    expect(isBlockedDirName("node_modules")).toBe(true);
    expect(isBlockedDirName("dist")).toBe(true);
    expect(isBlockedDirName(".git")).toBe(true);
    expect(isBlockedDirName("__pycache__")).toBe(true);
  });

  it("rejects non-blocked names", () => {
    expect(isBlockedDirName("src")).toBe(false);
    expect(isBlockedDirName("hello")).toBe(false);
  });
});

describe("normalizeExtractedPath", () => {
  it("strips surrounding double quotes", () => {
    expect(normalizeExtractedPath('"src/main.ts"')).toBe("src/main.ts");
  });

  it("strips surrounding single quotes", () => {
    expect(normalizeExtractedPath("'src/main.ts'")).toBe("src/main.ts");
  });

  it("normalizes Windows backslashes", () => {
    expect(normalizeExtractedPath("src\\lib\\util.ts")).toBe("src/lib/util.ts");
  });

  it("removes trailing slash", () => {
    expect(normalizeExtractedPath("src/")).toBe("src");
  });

  it("strips shell metacharacters from edges", () => {
    expect(normalizeExtractedPath("`src/main.ts`")).toBe("src/main.ts");
  });
});
