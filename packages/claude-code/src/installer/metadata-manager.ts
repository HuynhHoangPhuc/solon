import { readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import type { SolonMetadata } from './types.js';

const METADATA_FILE = 'metadata.json';
const SOLON_VERSION = '0.1.0';

export async function readMetadata(claudeDir: string): Promise<SolonMetadata | null> {
  const metaPath = join(claudeDir, 'hooks', 'solon', METADATA_FILE);
  try {
    const raw = await readFile(metaPath, 'utf-8');
    return JSON.parse(raw) as SolonMetadata;
  } catch {
    return null;
  }
}

export async function writeMetadata(claudeDir: string, installedFiles: string[]): Promise<void> {
  const metaPath = join(claudeDir, 'hooks', 'solon', METADATA_FILE);
  const metadata: SolonMetadata = {
    version: SOLON_VERSION,
    installedAt: new Date().toISOString(),
    installedFiles,
  };
  await writeFile(metaPath, JSON.stringify(metadata, null, 2), 'utf-8');
}
