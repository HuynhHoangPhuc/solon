// OpenCode plugin — Solon integration
// Written as framework-agnostic export; no direct @opencode-ai/plugin import
// to avoid hard dependency (it's a peerDep)

import { SOLON_AGENTS } from "./agents/agent-definitions.js";
import { TRUNCATABLE_TOOLS } from "./config/constants.js";
import { loadPluginConfig } from "./config/plugin-config-schema.js";
import { enhanceReadOutput } from "./hooks/hashline-read-enhancer.js";
import { truncateToolOutput } from "./hooks/output-truncator.js";
import { suppressWriteOutput } from "./hooks/write-output-suppressor.js";
import { hashlineEditToolDef } from "./tools/hashline-edit-tool.js";

export type { PluginConfig } from "./config/plugin-config-schema.js";
export { SOLON_AGENTS } from "./agents/agent-definitions.js";
export { hashlineEditToolDef } from "./tools/hashline-edit-tool.js";

// Plugin context type (subset of what OpenCode provides)
interface PluginContext {
  directory?: string;
}

// Tool hook input/output types
interface ToolHookInput {
  tool?: string;
  tool_name?: string;
  tool_input?: Record<string, unknown>;
}

interface ToolHookOutput {
  output: string;
}

export async function createSolonPlugin(ctx: PluginContext = {}) {
  const config = await loadPluginConfig(ctx.directory);

  return {
    name: "solon" as const,

    agent: SOLON_AGENTS,

    tool: {
      hashline_edit: hashlineEditToolDef,
    },

    mcp: {
      solon: { type: "local" as const, command: ["npx", "@solon/mcp-server"] },
    },

    async "tool.execute.after"(input: ToolHookInput, output: ToolHookOutput) {
      const toolName = (input.tool ?? input.tool_name ?? "").toLowerCase();

      if (toolName === "read" && config.hooks?.hashlineRead !== false) {
        await enhanceReadOutput(output);
        return;
      }

      if (toolName === "write" && config.hooks?.writeSuppression !== false) {
        await suppressWriteOutput(input, output);
        return;
      }

      if (
        TRUNCATABLE_TOOLS.has(toolName) &&
        config.hooks?.outputTruncation !== false
      ) {
        truncateToolOutput(toolName, output, config);
      }
    },

    async config(opencodeConfig: Record<string, unknown>) {
      // Merge agents into opencode config
      const agents = (opencodeConfig.agent ?? {}) as Record<string, unknown>;
      Object.assign(agents, SOLON_AGENTS);
      opencodeConfig.agent = agents;

      // Merge MCP
      const mcp = (opencodeConfig.mcp ?? {}) as Record<string, unknown>;
      mcp.solon = { type: "local", command: ["npx", "@solon/mcp-server"] };
      opencodeConfig.mcp = mcp;
    },
  };
}

// Default export as OpenCode plugin function
export default createSolonPlugin;
