import { describe, expect, it } from "vitest";
import { formatHashLine } from "../src/hashline/hash-line-formatter.js";

describe("formatHashLine", () => {
  it('produces "{lineNumber}#{hash}|{content}" format', () => {
    expect(formatHashLine(1, "VK", "const x = 1;")).toBe("1#VK|const x = 1;");
  });

  it("handles line number 0", () => {
    expect(formatHashLine(0, "ZZ", "")).toBe("0#ZZ|");
  });

  it("handles empty content", () => {
    expect(formatHashLine(42, "MB", "")).toBe("42#MB|");
  });

  it("preserves content with special chars", () => {
    const content = "foo | bar # baz";
    expect(formatHashLine(7, "NP", content)).toBe(`7#NP|${content}`);
  });

  it("handles large line numbers", () => {
    expect(formatHashLine(99999, "HH", "end")).toBe("99999#HH|end");
  });
});
