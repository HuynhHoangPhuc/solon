import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    name: "@solon/core",
    include: ["tests/**/*.test.ts"],
  },
});
