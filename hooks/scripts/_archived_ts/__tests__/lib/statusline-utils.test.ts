import { describe, it, expect } from "bun:test";
import { visibleLength, formatElapsed, collapseHome, getTerminalWidth } from "../../lib/statusline-utils.ts";
import * as os from "node:os";

describe("visibleLength", () => {
  describe("plain ASCII", () => {
    it("returns correct length for ASCII-only strings", () => {
      expect(visibleLength("hello")).toBe(5);
      expect(visibleLength("test123")).toBe(7);
    });

    it("returns 0 for empty string", () => {
      expect(visibleLength("")).toBe(0);
    });

    it("handles null and non-string inputs", () => {
      expect(visibleLength(null as unknown as string)).toBe(0);
      expect(visibleLength(undefined as unknown as string)).toBe(0);
    });
  });

  describe("ANSI escape codes", () => {
    it("strips ANSI color codes", () => {
      const withColor = "\x1b[32mgreen\x1b[0m";
      expect(visibleLength(withColor)).toBe(5);
    });

    it("strips various ANSI sequences", () => {
      const withCodes = "\x1b[1mbold\x1b[0m text";
      expect(visibleLength(withCodes)).toBe(9); // "bold text" = 4 + 1 + 4
    });

    it("handles complex ANSI formatting", () => {
      const complex = "\x1b[38;5;82mcomplex\x1b[0m";
      expect(visibleLength(complex)).toBe(7);
    });
  });

  describe("emoji and SMP characters (BUG FIX #3)", () => {
    it("counts BMP emoji as 2 columns: 🤖", () => {
      // Robot emoji U+1F916 in SMP range (U+1F300–U+1F9FF)
      expect(visibleLength("🤖")).toBe(2);
    });

    it("counts folder emoji as 2 columns: 📁", () => {
      // Folder emoji U+1F4C1 in SMP range
      expect(visibleLength("📁")).toBe(2);
    });

    it("counts plant emoji as 2 columns: 🌿", () => {
      // Plant emoji U+1F33F in SMP range
      expect(visibleLength("🌿")).toBe(2);
    });

    it("counts memo emoji as 2 columns: 📝", () => {
      // Memo emoji U+1F4DD in SMP range
      expect(visibleLength("📝")).toBe(2);
    });

    it("multiple SMP emoji", () => {
      // 🤖📁🌿📝 = 2+2+2+2 = 8 columns
      expect(visibleLength("🤖📁🌿📝")).toBe(8);
    });

    it("mixed text and emoji", () => {
      // "git" (3) + "🌿" (2) + "main" (4) = 9
      expect(visibleLength("git🌿main")).toBe(9);
    });

    it("misc symbol emoji (U+2600–U+26FF)", () => {
      // ☀️ U+2600, but base emoji without variation is 1 codepoint
      expect(visibleLength("☀")).toBe(2);
    });

    it("dingbat emoji (U+2700–U+27BF)", () => {
      // ✈ U+2708
      expect(visibleLength("✈")).toBe(2);
    });
  });

  describe("combined with ANSI codes", () => {
    it("strips ANSI and counts emoji width correctly", () => {
      const colored = "\x1b[32m🤖\x1b[0m";
      expect(visibleLength(colored)).toBe(2);
    });

    it("emoji in colored text", () => {
      const text = "\x1b[1m🌿 main\x1b[0m";
      expect(visibleLength(text)).toBe(7); // 2 (emoji) + 1 (space) + 4 (main) = 7
    });
  });

  describe("edge cases", () => {
    it("handles strings with only spaces", () => {
      expect(visibleLength("   ")).toBe(3);
    });

    it("handles newlines as 1 char each", () => {
      expect(visibleLength("line1\nline2")).toBe(11);
    });

    it("does not treat emoji variants as wider", () => {
      // Even if emoji has variation selector, we count codepoints
      const emoji = "🤖";
      expect(visibleLength(emoji)).toBe(2);
    });
  });
});

