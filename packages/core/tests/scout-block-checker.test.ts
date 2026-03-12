import { describe, expect, it } from "vitest";
import { checkBroadPattern } from "../src/scout-block/broad-pattern-detector.js";
import { isBlockedDirectory } from "../src/scout-block/directory-blocklist.js";
import { checkScoutBlock } from "../src/scout-block/scout-block-checker.js";

describe("checkBroadPattern", () => {
  it("blocks **/* pattern", () => {
    const result = checkBroadPattern("**/*", "Glob");
    expect(result.blocked).toBe(true);
    expect(result.suggestion).toBeDefined();
  });

  it("blocks **/*.ts pattern", () => {
    const result = checkBroadPattern("**/*.ts", "Glob");
    expect(result.blocked).toBe(true);
  });

  it("blocks bare * pattern", () => {
    const result = checkBroadPattern("*", "Glob");
    expect(result.blocked).toBe(true);
  });

  it("allows specific patterns like src/**/*.ts", () => {
    const result = checkBroadPattern("src/**/*.ts", "Glob");
    expect(result.blocked).toBe(false);
  });

  it("allows patterns like packages/core/src/**/*", () => {
    const result = checkBroadPattern("packages/core/src/**/*", "Glob");
    expect(result.blocked).toBe(false);
  });
});

describe("isBlockedDirectory", () => {
  it("blocks node_modules path", () => {
    expect(isBlockedDirectory("/project/node_modules/foo")).toBe(true);
  });

  it("blocks dist path", () => {
    expect(isBlockedDirectory("/project/dist/index.js")).toBe(true);
  });

  it("blocks .git path", () => {
    expect(isBlockedDirectory("/project/.git/config")).toBe(true);
  });

  it("allows regular src path", () => {
    expect(isBlockedDirectory("/project/src/index.ts")).toBe(false);
  });

  it("respects custom patterns", () => {
    expect(
      isBlockedDirectory("/project/custom-blocked/file.ts", ["custom-blocked"]),
    ).toBe(true);
  });
});

describe("checkScoutBlock", () => {
  it("blocks broad glob pattern in tool input", async () => {
    const result = await checkScoutBlock("Glob", { pattern: "**/*" });
    expect(result.blocked).toBe(true);
    expect(result.reason).toContain("broad");
  });

  it("blocks node_modules path in tool input", async () => {
    const result = await checkScoutBlock("Read", {
      file_path: "/project/node_modules/foo.js",
    });
    expect(result.blocked).toBe(true);
    expect(result.reason).toContain("blocked directory");
  });

  it("allows specific glob pattern", async () => {
    const result = await checkScoutBlock("Glob", { pattern: "src/**/*.ts" });
    expect(result.blocked).toBe(false);
  });

  it("allows normal file path", async () => {
    const result = await checkScoutBlock("Read", {
      file_path: "/project/src/index.ts",
    });
    expect(result.blocked).toBe(false);
  });

  it("returns blocked=false for tool with no path/pattern fields", async () => {
    const result = await checkScoutBlock("Bash", { command: "echo hello" });
    expect(result.blocked).toBe(false);
  });
});
