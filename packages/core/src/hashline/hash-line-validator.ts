import { computeLineHash } from "./hash-computation.js";
import { formatHashLine } from "./hash-line-formatter.js";
import { parseLineRef } from "./hash-line-parser.js";

/** Thrown when one or more hashline refs no longer match the file on disk */
export class HashlineMismatchError extends Error {
  /** Maps old "line#hash" refs to new "line#newHash" refs for all mismatched lines */
  constructor(
    message: string,
    public remaps: Map<string, string>,
  ) {
    super(message);
    this.name = "HashlineMismatchError";
  }
}

/**
 * Validate that all hashline refs still match the given file lines.
 * Throws HashlineMismatchError if any ref is stale, with ±2 context lines.
 */
export async function validateLineRefs(
  refs: string[],
  fileLines: string[],
): Promise<void> {
  interface Mismatch {
    lineNumber: number;
    oldRef: string;
    newHash: string;
  }

  const mismatches: Mismatch[] = [];
  const remaps = new Map<string, string>();

  for (const ref of refs) {
    const { line, hash } = parseLineRef(ref);

    if (line < 1 || line > fileLines.length) {
      throw new Error(
        `Line ${line} is out of bounds (file has ${fileLines.length} lines)`,
      );
    }

    const lineContent = fileLines[line - 1]!;
    const currentHash = await computeLineHash(line, lineContent);

    if (currentHash !== hash) {
      mismatches.push({ lineNumber: line, oldRef: ref, newHash: currentHash });
      remaps.set(ref, `${line}#${currentHash}`);
    }
  }

  if (mismatches.length === 0) return;

  // Build context block: ±2 lines around each mismatch, >>> prefix on changed lines
  const mismatchLines = new Set(mismatches.map((m) => m.lineNumber));
  const contextLineNums = new Set<number>();

  for (const lineNum of mismatchLines) {
    for (
      let i = Math.max(1, lineNum - 2);
      i <= Math.min(fileLines.length, lineNum + 2);
      i++
    ) {
      contextLineNums.add(i);
    }
  }

  const sortedNums = Array.from(contextLineNums).sort((a, b) => a - b);
  const contextLines: string[] = [];
  let prevNum: number | null = null;

  for (const num of sortedNums) {
    // Insert separator for gaps
    if (prevNum !== null && num > prevNum + 1) {
      contextLines.push("...");
    }
    const content = fileLines[num - 1]!;
    const hash = await computeLineHash(num, content);
    const formatted = formatHashLine(num, hash, content);
    if (mismatchLines.has(num)) {
      contextLines.push(`>>> ${formatted}`);
    } else {
      contextLines.push(`    ${formatted}`);
    }
    prevNum = num;
  }

  const n = mismatches.length;
  const message = `${n} line(s) have changed since last read. Use updated {line_number}#{hash_id} references below (>>> marks changed lines).\n\n${contextLines.join("\n")}`;

  throw new HashlineMismatchError(message, remaps);
}
