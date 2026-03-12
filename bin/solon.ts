#!/usr/bin/env node
/**
 * Solon CLI — dual-platform plugin installer
 * Usage:
 *   solon install --claude-code [--profile minimal|default|full]
 *   solon install --opencode
 *   solon uninstall --claude-code
 *   solon version
 */
import { installClaudeCode, uninstallClaudeCode } from '../packages/claude-code/src/index.js';

const args = process.argv.slice(2);
const command = args[0];
const flags = new Set(args.slice(1));

async function main() {
  if (!command || command === 'help' || command === '--help') {
    printHelp();
    return;
  }

  if (command === 'version' || command === '--version' || command === '-v') {
    console.log('solon v0.1.0');
    return;
  }

  if (command === 'install') {
    if (flags.has('--claude-code')) {
      console.log('Installing Solon for Claude Code...');
      await installClaudeCode({ projectDir: process.cwd() });
      console.log('\n✓ Done! Restart Claude Code to activate hooks.');
    } else if (flags.has('--opencode')) {
      console.log('OpenCode integration: add "@solon/opencode" as a plugin in your opencode config.');
      console.log('See: https://github.com/your-org/solon#opencode');
    } else {
      console.error('Error: specify --claude-code or --opencode');
      process.exit(1);
    }
    return;
  }

  if (command === 'uninstall') {
    if (flags.has('--claude-code')) {
      console.log('Uninstalling Solon from Claude Code...');
      await uninstallClaudeCode({ projectDir: process.cwd() });
    } else {
      console.error('Error: specify --claude-code');
      process.exit(1);
    }
    return;
  }

  console.error(`Unknown command: ${command}`);
  printHelp();
  process.exit(1);
}

function printHelp() {
  console.log(`
solon — dual-platform plugin for Claude Code and OpenCode

Usage:
  solon install --claude-code     Install hooks + MCP server for Claude Code
  solon install --opencode        Instructions for OpenCode integration
  solon uninstall --claude-code   Remove Solon from Claude Code
  solon version                   Show version
`);
}

main().catch((err: unknown) => {
  const message = err instanceof Error ? err.message : String(err);
  console.error('Error:', message);
  process.exit(1);
});
