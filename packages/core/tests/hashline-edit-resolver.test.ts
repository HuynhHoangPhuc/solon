import { describe, it, expect, beforeAll } from 'vitest';
import { writeFile, mkdir, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { resolveHashlineEdit } from '../src/hashline/hash-line-edit-resolver.js';
import { computeLineHash } from '../src/hashline/hash-computation.js';

const TEST_DIR = join(tmpdir(), 'solon-hashline-test');
const TEST_FILE = join(TEST_DIR, 'sample.ts');

const FILE_LINES = [
  'export function hello() {',  // line 1
  '  return "hello";',           // line 2
  '}',                           // line 3
  '',                            // line 4
  'export function world() {',  // line 5
  '  return "world";',           // line 6
  '}',                           // line 7
];

beforeAll(async () => {
  await mkdir(TEST_DIR, { recursive: true });
  await writeFile(TEST_FILE, FILE_LINES.join('\n'), 'utf-8');
});

async function ref(lineNum: number): Promise<string> {
  return `${lineNum}#${await computeLineHash(lineNum, FILE_LINES[lineNum - 1]!)}`;
}

describe('resolveHashlineEdit', () => {
  it('returns oldString unchanged when no hashline refs present', async () => {
    const input = 'no refs here, just normal text';
    const result = await resolveHashlineEdit(input, TEST_FILE);
    expect(result).toBe(input);
  });

  it('resolves a single line ref to that line content', async () => {
    const r = await ref(1);
    const result = await resolveHashlineEdit(r, TEST_FILE);
    expect(result).toBe(FILE_LINES[0]);
  });

  it('resolves a range ref to the joined lines', async () => {
    const start = await ref(1);
    const end = await ref(3);
    const result = await resolveHashlineEdit(`${start}...${end}`, TEST_FILE);
    expect(result).toBe(FILE_LINES.slice(0, 3).join('\n'));
  });

  it('throws HashlineMismatchError for stale ref', async () => {
    // Manufacture a stale ref by using a wrong hash
    const goodRef = await ref(1);
    const [lineNum] = goodRef.split('#');
    const staleRef = `${lineNum}#ZZ`; // almost certainly wrong

    // Only fails if ZZ is not the actual hash
    const actualHash = await computeLineHash(1, FILE_LINES[0]!);
    if (actualHash === 'ZZ') return; // skip if we happen to collide

    await expect(resolveHashlineEdit(staleRef, TEST_FILE)).rejects.toThrow();
  });
});
