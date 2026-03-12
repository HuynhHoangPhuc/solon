import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    projects: [
      "packages/core/vitest.config.ts",
      "packages/mcp-server/vitest.config.ts",
      "packages/claude-code/vitest.config.ts",
      "packages/opencode/vitest.config.ts",
    ],
  },
});
