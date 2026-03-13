// Slack notification provider — Block Kit format via incoming webhooks
import * as path from "node:path";
import { send } from "../lib/sender.ts";
import type { SendResult } from "../lib/sender.ts";
import type { HookInput } from "./telegram.ts";

function getTitle(hookType: string): string {
  switch (hookType) {
    case "Stop": return "Claude Code Session Complete";
    case "SubagentStop": return "Claude Code Subagent Complete";
    case "AskUserPrompt": return "Claude Code Needs Input";
    default: return "Claude Code Event";
  }
}

type Block = Record<string, unknown>;

function buildBlocks(input: HookInput, hookType: string, projectName: string, sessionId: string): Block[] {
  const timestamp = new Date().toLocaleString();
  const cwd = input.cwd || "Unknown";

  const blocks: Block[] = [
    { type: "header", text: { type: "plain_text", text: getTitle(hookType) } },
    {
      type: "section",
      fields: [
        { type: "mrkdwn", text: `*Project:*\n${projectName}` },
        { type: "mrkdwn", text: `*Time:*\n${timestamp}` },
        { type: "mrkdwn", text: `*Session:*\n\`${sessionId}...\`` },
        { type: "mrkdwn", text: `*Event:*\n${hookType}` },
      ],
    },
    { type: "divider" },
    { type: "context", elements: [{ type: "mrkdwn", text: `📍 \`${cwd}\`` }] },
  ];

  if (hookType === "SubagentStop" && input.agent_type) {
    blocks.splice(2, 0, {
      type: "section",
      text: { type: "mrkdwn", text: `*Agent Type:* ${input.agent_type}` },
    });
  }

  return blocks;
}

export const slack = {
  name: "slack",
  isEnabled: (env: Record<string, string>) => !!env.SLACK_WEBHOOK_URL,
  send: async (input: HookInput, env: Record<string, string>): Promise<SendResult> => {
    const hookType = input.hook_event_name || "unknown";
    const projectDir = input.cwd || "";
    const projectName = path.basename(projectDir) || "Unknown";
    const sessionId = (input.session_id || "").slice(0, 8);

    const payload = {
      text: `Claude Code: ${hookType} in ${projectName}`,
      blocks: buildBlocks(input, hookType, projectName, sessionId),
    };
    return send("slack", env.SLACK_WEBHOOK_URL, payload);
  },
};
