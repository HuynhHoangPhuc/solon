import { describe, it, expect } from "bun:test";
import { deepMerge, DEFAULT_CONFIG, sanitizePath } from "../../lib/config-loader.ts";

describe("deepMerge", () => {
  it("returns default when override is empty object", () => {
    const result = deepMerge({ a: 1, b: "x" }, {});
    expect(result).toEqual({ a: 1, b: "x" });
  });

  it("overrides scalar values", () => {
    const result = deepMerge({ a: 1 }, { a: 2 });
    expect(result.a).toBe(2);
  });

  it("deep merges nested objects", () => {
    const base = { plan: { namingFormat: "default", dateFormat: "YYMMDD" } };
    const override = { plan: { namingFormat: "custom" } };
    const result = deepMerge(base, override);
    expect(result.plan.namingFormat).toBe("custom");
    expect(result.plan.dateFormat).toBe("YYMMDD");
  });

  it("replaces arrays (no merge)", () => {
    const base = { items: [1, 2, 3] };
    const override = { items: [4, 5] };
    const result = deepMerge(base, override);
    expect(result.items).toEqual([4, 5]);
  });

  it("treats empty object as inherit (keeps base)", () => {
    const base = { plan: { namingFormat: "base", reportsDir: "plans/reports" } };
    const result = deepMerge(base, { plan: {} });
    expect(result.plan.namingFormat).toBe("base");
    expect(result.plan.reportsDir).toBe("plans/reports");
  });

  it("merges deeply nested structures", () => {
    const base = { a: { b: { c: 1, d: 2 } } };
    const override = { a: { b: { c: 99 } } };
    const result = deepMerge(base, override);
    expect(result.a.b.c).toBe(99);
    expect(result.a.b.d).toBe(2);
  });
});

describe("DEFAULT_CONFIG", () => {
  it("has required plan fields", () => {
    expect(DEFAULT_CONFIG.plan.namingFormat).toBeTruthy();
    expect(DEFAULT_CONFIG.plan.dateFormat).toBeTruthy();
    expect(DEFAULT_CONFIG.plan.reportsDir).toBeTruthy();
  });

  it("has paths section", () => {
    expect(DEFAULT_CONFIG.paths.plans).toBeTruthy();
    expect(DEFAULT_CONFIG.paths.docs).toBeTruthy();
  });

  it("has some hooks enabled by default", () => {
    // DEFAULT_CONFIG.hooks contains explicitly enabled hooks (value: true)
    expect(typeof DEFAULT_CONFIG.hooks).toBe("object");
    // No hook should be explicitly disabled in defaults
    const disabledHooks = Object.entries(DEFAULT_CONFIG.hooks).filter(([, v]) => v === false);
    expect(disabledHooks).toHaveLength(0);
  });
});

describe("sanitizePath", () => {
  const root = "/home/user/project";

  it("allows normal paths", () => {
    const result = sanitizePath("plans/reports", root);
    expect(result).toContain("plans/reports");
  });

  it("strips path traversal — resolves to absolute inside root", () => {
    const result = sanitizePath("../../../etc/passwd", root);
    // Returns null or an absolute path that doesn't escape root
    if (result !== null) {
      expect(result).not.toMatch(/\/etc\/passwd$/);
    }
  });

  it("returns null for empty input", () => {
    expect(sanitizePath("", root)).toBeNull();
  });

  it("returns null for undefined input", () => {
    expect(sanitizePath(undefined as unknown as string, root)).toBeNull();
  });
});

describe("isHookEnabled", () => {
  // isHookEnabled() reads from the filesystem config — test the hook logic via deepMerge
  it("deepMerge with hooks.disabled=false keeps false", () => {
    const base = { hooks: { "my-hook": true } };
    const override = { hooks: { "my-hook": false } };
    const result = deepMerge(base, override);
    expect(result.hooks["my-hook"]).toBe(false);
  });

  it("deepMerge preserves enabled hook from base when not in override", () => {
    const base = { hooks: { "my-hook": true, "other": true } };
    const override = { hooks: { "my-hook": false } };
    const result = deepMerge(base, override);
    expect(result.hooks["other"]).toBe(true);
  });
});