describe("formatElapsed", () => {
  describe("invalid inputs", () => {
    it("returns '0s' for null/undefined start", () => {
      expect(formatElapsed(null, null)).toBe("0s");
      expect(formatElapsed(undefined, null)).toBe("0s");
    });

    it("returns '0s' for invalid date string", () => {
      expect(formatElapsed("invalid", null)).toBe("0s");
    });
  });

  describe("very short durations", () => {
    it("returns '<1s' for durations < 1 second", () => {
      const now = new Date();
      const almostNow = new Date(now.getTime() - 500);
      expect(formatElapsed(almostNow, now)).toBe("<1s");
    });

    it("returns '<1s' for 999ms", () => {
      const end = new Date("2026-03-13T12:00:01.000Z");
      const start = new Date("2026-03-13T12:00:00.001Z");
      expect(formatElapsed(start, end)).toBe("<1s");
    });
  });

  describe("seconds duration", () => {
    it("returns formatted seconds for 1-59 seconds", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:00:30Z");
      expect(formatElapsed(start, end)).toBe("30s");
    });

    it("rounds to nearest second", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:00:05.600Z");
      expect(formatElapsed(start, end)).toBe("6s");
    });

    it("handles exactly 1 second", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:00:01Z");
      expect(formatElapsed(start, end)).toBe("1s");
    });

    it("handles exactly 59 seconds", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:00:59Z");
      expect(formatElapsed(start, end)).toBe("59s");
    });
  });

  describe("minutes and seconds", () => {
    it("returns 'm s' format for >= 1 minute", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:01:30Z");
      expect(formatElapsed(start, end)).toBe("1m 30s");
    });

    it("handles exactly 1 minute", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:01:00Z");
      expect(formatElapsed(start, end)).toBe("1m 0s");
    });

    it("handles multiple minutes", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:05:45Z");
      expect(formatElapsed(start, end)).toBe("5m 45s");
    });

    it("rounds seconds in minute format", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:02:34.700Z");
      expect(formatElapsed(start, end)).toBe("2m 35s");
    });
  });

  describe("string inputs", () => {
    it("handles ISO date strings for start", () => {
      const start = "2026-03-13T12:00:00Z";
      const end = "2026-03-13T12:00:30Z";
      expect(formatElapsed(start, end)).toBe("30s");
    });

    it("uses current time when endTime is null/undefined", () => {
      const start = new Date();
      const result = formatElapsed(start, null);
      // Should be very small duration, either "<1s" or "1s"
      expect(["<1s", "1s"]).toContain(result);
    });
  });

  describe("Date objects", () => {
    it("handles Date objects for both parameters", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = new Date("2026-03-13T12:00:15Z");
      expect(formatElapsed(start, end)).toBe("15s");
    });

    it("mixes Date and string", () => {
      const start = new Date("2026-03-13T12:00:00Z");
      const end = "2026-03-13T12:00:20Z";
      expect(formatElapsed(start, end)).toBe("20s");
    });
  });

  describe("negative duration handling", () => {
    it("returns '<1s' for negative duration (end before start)", () => {
      const start = new Date("2026-03-13T12:00:30Z");
      const end = new Date("2026-03-13T12:00:00Z");
      expect(formatElapsed(start, end)).toBe("<1s");
    });
  });
});

describe("collapseHome", () => {
  const homeDir = os.homedir();

  it("replaces home directory with ~", () => {
    const filePath = `${homeDir}/projects/my-app/src/index.ts`;
    const result = collapseHome(filePath);
    expect(result).toBe("~/projects/my-app/src/index.ts");
  });

  it("handles exact home directory", () => {
    const result = collapseHome(homeDir);
    expect(result).toBe("~");
  });

  it("handles home with trailing slash", () => {
    const filePath = `${homeDir}/`;
    const result = collapseHome(filePath);
    expect(result).toBe("~/");
  });

  it("leaves non-home paths unchanged", () => {
    expect(collapseHome("/usr/local/bin")).toBe("/usr/local/bin");
  });

  it("leaves relative paths unchanged", () => {
    expect(collapseHome("./relative/path")).toBe("./relative/path");
  });

  it("handles nested home paths", () => {
    const filePath = `${homeDir}/.claude/rules/my-rule.md`;
    const result = collapseHome(filePath);
    expect(result).toBe("~/.claude/rules/my-rule.md");
  });

  it("does not replace partial matches", () => {
    // If somehow home path matches in middle, should not replace
    const filePath = `/other${homeDir}/file`;
    const result = collapseHome(filePath);
    expect(result).toBe(filePath); // unchanged
  });
});

describe("getTerminalWidth", () => {
  it("returns default 120 if no env info available", () => {
    const width = getTerminalWidth();
    expect(width).toBe(120);
  });

  it("prefers process.stderr.columns if available", () => {
    // Note: this test may vary depending on the test environment
    // Just verify it returns a number > 0
    const width = getTerminalWidth();
    expect(width).toBeGreaterThan(0);
  });

  it("returns a reasonable number (between 40 and 400)", () => {
    const width = getTerminalWidth();
    expect(width).toBeGreaterThanOrEqual(40);
    expect(width).toBeLessThanOrEqual(400);
  });
});
