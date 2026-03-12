import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    name: '@solon/mcp-server',
    include: ['tests/**/*.test.ts'],
  },
});
