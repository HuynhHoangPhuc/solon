import { computeLineHash, formatHashLine } from '@solon/core';

// Matches: "  42: content" or "  42| content" (Claude Code/OpenCode Read output format)
const READ_LINE_PATTERN = /^\s*(\d+)[:|\|] ?(.*)/;
const TRUNCATED_SUFFIX = '... (line truncated to 2000 chars)';

export async function enhanceReadOutput(output: { output: string }): Promise<void> {
  const lines = output.output.split('\n');
  const transformed: string[] = [];

  for (const line of lines) {
    // Skip truncated lines
    if (line.includes(TRUNCATED_SUFFIX)) {
      transformed.push(line);
      continue;
    }

    const match = line.match(READ_LINE_PATTERN);
    if (!match) {
      transformed.push(line);
      continue;
    }

    const lineNumber = parseInt(match[1] ?? '0', 10);
    const content = match[2] ?? '';
    const hash = await computeLineHash(lineNumber, content);
    transformed.push(formatHashLine(lineNumber, hash, content));
  }

  output.output = transformed.join('\n');
}
