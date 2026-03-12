#!/usr/bin/env node
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  solonReadInputSchema,
  solonReadHandler,
  SOLON_READ_DESCRIPTION,
} from './tools/solon-read-tool.js';

const server = new McpServer({
  name: 'solon',
  version: '0.1.0',
});

server.registerTool(
  'solon_read',
  {
    description: SOLON_READ_DESCRIPTION,
    inputSchema: solonReadInputSchema,
  },
  solonReadHandler,
);

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

main().catch((err) => {
  console.error('solon-mcp fatal error:', err);
  process.exit(1);
});
