import { readFile } from 'node:fs/promises';
import { join } from 'node:path';

/** Directories that should never be scanned by scout/glob tools */
const DEFAULT_BLOCKED_DIRS = [
  'node_modules',
  'dist',
  '.git',
  '.next',
  'build',
  'coverage',
  '.turbo',
  '.cache',
  '__pycache__',
  'target',
  'vendor',
  '.venv',
  'venv',
];

/**
 * Check whether a file/directory path contains a blocked directory segment.
 * Optionally provide custom patterns (substring match).
 */
export function isBlockedDirectory(path: string, customPatterns?: string[]): boolean {
  const blocked = customPatterns
    ? [...DEFAULT_BLOCKED_DIRS, ...customPatterns]
    : DEFAULT_BLOCKED_DIRS;

  const segments = path.replace(/\\/g, '/').split('/');
  return segments.some(seg => blocked.includes(seg));
}

/**
 * Load custom ignore patterns from a .solonignore file in the project root.
 * Returns empty array if file does not exist.
 */
export async function loadSolonIgnore(projectDir: string): Promise<string[]> {
  try {
    const content = await readFile(join(projectDir, '.solonignore'), 'utf-8');
    return content
      .split('\n')
      .map((line: string) => line.trim())
      .filter((line: string) => line.length > 0 && !line.startsWith('#'));
  } catch {
    return [];
  }
}
