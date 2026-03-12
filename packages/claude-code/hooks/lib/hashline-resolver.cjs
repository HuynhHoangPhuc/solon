'use strict';
const fs = require('node:fs');

const NIBBLE_STR = 'ZPMQVRWSNKTXJBYH';
const HASHLINE_DICT = Array.from({ length: 256 }, (_, i) =>
  NIBBLE_STR[i >>> 4] + NIBBLE_STR[i & 0x0f]
);

// xxHash32 pure JS implementation (no WASM needed for hooks - uses sync API)
// Based on xxHash32 spec: https://github.com/Cyan4973/xxHash/blob/dev/doc/xxhash_spec.md
const PRIME1 = 0x9e3779b1;
const PRIME2 = 0x85ebca77;
const PRIME3 = 0xc2b2ae3d;
const PRIME4 = 0x27d4eb2f;
const PRIME5 = 0x165667b1;

function rotl32(x, r) {
  return ((x << r) | (x >>> (32 - r))) >>> 0;
}

function xxHash32(str, seed = 0) {
  const buf = Buffer.from(str, 'utf8');
  const len = buf.length;
  let h32;
  let i = 0;

  if (len >= 16) {
    let v1 = (seed + PRIME1 + PRIME2) >>> 0;
    let v2 = (seed + PRIME2) >>> 0;
    let v3 = seed >>> 0;
    let v4 = (seed - PRIME1) >>> 0;

    do {
      v1 = (rotl32((v1 + (Math.imul(buf.readUInt32LE(i), PRIME2) >>> 0)) >>> 0, 13) * PRIME1) >>> 0; i += 4;
      v2 = (rotl32((v2 + (Math.imul(buf.readUInt32LE(i), PRIME2) >>> 0)) >>> 0, 13) * PRIME1) >>> 0; i += 4;
      v3 = (rotl32((v3 + (Math.imul(buf.readUInt32LE(i), PRIME2) >>> 0)) >>> 0, 13) * PRIME1) >>> 0; i += 4;
      v4 = (rotl32((v4 + (Math.imul(buf.readUInt32LE(i), PRIME2) >>> 0)) >>> 0, 13) * PRIME1) >>> 0; i += 4;
    } while (i <= len - 16);

    h32 = (rotl32(v1, 1) + rotl32(v2, 7) + rotl32(v3, 12) + rotl32(v4, 18)) >>> 0;
  } else {
    h32 = (seed + PRIME5) >>> 0;
  }

  h32 = (h32 + len) >>> 0;

  while (i <= len - 4) {
    h32 = (rotl32((h32 + (Math.imul(buf.readUInt32LE(i), PRIME3) >>> 0)) >>> 0, 17) * PRIME4) >>> 0;
    i += 4;
  }

  while (i < len) {
    h32 = (rotl32((h32 + (Math.imul(buf[i], PRIME5) >>> 0)) >>> 0, 11) * PRIME1) >>> 0;
    i++;
  }

  h32 = (Math.imul(h32 ^ (h32 >>> 15), PRIME2)) >>> 0;
  h32 = (Math.imul(h32 ^ (h32 >>> 13), PRIME3)) >>> 0;
  h32 = (h32 ^ (h32 >>> 16)) >>> 0;

  return h32;
}

const RE_SIGNIFICANT = /[\p{L}\p{N}]/u;

function computeLineHash(lineNumber, content) {
  const stripped = content.replace(/\s+/g, '');
  const seed = RE_SIGNIFICANT.test(stripped) ? 0 : lineNumber;
  const hash = xxHash32(stripped, seed) % 256;
  return HASHLINE_DICT[hash];
}

const HASHLINE_REF_PATTERN = /(\d+)#([ZPMQVRWSNKTXJBYH]{2})/g;
const HASHLINE_SINGLE_REF = /^(\d+)#([ZPMQVRWSNKTXJBYH]{2})$/;
const HASHLINE_RANGE_REF = /^(\d+)#([ZPMQVRWSNKTXJBYH]{2})\.\.\.(\d+)#([ZPMQVRWSNKTXJBYH]{2})$/;

function resolveHashlineEdit(oldString, filePath) {
  if (!HASHLINE_REF_PATTERN.test(oldString)) return null; // no refs
  HASHLINE_REF_PATTERN.lastIndex = 0; // reset after test()

  const fileContent = fs.readFileSync(filePath, 'utf8');
  const fileLines = fileContent.split('\n');

  // Check for range ref pattern (full old_string is a range ref)
  const rangeMatch = oldString.trim().match(HASHLINE_RANGE_REF);
  if (rangeMatch) {
    const startLine = parseInt(rangeMatch[1], 10);
    const startHash = rangeMatch[2];
    const endLine = parseInt(rangeMatch[3], 10);
    const endHash = rangeMatch[4];

    // Validate start hash
    const actualStartHash = computeLineHash(startLine, fileLines[startLine - 1] ?? '');
    if (actualStartHash !== startHash) {
      return { error: `Hash mismatch at line ${startLine}: expected ${startHash}, got ${actualStartHash}` };
    }
    // Validate end hash
    const actualEndHash = computeLineHash(endLine, fileLines[endLine - 1] ?? '');
    if (actualEndHash !== endHash) {
      return { error: `Hash mismatch at line ${endLine}: expected ${endHash}, got ${actualEndHash}` };
    }

    return { resolved: fileLines.slice(startLine - 1, endLine).join('\n') };
  }

  // Check for single ref
  const singleMatch = oldString.trim().match(HASHLINE_SINGLE_REF);
  if (singleMatch) {
    const lineNum = parseInt(singleMatch[1], 10);
    const hash = singleMatch[2];
    const actualHash = computeLineHash(lineNum, fileLines[lineNum - 1] ?? '');
    if (actualHash !== hash) {
      return { error: `Hash mismatch at line ${lineNum}: expected ${hash}, got ${actualHash}` };
    }
    return { resolved: fileLines[lineNum - 1] ?? '' };
  }

  // Mixed refs — not a pure ref, return null to pass through
  return null;
}

module.exports = { resolveHashlineEdit, computeLineHash, HASHLINE_DICT };
