// Config Counter - Count CLAUDE.md, rules, MCPs, hooks across user and project scopes
// All fs errors fail silently (statusline should never crash)

import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import type { ConfigCounts } from "./types.ts";

/** Extract MCP server names from a settings JSON file */
export function getMcpServerNames(filePath: string): Set<string> {
  if (!fs.existsSync(filePath)) return new Set();
  try {
    const config = JSON.parse(fs.readFileSync(filePath, "utf8"));
    if (config.mcpServers && typeof config.mcpServers === "object") {
      return new Set(Object.keys(config.mcpServers));
    }
  } catch {
    // Silent fail
  }
  return new Set();
}

/** Count MCP servers in file, optionally excluding servers already in another file */
export function countMcpServersInFile(filePath: string, excludeFrom?: string): number {
  const servers = getMcpServerNames(filePath);
  if (excludeFrom) {
    const exclude = getMcpServerNames(excludeFrom);
    for (const name of exclude) servers.delete(name);
  }
  return servers.size;
}

/** Count hook event types in a settings JSON file */
export function countHooksInFile(filePath: string): number {
  if (!fs.existsSync(filePath)) return 0;
  try {
    const config = JSON.parse(fs.readFileSync(filePath, "utf8"));
    if (config.hooks && typeof config.hooks === "object") {
      return Object.keys(config.hooks).length;
    }
  } catch {
    // Silent fail
  }
  return 0;
}

/** Count .md rule files recursively (depth-limited to 5, skips symlinks) */
export function countRulesInDir(rulesDir: string, depth = 0): number {
  if (depth > 5 || !fs.existsSync(rulesDir)) return 0;
  let count = 0;
  try {
    const entries = fs.readdirSync(rulesDir, { withFileTypes: true });
    for (const entry of entries) {
      if (entry.isSymbolicLink()) continue;
      const fullPath = path.join(rulesDir, entry.name);
      if (entry.isDirectory()) {
        count += countRulesInDir(fullPath, depth + 1);
      } else if (entry.isFile() && entry.name.endsWith(".md")) {
        count++;
      }
    }
  } catch {
    // Silent fail
  }
  return count;
}

/** Count all configs across user (~/.claude/) and project (.claude/) scopes */
export function countConfigs(cwd: string): ConfigCounts {
  let claudeMdCount = 0, rulesCount = 0, mcpCount = 0, hooksCount = 0;
  const homeDir = os.homedir();
  const claudeDir = path.join(homeDir, ".claude");

  // User scope
  if (fs.existsSync(path.join(claudeDir, "CLAUDE.md"))) claudeMdCount++;
  rulesCount += countRulesInDir(path.join(claudeDir, "rules"));
  const userSettings = path.join(claudeDir, "settings.json");
  mcpCount += countMcpServersInFile(userSettings);
  hooksCount += countHooksInFile(userSettings);
  mcpCount += countMcpServersInFile(path.join(homeDir, ".claude.json"), userSettings);

  // Project scope
  if (cwd) {
    if (fs.existsSync(path.join(cwd, "CLAUDE.md"))) claudeMdCount++;
    if (fs.existsSync(path.join(cwd, "CLAUDE.local.md"))) claudeMdCount++;
    if (fs.existsSync(path.join(cwd, ".claude", "CLAUDE.md"))) claudeMdCount++;
    if (fs.existsSync(path.join(cwd, ".claude", "CLAUDE.local.md"))) claudeMdCount++;
    rulesCount += countRulesInDir(path.join(cwd, ".claude", "rules"));
    mcpCount += countMcpServersInFile(path.join(cwd, ".mcp.json"));
    const projectSettings = path.join(cwd, ".claude", "settings.json");
    mcpCount += countMcpServersInFile(projectSettings);
    hooksCount += countHooksInFile(projectSettings);
    const localSettings = path.join(cwd, ".claude", "settings.local.json");
    mcpCount += countMcpServersInFile(localSettings);
    hooksCount += countHooksInFile(localSettings);
  }

  return { claudeMdCount, rulesCount, mcpCount, hooksCount };
}
