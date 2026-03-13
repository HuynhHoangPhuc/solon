// Discord notification provider — sends rich embed messages to Discord webhooks
import * as path from "node:path";
import { send } from "../lib/sender.ts";
import type { SendResult } from "../lib/sender.ts";
import type { HookInput } from "./telegram.ts";

const COLORS = {
  Stop: 5763719,          // Green
  SubagentStop: 3447003,  // Blue
  AskUserPrompt: 15844367, // Yellow
  default: 10070709,      // Gray
} as const;

function getProjectName(cwd?: string): string {
  if (!cwd) return "Unknown";
  return path.basename(cwd) || "Unknown";
}

function formatTimestamp(): string {
  return new Date().toLocaleTimeString("en-US", {
    hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false,
  });
}

function truncateSessionId(sessionId?: string): string {
  if (!sessionId) return "N/A";
  return sessionId.length > 8 ? `${sessionId.slice(0, 8)}...` : sessionId;
}

interface DiscordEmbed {
  title: string;
  description: string;
  color: number;
  timestamp: string;
  footer: { text: string };
  fields: { name: string; value: string; inline: boolean }[];
}

function buildEmbed(input: HookInput): DiscordEmbed {
  const hookType = input.hook_event_name || "unknown";
  const cwd = input.cwd || "";
  const sessionId = input.session_id || "";
  const projectName = getProjectName(cwd);
  const color = (COLORS as Record<string, number>)[hookType] ?? COLORS.default;
  const sessionDisplay = `\`${truncateSessionId(sessionId)}\``;
  const locationField = { name: "📍 Location", value: `\`${cwd || "Unknown"}\``, inline: false };

  switch (hookType) {
    case "Stop":
      return {
        title: "Claude Code Session Complete",
        description: "Session completed successfully",
        color, timestamp: new Date().toISOString(),
        footer: { text: `Project • ${projectName}` },
        fields: [
          { name: "⏰ Time", value: formatTimestamp(), inline: true },
          { name: "🆔 Session", value: sessionDisplay, inline: true },
          locationField,
        ],
      };

    case "SubagentStop":
      return {
        title: "Claude Code Subagent Complete",
        description: "Specialized agent completed its task",
        color, timestamp: new Date().toISOString(),
        footer: { text: `Project • ${projectName}` },
        fields: [
          { name: "⏰ Time", value: formatTimestamp(), inline: true },
          { name: "🔧 Agent Type", value: input.agent_type || "unknown", inline: true },
          { name: "🆔 Session", value: sessionDisplay, inline: true },
          locationField,
        ],
      };

    case "AskUserPrompt":
      return {
        title: "Claude Code Needs Input",
        description: "Claude is waiting for user input",
        color, timestamp: new Date().toISOString(),
        footer: { text: `Project • ${projectName}` },
        fields: [
          { name: "⏰ Time", value: formatTimestamp(), inline: true },
          { name: "🆔 Session", value: sessionDisplay, inline: true },
          locationField,
        ],
      };

    default:
      return {
        title: "Claude Code Event",
        description: "Claude Code event triggered",
        color, timestamp: new Date().toISOString(),
        footer: { text: `Project • ${projectName}` },
        fields: [
          { name: "⏰ Time", value: formatTimestamp(), inline: true },
          { name: "📋 Event", value: hookType, inline: true },
          { name: "🆔 Session", value: sessionDisplay, inline: true },
          locationField,
        ],
      };
  }
}

export const discord = {
  name: "discord",
  isEnabled: (env: Record<string, string>) => !!env.DISCORD_WEBHOOK_URL,
  send: async (input: HookInput, env: Record<string, string>): Promise<SendResult> => {
    if (!env.DISCORD_WEBHOOK_URL) return { success: false, error: "DISCORD_WEBHOOK_URL not configured" };
    return send("discord", env.DISCORD_WEBHOOK_URL, { embeds: [buildEmbed(input)] });
  },
};
