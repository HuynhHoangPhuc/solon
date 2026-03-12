import { estimateTokens, truncateOutput } from "@solon/core";
import {
  DEFAULT_MAX_TOKENS,
  WEBFETCH_MAX_TOKENS,
} from "../config/constants.js";
import type { PluginConfig } from "../config/plugin-config-schema.js";

export function truncateToolOutput(
  toolName: string,
  output: { output: string },
  config: PluginConfig,
): void {
  const isWebFetch = toolName.toLowerCase().includes("webfetch");
  const maxTokens = isWebFetch
    ? (config.truncation?.webFetchMaxTokens ?? WEBFETCH_MAX_TOKENS)
    : (config.truncation?.maxTokens ?? DEFAULT_MAX_TOKENS);

  const tokens = estimateTokens(output.output);
  if (tokens <= maxTokens) return;

  output.output = truncateOutput(output.output, maxTokens);
}
