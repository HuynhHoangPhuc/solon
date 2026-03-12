import { describe, expect, it } from "vitest";
import { truncateOutput } from "../src/truncation/output-truncator.js";
import { estimateTokens } from "../src/truncation/token-estimator.js";

describe("truncateOutput", () => {
  it("returns text unchanged when within token budget", () => {
    const text = "short text";
    expect(truncateOutput(text, 1000)).toBe(text);
  });

  it("truncates text exceeding budget", () => {
    // Build a text that clearly exceeds a small budget
    const lines = Array.from(
      { length: 100 },
      (_, i) => `line ${i}: ${"x".repeat(40)}`,
    );
    const text = lines.join("\n");
    const maxTokens = 50;

    const result = truncateOutput(text, maxTokens);
    expect(estimateTokens(result)).toBeLessThanOrEqual(maxTokens + 20); // small margin for notice
    expect(result).toContain("truncated due to context window limit");
  });

  it("preserves header lines at top", () => {
    const header = ["# Title", "## Subtitle", "description"];
    const body = Array.from(
      { length: 200 },
      (_, i) => `body line ${i}: ${"y".repeat(30)}`,
    );
    const text = [...header, ...body].join("\n");

    const result = truncateOutput(text, 100, 3);
    const resultLines = result.split("\n");
    expect(resultLines[0]).toBe("# Title");
    expect(resultLines[1]).toBe("## Subtitle");
    expect(resultLines[2]).toBe("description");
  });

  it("appends truncation notice with count", () => {
    const lines = Array.from(
      { length: 500 },
      (_, i) => `line ${i}: ${"z".repeat(20)}`,
    );
    const text = lines.join("\n");
    const result = truncateOutput(text, 100);
    expect(result).toMatch(
      /\[\d+ more lines truncated due to context window limit\]/,
    );
  });

  it("handles text with fewer lines than preserveHeaderLines", () => {
    // Force truncation on a very short text with a tiny budget
    const text = "a".repeat(1000);
    const result = truncateOutput(text, 10);
    expect(result).toContain("[Output truncated due to context window limit]");
  });

  it("uses default 3 header lines when not specified", () => {
    const lines = Array.from(
      { length: 200 },
      (_, i) => `x line ${i}: ${"a".repeat(30)}`,
    );
    const text = lines.join("\n");
    const result = truncateOutput(text, 80);
    // First line always preserved
    expect(result.startsWith(lines[0]!)).toBe(true);
  });
});
