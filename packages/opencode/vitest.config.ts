import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    name: "@solon/opencode",
    include: ["tests/**/*.test.ts"],
  },
});
