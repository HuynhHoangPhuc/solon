/** Matches a valid hashline ref like "11#VK" */
export const HASHLINE_REF_PATTERN = /^([0-9]+)#([ZPMQVRWSNKTXJBYH]{2})$/;

export interface LineRef {
  line: number;
  hash: string;
}

/**
 * Normalize LLM-noisy line refs:
 * - Strip leading >>>, +, - prefixes (diff noise)
 * - Strip spaces around #
 * - Strip trailing |content portion
 */
export function normalizeLineRef(ref: string): string {
  let s = ref.trim();
  // Strip leading diff/context markers
  s = s.replace(/^[>+\-]+\s*/, "");
  // Strip trailing |content
  const pipeIdx = s.indexOf("|");
  if (pipeIdx !== -1) s = s.slice(0, pipeIdx);
  // Strip spaces around #
  s = s.replace(/\s*#\s*/, "#");
  return s.trim();
}

/**
 * Parse a hashline reference string into { line, hash }.
 * Throws a descriptive error if format is invalid.
 */
export function parseLineRef(ref: string): LineRef {
  const normalized = normalizeLineRef(ref);
  const hashIdx = normalized.indexOf("#");

  if (hashIdx !== -1) {
    const prefix = normalized.slice(0, hashIdx);
    if (prefix.length > 0 && !/^[0-9]+$/.test(prefix)) {
      throw new Error(
        `"${prefix}" is not a line number. Use the actual line number from the read output.`,
      );
    }
  }

  const match = HASHLINE_REF_PATTERN.exec(normalized);
  if (!match) {
    throw new Error(
      `Invalid line reference format: "${ref}". Expected format: "{line_number}#{hash_id}"`,
    );
  }

  return {
    line: Number.parseInt(match[1]!, 10),
    hash: match[2]!,
  };
}
