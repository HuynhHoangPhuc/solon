import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    name: "@solon/claude-code",
    include: ["tests/**/*.test.ts"],
  },
});
