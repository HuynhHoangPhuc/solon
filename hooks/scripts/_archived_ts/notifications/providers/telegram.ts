// Telegram notification provider — uses Telegram Bot API with Markdown formatting
import * as path from "node:path";
import { send } from "../lib/sender.ts";
import type { SendResult } from "../lib/sender.ts";

export interface HookInput {
  hook_event_name?: string;
  cwd?: string;
  session_id?: string;
  agent_type?: string;
}

function getTimestamp(): string {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ` +
    `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`;
}

function formatMessage(input: HookInput): string {
  const hookType = input.hook_event_name || "unknown";
  const projectDir = input.cwd || "";
  const sessionId = input.session_id || "";
  const projectName = projectDir ? path.basename(projectDir) : "unknown";
  const timestamp = getTimestamp();
  const sessionDisplay = sessionId ? `${sessionId.slice(0, 8)}...` : "N/A";

  switch (hookType) {
    case "Stop":
      return `🚀 *Project Task Completed*\n\n📅 *Time:* ${timestamp}\n📁 *Project:* ${projectName}\n🆔 *Session:* ${sessionDisplay}\n\n📍 *Location:* \`${projectDir}\``;

    case "SubagentStop": {
      const agentType = input.agent_type || "unknown";
      return `🤖 *Project Subagent Completed*\n\n📅 *Time:* ${timestamp}\n📁 *Project:* ${projectName}\n🔧 *Agent Type:* ${agentType}\n🆔 *Session:* ${sessionDisplay}\n\nSpecialized agent completed its task.\n\n📍 *Location:* \`${projectDir}\``;
    }

    case "AskUserPrompt":
      return `💬 *User Input Needed*\n\n📅 *Time:* ${timestamp}\n📁 *Project:* ${projectName}\n🆔 *Session:* ${sessionDisplay}\n\nClaude is waiting for your input.\n\n📍 *Location:* \`${projectDir}\``;

    default:
      return `📝 *Project Code Event*\n\n📅 *Time:* ${timestamp}\n📁 *Project:* ${projectName}\n📋 *Event:* ${hookType}\n🆔 *Session:* ${sessionDisplay}\n\n📍 *Location:* \`${projectDir}\``;
  }
}

export const telegram = {
  name: "telegram",
  isEnabled: (env: Record<string, string>) => !!(env.TELEGRAM_BOT_TOKEN && env.TELEGRAM_CHAT_ID),
  send: async (input: HookInput, env: Record<string, string>): Promise<SendResult> => {
    const message = formatMessage(input);
    const url = `https://api.telegram.org/bot${env.TELEGRAM_BOT_TOKEN}/sendMessage`;
    return send("telegram", url, {
      chat_id: env.TELEGRAM_CHAT_ID,
      text: message,
      parse_mode: "Markdown",
      disable_web_page_preview: true,
    });
  },
};
