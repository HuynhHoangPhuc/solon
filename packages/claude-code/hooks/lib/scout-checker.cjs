'use strict';
const fs = require('node:fs');
const path = require('node:path');

const DEFAULT_BLOCKED_DIRS = [
  'node_modules', 'dist', '.git', '.next', 'build',
  'coverage', '.turbo', '.cache', '__pycache__',
  'target', 'vendor', '.venv', 'venv', '.yarn',
];

// Build commands that should always be allowed despite broad patterns
const BUILD_COMMAND_PATTERNS = [
  /\bnpm\s+run\b/, /\bpnpm\s+run\b/, /\byarn\s+run\b/, /\bbun\s+run\b/,
  /\bcargo\s+build\b/, /\bgo\s+build\b/, /\bmake\b/,
];

// Patterns that indicate a broad/dangerous glob
const BROAD_GLOB_PATTERNS = [
  /^\*\*\/\*$/, /^\*$/, /^\.\/\*$/, /^\.\*$/, /^\*\.\w+$/,
];

function isBuildCommand(input) {
  if (typeof input !== 'string') return false;
  return BUILD_COMMAND_PATTERNS.some(p => p.test(input));
}

function isBlockedDirectory(targetPath) {
  const parts = targetPath.replace(/\\/g, '/').split('/');
  return parts.some(part => DEFAULT_BLOCKED_DIRS.includes(part));
}

function hasBroadGlobPattern(pattern) {
  if (typeof pattern !== 'string') return false;
  return BROAD_GLOB_PATTERNS.some(p => p.test(pattern));
}

function loadSolonIgnore(projectDir) {
  const ignoreFile = path.join(projectDir, '.solon-ignore');
  if (!fs.existsSync(ignoreFile)) return [];
  try {
    return fs.readFileSync(ignoreFile, 'utf8')
      .split('\n')
      .map(l => l.trim())
      .filter(l => l && !l.startsWith('#'));
  } catch { return []; }
}

function checkScoutBlock(toolName, toolInput, projectDir) {
  // Allow build commands
  if (toolName === 'Bash' && isBuildCommand(toolInput.command)) {
    return { blocked: false };
  }

  // Check path-based inputs for blocked directories
  const pathInputs = ['path', 'file_path', 'filePath', 'pattern'];
  for (const key of pathInputs) {
    const val = toolInput[key];
    if (typeof val === 'string' && isBlockedDirectory(val)) {
      return {
        blocked: true,
        reason: `Path "${val}" targets a blocked directory. Avoid scanning node_modules, dist, .git, etc.`,
        suggestion: 'Use a more specific path that excludes build artifacts and dependencies.',
      };
    }
    if (typeof val === 'string' && hasBroadGlobPattern(val)) {
      return {
        blocked: true,
        reason: `Pattern "${val}" is too broad and would scan too many files.`,
        suggestion: 'Use a specific path or pattern like "src/**/*.ts" instead of broad wildcards.',
      };
    }
  }

  // Check custom .solon-ignore patterns if projectDir provided
  if (projectDir) {
    const customPatterns = loadSolonIgnore(projectDir);
    for (const key of pathInputs) {
      const val = toolInput[key];
      if (typeof val === 'string') {
        for (const pattern of customPatterns) {
          if (val.includes(pattern)) {
            return {
              blocked: true,
              reason: `Path "${val}" matches custom blocked pattern "${pattern}" from .solon-ignore.`,
            };
          }
        }
      }
    }
  }

  return { blocked: false };
}

module.exports = { checkScoutBlock, isBlockedDirectory, loadSolonIgnore };
