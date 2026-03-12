import xxhashWasm from "xxhash-wasm";

/** 16-char alphabet for encoding hash bytes as 2-char strings */
export const NIBBLE_STR = "ZPMQVRWSNKTXJBYH";

/** 256-entry lookup: HASHLINE_DICT[i] = two-char code for byte i */
export const HASHLINE_DICT: string[] = Array.from(
  { length: 256 },
  (_, i) => NIBBLE_STR[i >>> 4]! + NIBBLE_STR[i & 0x0f]!,
);

/** Singleton xxhash-wasm instance (lazy-initialized) */
let _hasher: { h32: (str: string, seed?: number) => number } | null = null;

async function getHasher(): Promise<{
  h32: (str: string, seed?: number) => number;
}> {
  if (!_hasher) _hasher = await xxhashWasm();
  return _hasher;
}

/**
 * Compute a 2-char hash code for a single file line.
 * - Strips all whitespace before hashing
 * - Lines with no alphanumeric content use lineNumber as seed
 */
export async function computeLineHash(
  lineNumber: number,
  content: string,
): Promise<string> {
  const hasher = await getHasher();
  const stripped = content.replace(/\s+/g, "");
  // Use seed=0 if line has any letter/digit, else use lineNumber (for blank/symbol-only lines)
  const hasAlphaNum = /[\p{L}\p{N}]/u.test(stripped);
  const seed = hasAlphaNum ? 0 : lineNumber;
  const hash = hasher.h32(stripped, seed) % 256;
  return HASHLINE_DICT[hash]!;
}
