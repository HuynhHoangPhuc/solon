import { describe, expect, it } from "vitest";
import { computeLineHash } from "../src/hashline/hash-computation.js";
import {
  HashlineMismatchError,
  validateLineRefs,
} from "../src/hashline/hash-line-validator.js";

const lines = [
  'import { foo } from "./foo.js";', // line 1
  "", // line 2
  "export function bar() {", // line 3
  "  return foo();", // line 4
  "}", // line 5
];

async function refFor(lineNum: number): Promise<string> {
  const hash = await computeLineHash(lineNum, lines[lineNum - 1]!);
  return `${lineNum}#${hash}`;
}

describe("validateLineRefs", () => {
  it("passes for valid refs", async () => {
    const refs = [await refFor(1), await refFor(3)];
    await expect(validateLineRefs(refs, lines)).resolves.toBeUndefined();
  });

  it("throws HashlineMismatchError for stale ref", async () => {
    const hash = await computeLineHash(1, lines[0]!);
    // Corrupt the hash by flipping chars
    const badHash = hash === "ZZ" ? "HH" : "ZZ";
    const staleRef = `1#${badHash}`;

    await expect(validateLineRefs([staleRef], lines)).rejects.toThrow(
      HashlineMismatchError,
    );
  });

  it("error message contains line count and >>> markers", async () => {
    const hash = await computeLineHash(3, lines[2]!);
    const badHash = hash === "ZZ" ? "HH" : "ZZ";

    try {
      await validateLineRefs([`3#${badHash}`], lines);
      expect.fail("should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(HashlineMismatchError);
      const err = e as HashlineMismatchError;
      expect(err.message).toContain("1 line(s) have changed");
      expect(err.message).toContain(">>>");
    }
  });

  it("includes ±2 context lines around mismatch", async () => {
    const hash = await computeLineHash(3, lines[2]!);
    const badHash = hash === "ZZ" ? "HH" : "ZZ";

    try {
      await validateLineRefs([`3#${badHash}`], lines);
      expect.fail("should have thrown");
    } catch (e) {
      const err = e as HashlineMismatchError;
      // Line 3 mismatch → context should include lines 1,2,3,4,5
      expect(err.message).toContain("1#");
      expect(err.message).toContain("5#");
    }
  });

  it("populates remaps with correct new refs", async () => {
    const hash = await computeLineHash(1, lines[0]!);
    const badHash = hash === "ZZ" ? "HH" : "ZZ";
    const staleRef = `1#${badHash}`;

    try {
      await validateLineRefs([staleRef], lines);
      expect.fail("should have thrown");
    } catch (e) {
      const err = e as HashlineMismatchError;
      expect(err.remaps.has(staleRef)).toBe(true);
      const newRef = err.remaps.get(staleRef)!;
      expect(newRef).toMatch(/^1#[ZPMQVRWSNKTXJBYH]{2}$/);
    }
  });

  it("throws for out-of-bounds line number", async () => {
    await expect(validateLineRefs(["999#ZZ"], lines)).rejects.toThrow(
      "out of bounds",
    );
  });
});
