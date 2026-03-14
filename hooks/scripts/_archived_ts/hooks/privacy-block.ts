// PreToolUse: Block access to sensitive files (.env, keys, credentials) unless user-approved
//
// Flow:
// 1. LLM tries: Read ".env"         → BLOCKED
// 2. LLM asks user for permission
// 3. User approves
// 4. LLM retries: Read "APPROVED:.env" → ALLOWED
import * as path from "node:path";
import { isHookEnabled } from "../lib/config-loader.ts";
import {
  checkPrivacy,
  isSafeFile,
  isPrivacySensitive,
  hasApprovalPrefix,
  stripApprovalPrefix,
  extractPaths,
  buildPromptData,
} from "../lib/privacy-checker.ts";

if (!isHookEnabled("privacy-block")) process.exit(0);

function formatBlockMessage(filePath: string): string {
  const basename = path.basename(filePath);
  const promptData = buildPromptData(filePath);
  return `
\x1b[36mNOTE:\x1b[0m This is not an error - this block protects sensitive data.

\x1b[33mPRIVACY BLOCK\x1b[0m: Sensitive file access requires user approval

  \x1b[33mFile:\x1b[0m ${filePath}

  This file may contain secrets (API keys, passwords, tokens).

\x1b[90m@@PRIVACY_PROMPT_START@@\x1b[0m
${JSON.stringify(promptData, null, 2)}
\x1b[90m@@PRIVACY_PROMPT_END@@\x1b[0m

  \x1b[34mClaude:\x1b[0m Use AskUserQuestion tool with the JSON above, then:
  \x1b[32mIf "Yes":\x1b[0m Use bash to read: cat "${filePath}"
  \x1b[31mIf "No":\x1b[0m  Continue without this file.
`;
}

function formatApprovalNotice(filePath: string): string {
  return `\x1b[32m✓\x1b[0m Privacy: User-approved access to ${path.basename(filePath)}`;
}

async function main(): Promise<void> {
  let input = "";
  for await (const chunk of process.stdin) {
    input += chunk;
  }

  let hookData: { tool_input?: Record<string, unknown>; tool_name?: string };
  try {
    hookData = JSON.parse(input);
  } catch {
    process.exit(0);
  }

  const { tool_input: toolInput, tool_name: toolName } = hookData;
  if (!toolInput || !toolName) { process.exit(0); }

  const result = checkPrivacy({ toolName, toolInput, options: { allowBash: true } });

  if (result.approved) {
    if (result.suspicious) {
      process.stderr.write(`\x1b[33mWARN:\x1b[0m Approved path is outside project: ${result.filePath}\n`);
    }
    process.stderr.write(formatApprovalNotice(result.filePath!) + "\n");
    process.exit(0);
  }

  if (result.isBash) {
    process.stderr.write(`\x1b[33mWARN:\x1b[0m ${result.reason}\n`);
    process.exit(0);
  }

  if (result.blocked) {
    process.stderr.write(formatBlockMessage(result.filePath!));
    process.exit(2);
  }

  process.exit(0);
}

main().catch(() => process.exit(0));

export { isSafeFile, isPrivacySensitive, hasApprovalPrefix, stripApprovalPrefix, extractPaths };
