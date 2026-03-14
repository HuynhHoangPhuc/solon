// Notification router — reads stdin JSON and dispatches to enabled providers (Telegram, Discord, Slack)
import { loadEnv } from "./lib/env-loader.ts";
import { telegram } from "./providers/telegram.ts";
import { discord } from "./providers/discord.ts";
import { slack } from "./providers/slack.ts";
import type { HookInput } from "./providers/telegram.ts";

const PROVIDERS = [telegram, discord, slack];

interface ProviderResult {
  provider: string;
  success: boolean;
  error?: string;
  throttled?: boolean;
}

async function readStdin(): Promise<HookInput> {
  if (process.stdin.isTTY) return {};

  return new Promise((resolve) => {
    let data = "";
    process.stdin.setEncoding("utf8");
    process.stdin.on("data", (chunk) => { data += chunk; });
    process.stdin.on("end", () => {
      if (!data.trim()) { resolve({}); return; }
      try { resolve(JSON.parse(data) as HookInput); }
      catch (err) {
        process.stderr.write(`[notify] Invalid JSON input: ${(err as Error).message}\n`);
        resolve({});
      }
    });
    process.stdin.on("error", (err) => {
      process.stderr.write(`[notify] Stdin error: ${err.message}\n`);
      resolve({});
    });
    setTimeout(() => { process.stderr.write("[notify] Stdin timeout\n"); resolve({}); }, 5000);
  });
}

function hasProviderEnv(prefix: string, env: Record<string, string>): boolean {
  return Object.keys(env).some((key) => key.startsWith(prefix + "_"));
}

async function main(): Promise<void> {
  try {
    const input = await readStdin();
    const cwd = input.cwd || process.cwd();
    const env = loadEnv(cwd);
    const results: ProviderResult[] = [];

    for (const provider of PROVIDERS) {
      const prefix = provider.name.toUpperCase();
      if (!hasProviderEnv(prefix, env)) continue;
      if (!provider.isEnabled(env)) continue;

      try {
        const result = await provider.send(input, env);
        results.push({ provider: provider.name, ...result });
        if (result.success) {
          process.stderr.write(`[notify] ${provider.name}: sent\n`);
        } else if (result.throttled) {
          process.stderr.write(`[notify] ${provider.name}: throttled\n`);
        } else {
          process.stderr.write(`[notify] ${provider.name}: failed - ${result.error}\n`);
        }
      } catch (err) {
        process.stderr.write(`[notify] ${provider.name} error: ${(err as Error).message}\n`);
        results.push({ provider: provider.name, success: false, error: (err as Error).message });
      }
    }

    if (results.length > 0) {
      const successful = results.filter((r) => r.success).length;
      process.stderr.write(`[notify] Summary: ${successful}/${results.length} succeeded\n`);
    }
  } catch (err) {
    process.stderr.write(`[notify] Fatal error: ${(err as Error).message}\n`);
  }

  process.exit(0); // Always exit 0 — never block Claude
}

main();
