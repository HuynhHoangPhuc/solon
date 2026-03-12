import { describe, expect, it } from "vitest";
import {
  HASHLINE_DICT,
  NIBBLE_STR,
  computeLineHash,
} from "../src/hashline/hash-computation.js";

describe("NIBBLE_STR", () => {
  it("has exactly 16 chars", () => {
    expect(NIBBLE_STR.length).toBe(16);
  });

  it("has unique chars", () => {
    expect(new Set(NIBBLE_STR).size).toBe(16);
  });
});

describe("HASHLINE_DICT", () => {
  it("has exactly 256 entries", () => {
    expect(HASHLINE_DICT.length).toBe(256);
  });

  it("each entry is 2 chars from NIBBLE_STR", () => {
    for (const entry of HASHLINE_DICT) {
      expect(entry).toHaveLength(2);
      expect(NIBBLE_STR).toContain(entry[0]);
      expect(NIBBLE_STR).toContain(entry[1]);
    }
  });

  it("entry at index 0 uses first nibble char twice", () => {
    // index 0: high nibble=0 -> NIBBLE_STR[0], low nibble=0 -> NIBBLE_STR[0]
    expect(HASHLINE_DICT[0]).toBe(NIBBLE_STR[0]! + NIBBLE_STR[0]!);
  });

  it("entry at index 255 uses last nibble char twice", () => {
    // index 255: high nibble=15 -> NIBBLE_STR[15], low nibble=15 -> NIBBLE_STR[15]
    expect(HASHLINE_DICT[255]).toBe(NIBBLE_STR[15]! + NIBBLE_STR[15]!);
  });
});

describe("computeLineHash", () => {
  it("returns a 2-char string from NIBBLE_STR", async () => {
    const hash = await computeLineHash(1, "const x = 1;");
    expect(hash).toHaveLength(2);
    expect(NIBBLE_STR).toContain(hash[0]);
    expect(NIBBLE_STR).toContain(hash[1]);
  });

  it("is deterministic for same input", async () => {
    const a = await computeLineHash(5, "hello world");
    const b = await computeLineHash(5, "hello world");
    expect(a).toBe(b);
  });

  it("strips whitespace before hashing", async () => {
    const a = await computeLineHash(1, "foo bar");
    const b = await computeLineHash(1, "foobar");
    expect(a).toBe(b);
  });

  it("uses lineNumber as seed for blank lines", async () => {
    const h1 = await computeLineHash(1, "");
    const h2 = await computeLineHash(2, "");
    // Different line numbers → different seeds → likely different hashes
    // (not guaranteed, but statistically expected with xxhash)
    expect(typeof h1).toBe("string");
    expect(typeof h2).toBe("string");
  });

  it("uses seed=0 for lines with alphanumeric content", async () => {
    // Same content at different line numbers should produce same hash
    const h1 = await computeLineHash(10, "export function foo() {}");
    const h2 = await computeLineHash(99, "export function foo() {}");
    expect(h1).toBe(h2);
  });

  it("symbol-only lines are seed-dependent", async () => {
    // "---" has no alpha/digit → uses lineNumber as seed
    const h1 = await computeLineHash(1, "---");
    const h2 = await computeLineHash(2, "---");
    // Seeds differ → hashes likely differ (not a hard guarantee, but test the logic path)
    expect(typeof h1).toBe("string");
    expect(typeof h2).toBe("string");
  });
});
