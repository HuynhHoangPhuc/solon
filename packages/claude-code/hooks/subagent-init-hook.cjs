#!/usr/bin/env node
'use strict';
// SubagentStart(*) — inject compact context (plan path, naming pattern, principles)
try {
  const fs = require('node:fs');
  const path = require('node:path');
  const { loadConfig, isHookEnabled } = require('./lib/config-utils.cjs');

  const input = JSON.parse(fs.readFileSync('/dev/stdin', 'utf8'));
  const config = loadConfig(process.env.SOLON_PROJECT_DIR || process.cwd());

  if (!isHookEnabled(config, 'subagentInit')) process.exit(0);

  const projectDir = process.env.SOLON_PROJECT_DIR || process.cwd();
  const plansDir = path.join(projectDir, config.paths?.plans ?? 'plans');
  const reportsDir = path.join(plansDir, 'reports');

  // Read session state for active plan
  const sessionFile = process.env.SOLON_SESSION_FILE;
  let activePlan = '';
  if (sessionFile && fs.existsSync(sessionFile)) {
    try {
      const session = JSON.parse(fs.readFileSync(sessionFile, 'utf8'));
      activePlan = session.activePlan ?? '';
    } catch {}
  }

  const context = [
    '## Session',
    `- Plans: ${plansDir}`,
    `- Reports: ${reportsDir}`,
    activePlan ? `- Active plan: ${activePlan}` : '',
    '## Principles',
    '- YAGNI, KISS, DRY',
    '- Keep files under 200 lines',
    '- Fail-open on hook errors',
  ].filter(Boolean).join('\n');

  process.stdout.write(JSON.stringify({
    hookSpecificOutput: {
      hookEventName: 'SubagentStart',
      additionalContext: context,
    },
  }));
} catch {
  process.exit(0);
}
