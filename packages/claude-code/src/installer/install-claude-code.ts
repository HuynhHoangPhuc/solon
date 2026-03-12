import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { generateSettings } from './settings-generator.js';
import { writeMetadata } from './metadata-manager.js';
import { copyDirectory } from './file-copier.js';
import type { ClaudeSettings, InstallOptions } from './types.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
// hooks directory is at ../../hooks relative to dist/installer/
const HOOKS_SRC = join(__dirname, '..', '..', '..', 'hooks');

export async function installClaudeCode(options: InstallOptions = {}): Promise<void> {
  const projectDir = options.projectDir ?? process.cwd();
  const claudeDir = join(projectDir, '.claude');
  const solonHooksDir = join(claudeDir, 'hooks', 'solon');

  // Create directories
  await mkdir(solonHooksDir, { recursive: true });

  // Copy hooks
  const copiedFiles = await copyDirectory(HOOKS_SRC, solonHooksDir);

  // Read existing settings.json
  const settingsPath = join(claudeDir, 'settings.json');
  let existingSettings: ClaudeSettings = {};
  try {
    const raw = await readFile(settingsPath, 'utf-8');
    existingSettings = JSON.parse(raw) as ClaudeSettings;
  } catch {
    // No existing settings — start fresh
  }

  // Generate merged settings
  const newSettings = generateSettings(existingSettings);
  await writeFile(settingsPath, JSON.stringify(newSettings, null, 2), 'utf-8');

  // Write metadata
  await writeMetadata(claudeDir, copiedFiles);

  console.log(`Solon installed to ${claudeDir}`);
  console.log(`  Hooks: ${copiedFiles.length} files`);
  console.log(`  Settings: ${settingsPath}`);
}

export async function uninstallClaudeCode(options: InstallOptions = {}): Promise<void> {
  const projectDir = options.projectDir ?? process.cwd();
  const claudeDir = join(projectDir, '.claude');
  const settingsPath = join(claudeDir, 'settings.json');

  // Remove solon hooks from settings
  try {
    const raw = await readFile(settingsPath, 'utf-8');
    const settings = JSON.parse(raw) as ClaudeSettings;
    const cleaned = removeSolonFromSettings(settings);
    await writeFile(settingsPath, JSON.stringify(cleaned, null, 2), 'utf-8');
  } catch {}

  console.log(`Solon uninstalled from ${claudeDir}`);
}

function removeSolonFromSettings(settings: ClaudeSettings): ClaudeSettings {
  const cleaned = { ...settings };
  if (cleaned.hooks) {
    for (const [event, bindings] of Object.entries(cleaned.hooks)) {
      cleaned.hooks[event] = bindings.filter(b =>
        !b.hooks?.some(h => h.command?.includes('solon'))
      );
    }
  }
  if (cleaned.mcpServers?.['solon']) {
    const { solon: _solon, ...rest } = cleaned.mcpServers;
    cleaned.mcpServers = rest;
  }
  return cleaned;
}
