export interface HookSpec {
  type: 'command';
  command: string;
  timeout?: number;
}

export interface HookBinding {
  matcher?: string;
  hooks?: HookSpec[];
}

export interface McpServerConfig {
  type: 'stdio';
  command: string;
  args?: string[];
  env?: Record<string, string>;
}

export interface ClaudeSettings {
  hooks?: Record<string, HookBinding[]>;
  mcpServers?: Record<string, McpServerConfig>;
  [key: string]: unknown;
}

export interface SolonMetadata {
  version: string;
  installedAt: string;
  installedFiles: string[];
  deletions?: string[];
}

export interface InstallOptions {
  projectDir?: string;
  force?: boolean;
}
