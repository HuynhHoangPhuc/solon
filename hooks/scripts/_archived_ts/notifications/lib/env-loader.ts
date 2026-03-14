// Environment loader with cascade: process.env > ~/.claude/.env > .claude/.env
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";

/** Parse a .env file content into key-value pairs (supports comments, quoted values) */
export function parseEnvContent(content: string): Record<string, string> {
  const result: Record<string, string> = {};
  for (const line of content.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;

    const eqIndex = trimmed.indexOf("=");
    if (eqIndex === -1) continue;

    const key = trimmed.slice(0, eqIndex).trim();
    let value = trimmed.slice(eqIndex + 1).trim();

    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    } else {
      // Strip inline comments only for unquoted values
      const commentIndex = value.indexOf("#");
      if (commentIndex !== -1) value = value.slice(0, commentIndex).trim();
    }

    if (key) result[key] = value;
  }
  return result;
}

function loadEnvFile(filePath: string): Record<string, string> {
  try {
    if (fs.existsSync(filePath)) {
      const parsed = parseEnvContent(fs.readFileSync(filePath, "utf8"));
      if (Object.keys(parsed).length > 0) {
        process.stderr.write(`[env-loader] Loaded: ${filePath}\n`);
      }
      return parsed;
    }
  } catch (err) {
    process.stderr.write(`[env-loader] Failed to read ${filePath}: ${(err as Error).message}\n`);
  }
  return {};
}

/** Load environment with cascade: .claude/.env (low) → ~/.claude/.env → process.env (high) */
export function loadEnv(cwd = process.cwd()): Record<string, string> {
  const envFiles = [
    path.join(cwd, ".claude", ".env"),
    path.join(os.homedir(), ".claude", ".env"),
  ];

  let merged: Record<string, string> = {};
  for (const filePath of envFiles) {
    merged = { ...merged, ...loadEnvFile(filePath) };
  }
  return { ...merged, ...process.env } as Record<string, string>;
}
