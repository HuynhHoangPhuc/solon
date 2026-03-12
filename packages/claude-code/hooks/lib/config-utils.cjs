'use strict';
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');

// Default config
const DEFAULT_CONFIG = {
  hooks: {
    hashlineEdit: true,
    outputTruncation: true,
    scoutBlock: true,
    writeSuppression: true,
    subagentInit: true,
    dedupGuard: true,
  },
  truncation: {
    maxTokens: 50000,
    webFetchMaxTokens: 10000,
  },
};

function deepMerge(base, override) {
  const result = { ...base };
  for (const [key, value] of Object.entries(override)) {
    if (value && typeof value === 'object' && !Array.isArray(value) && typeof result[key] === 'object') {
      result[key] = deepMerge(result[key], value);
    } else {
      result[key] = value;
    }
  }
  return result;
}

function loadConfig(projectDir) {
  let config = { ...DEFAULT_CONFIG };

  // Load global config
  const globalConfigPath = path.join(os.homedir(), '.solon.json');
  if (fs.existsSync(globalConfigPath)) {
    try {
      const global = JSON.parse(fs.readFileSync(globalConfigPath, 'utf8'));
      config = deepMerge(config, global);
    } catch {}
  }

  // Load local config (local wins)
  if (projectDir) {
    const localConfigPath = path.join(projectDir, '.solon.json');
    if (fs.existsSync(localConfigPath)) {
      try {
        const local = JSON.parse(fs.readFileSync(localConfigPath, 'utf8'));
        config = deepMerge(config, local);
      } catch {}
    }
  }

  return config;
}

function isHookEnabled(config, hookName) {
  return config?.hooks?.[hookName] !== false;
}

module.exports = { loadConfig, isHookEnabled, DEFAULT_CONFIG };
