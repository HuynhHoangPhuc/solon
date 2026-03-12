import { readFile } from "node:fs/promises";
import { HASHLINE_REF_PATTERN, parseLineRef } from "./hash-line-parser.js";
import { validateLineRefs } from "./hash-line-validator.js";

/** Matches a hashline range: "11#VK...13#MB" */
const RANGE_PATTERN =
  /([0-9]+#[ZPMQVRWSNKTXJBYH]{2})\.\.\.([0-9]+#[ZPMQVRWSNKTXJBYH]{2})/g;

/**
 * If oldString contains hashline refs, resolve them to actual file content.
 * Supports single refs (11#VK) and ranges (11#VK...13#MB).
 * Returns oldString unchanged if no refs found.
 */
export async function resolveHashlineEdit(
  oldString: string,
  filePath: string,
): Promise<string> {
  // Collect all refs present in oldString
  const singleRefs: string[] = [];
  const rangeRefs: Array<{ start: string; end: string }> = [];

  // Detect ranges first
  const rangeCopy = oldString;
  let rangeMatch: RegExpExecArray | null;
  RANGE_PATTERN.lastIndex = 0;
  while ((rangeMatch = RANGE_PATTERN.exec(rangeCopy)) !== null) {
    rangeRefs.push({ start: rangeMatch[1]!, end: rangeMatch[2]! });
  }

  // Detect standalone single refs (not part of a range)
  const singlePattern = new RegExp(HASHLINE_REF_PATTERN.source, "g");
  // Remove range occurrences before scanning for singles
  const withoutRanges = oldString.replace(RANGE_PATTERN, "");
  let singleMatch: RegExpExecArray | null;
  while ((singleMatch = singlePattern.exec(withoutRanges)) !== null) {
    singleRefs.push(singleMatch[0]!);
  }

  const hasRefs = singleRefs.length > 0 || rangeRefs.length > 0;
  if (!hasRefs) return oldString;

  // Read file from disk
  const fileContent = await readFile(filePath, "utf-8");
  const fileLines = fileContent.split("\n");

  // Collect all refs to validate
  const allRefs: string[] = [
    ...singleRefs,
    ...rangeRefs.map((r) => r.start),
    ...rangeRefs.map((r) => r.end),
  ];
  await validateLineRefs(allRefs, fileLines);

  // Resolve ranges
  if (rangeRefs.length > 0) {
    const { start, end } = rangeRefs[0]!;
    const startRef = parseLineRef(start);
    const endRef = parseLineRef(end);
    const startLine = Math.min(startRef.line, endRef.line);
    const endLine = Math.max(startRef.line, endRef.line);
    return fileLines.slice(startLine - 1, endLine).join("\n");
  }

  // Resolve single ref
  if (singleRefs.length > 0) {
    const ref = parseLineRef(singleRefs[0]!);
    return fileLines[ref.line - 1]!;
  }

  return oldString;
}
