import { checkBroadPattern } from './broad-pattern-detector.js';
import { isBlockedDirectory, loadSolonIgnore } from './directory-blocklist.js';

export interface ScoutBlockResult {
  blocked: boolean;
  reason?: string;
  suggestion?: string;
}

/**
 * Central scout-block check for tool calls.
 * Evaluates both broad glob patterns and blocked directory paths.
 */
export async function checkScoutBlock(
  toolName: string,
  toolInput: Record<string, unknown>,
  projectDir?: string
): Promise<ScoutBlockResult> {
  const customPatterns = projectDir ? await loadSolonIgnore(projectDir) : [];

  // Check glob pattern (Glob tool uses "pattern" field)
  const pattern = typeof toolInput['pattern'] === 'string' ? toolInput['pattern'] : null;
  if (pattern !== null) {
    const result = checkBroadPattern(pattern, toolName);
    if (result.blocked) {
      return {
        blocked: true,
        reason: `Overly broad glob pattern: "${result.pattern}"`,
        ...(result.suggestion !== undefined ? { suggestion: result.suggestion } : {}),
      };
    }
  }

  // Check path fields for blocked directories
  const pathFields = ['path', 'file_path', 'directory', 'dir'];
  for (const field of pathFields) {
    const value = typeof toolInput[field] === 'string' ? toolInput[field] : null;
    if (value !== null && isBlockedDirectory(value, customPatterns)) {
      return {
        blocked: true,
        reason: `Path contains a blocked directory: "${value}"`,
        suggestion: 'Use a path outside blocked directories (node_modules, dist, .git, etc.)',
      };
    }
  }

  return { blocked: false };
}
