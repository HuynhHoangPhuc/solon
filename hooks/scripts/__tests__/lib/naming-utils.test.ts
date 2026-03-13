import { describe, it, expect } from "bun:test";
import { formatDate, resolveNamingPattern, validateNamingPattern, sanitizeSlug } from "../../lib/naming-utils.ts";

describe("sanitizeSlug", () => {
  it("replaces spaces with hyphens", () => {
    // sanitizeSlug does NOT lowercase — preserves case
    expect(sanitizeSlug("Hello World")).toBe("Hello-World");
  });

  it("replaces special chars with hyphens", () => {
    // @, ! and : become hyphens, then collapse to single hyphen
    expect(sanitizeSlug("feat: add user@auth!")).toBe("feat-add-user-auth");
  });

  it("trims leading/trailing hyphens", () => {
    expect(sanitizeSlug("--hello--")).toBe("hello");
  });

  it("collapses multiple hyphens", () => {
    expect(sanitizeSlug("foo---bar")).toBe("foo-bar");
  });

  it("returns empty string for empty input", () => {
    expect(sanitizeSlug("")).toBe("");
    expect(sanitizeSlug("   ")).toBe("");
  });

  it("truncates at 100 characters", () => {
    const long = "a".repeat(120);
    expect(sanitizeSlug(long).length).toBeLessThanOrEqual(100);
  });
});

describe("formatDate", () => {
  it("formats YYMMDD-HHmm (default)", () => {
    const result = formatDate("YYMMDD-HHmm");
    expect(result).toMatch(/^\d{6}-\d{4}$/);
  });

  it("formats YYYYMMDD", () => {
    const result = formatDate("YYYYMMDD");
    expect(result).toMatch(/^\d{8}$/);
  });

  it("formats YYYY-MM-DD", () => {
    const result = formatDate("YYYY-MM-DD");
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("returns format string unchanged for unknown tokens", () => {
    // No fallback — unknown format is returned as-is
    const result = formatDate("UNKNOWN");
    expect(result).toBe("UNKNOWN");
  });
});

describe("resolveNamingPattern", () => {
  const basePlan = {
    namingFormat: "{date}-{issue}-{slug}",
    dateFormat: "YYMMDD-HHmm",
    reportsDir: "plans/reports",
    issuePrefix: "SL-",
  };

  it("replaces {date} placeholder", () => {
    const result = resolveNamingPattern(basePlan, null);
    expect(result).toMatch(/\d{6}-\d{4}/);
  });

  it("extracts issue number from branch name when prefix matches", () => {
    const result = resolveNamingPattern(basePlan, "feature/SL-123-some-feature");
    expect(result).toContain("123");
  });

  it("removes {issue} placeholder when no issue number found", () => {
    const result = resolveNamingPattern(basePlan, "main");
    expect(result).not.toContain("{issue}");
  });

  it("leaves {slug} placeholder for later resolution", () => {
    // {slug} is NOT auto-resolved from branch — it stays in output
    const result = resolveNamingPattern(basePlan, "feature/add-user-auth");
    expect(result).toContain("{slug}");
  });

  it("handles null branch", () => {
    const result = resolveNamingPattern(basePlan, null);
    expect(typeof result).toBe("string");
    expect(result.length).toBeGreaterThan(0);
  });
});

describe("validateNamingPattern", () => {
  it("accepts patterns that still contain {slug} placeholder", () => {
    // Pattern is valid if it has {slug} (will be replaced when slug is known)
    const result = validateNamingPattern("260313-1230-{slug}");
    expect(result.valid).toBe(true);
  });

  it("rejects patterns without {slug} placeholder", () => {
    // If {slug} is missing, the pattern is considered incomplete
    const result = validateNamingPattern("260313-1230-add-auth");
    expect(result.valid).toBe(false);
  });

  it("rejects patterns with OTHER unresolved placeholders (not {slug})", () => {
    const result = validateNamingPattern("{date}-{slug}");
    expect(result.valid).toBe(false);
    expect(result.error).toContain("{date}");
  });

  it("rejects empty pattern", () => {
    const result = validateNamingPattern("");
    expect(result.valid).toBe(false);
  });
});
