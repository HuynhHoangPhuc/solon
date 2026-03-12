import { z } from 'zod';

export const PluginConfigSchema = z.object({
  hooks: z.object({
    hashlineRead: z.boolean().default(true),
    outputTruncation: z.boolean().default(true),
    writeSuppression: z.boolean().default(true),
  }).optional(),
  truncation: z.object({
    maxTokens: z.number().default(50000),
    webFetchMaxTokens: z.number().default(10000),
  }).optional(),
}).strict();

export type PluginConfig = z.infer<typeof PluginConfigSchema>;

export async function loadPluginConfig(directory?: string): Promise<PluginConfig> {
  if (!directory) return {};
  const configPath = `${directory}/.solon.json`;
  try {
    const { readFile } = await import('node:fs/promises');
    const raw = await readFile(configPath, 'utf-8');
    const parsed = JSON.parse(raw) as unknown;
    return PluginConfigSchema.parse(parsed);
  } catch {
    return {};
  }
}
