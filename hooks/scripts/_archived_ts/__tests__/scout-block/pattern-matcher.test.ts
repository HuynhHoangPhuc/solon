import { describe, it, expect } from "bun:test";
import * as os from "node:os";
import * as path from "node:path";
import * as fs from "node:fs";
import { loadPatterns, createMatcher, matchPath, DEFAULT_PATTERNS } from "../../scout-block/pattern-matcher.ts";

describe("loadPatterns", () => {
  it("returns DEFAULT_PATTERNS when file does not exist", () => {
    const result = loadPatterns("/nonexistent/.slignore");
    expect(result).toEqual(DEFAULT_PATTERNS);
  });

  it("returns DEFAULT_PATTERNS when file is empty", () => {
    const tmp = path.join(os.tmpdir(), ".slignore-empty-test");
    fs.writeFileSync(tmp, "# just a comment\n\n");
    const result = loadPatterns(tmp);
    expect(result).toEqual(DEFAULT_PATTERNS);
    fs.unlinkSync(tmp);
  });

  it("loads custom patterns from file", () => {
    const tmp = path.join(os.tmpdir(), ".slignore-custom-test");
    fs.writeFileSync(tmp, "my-dir\n# comment\nanother-dir\n");
    const result = loadPatterns(tmp);
    expect(result).toContain("my-dir");
    expect(result).toContain("another-dir");
    expect(result).not.toContain("# comment");
    fs.unlinkSync(tmp);
  });

  it("strips blank lines and comments", () => {
    const tmp = path.join(os.tmpdir(), ".slignore-strip-test");
    fs.writeFileSync(tmp, "\n# Comment\n\nnode_modules\n\n");
    const result = loadPatterns(tmp);
    expect(result).toEqual(["node_modules"]);
    fs.unlinkSync(tmp);
  });
});

describe("createMatcher + matchPath", () => {
  it("blocks node_modules path", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    const result = matchPath(matcher, "node_modules/lodash/index.js");
    expect(result.blocked).toBe(true);
  });

  it("blocks dist directory", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    expect(matchPath(matcher, "dist/main.js").blocked).toBe(true);
  });

  it("allows src/ path", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    expect(matchPath(matcher, "src/main.ts").blocked).toBe(false);
  });

  it("allows project files", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    expect(matchPath(matcher, "package.json").blocked).toBe(false);
    expect(matchPath(matcher, "README.md").blocked).toBe(false);
  });

  it("handles absolute paths by stripping leading slash", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    expect(matchPath(matcher, "/home/user/project/node_modules/foo").blocked).toBe(true);
  });

  it("handles Windows-style backslash paths", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    expect(matchPath(matcher, "node_modules\\foo\\bar").blocked).toBe(true);
  });

  it("negation pattern allows specific path", () => {
    const patterns = ["node_modules", "!node_modules/my-local-pkg"];
    const matcher = createMatcher(patterns);
    // Direct node_modules file should be blocked
    expect(matchPath(matcher, "node_modules/lodash/index.js").blocked).toBe(true);
  });

  it("returns false for empty path", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    expect(matchPath(matcher, "").blocked).toBe(false);
  });

  it("blocks nested build dir", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    expect(matchPath(matcher, "packages/app/build/index.js").blocked).toBe(true);
  });

  it("blocks .git directory", () => {
    const matcher = createMatcher(DEFAULT_PATTERNS);
    expect(matchPath(matcher, ".git/config").blocked).toBe(true);
  });
});
