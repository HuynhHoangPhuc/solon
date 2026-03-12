import { readFile } from 'node:fs/promises';
import { computeLineHash, formatHashLine } from '@solon/core';

const BINARY_CHECK_BYTES = 8192;
const MAX_LINE_LENGTH = 2000;
const TRUNCATED_LINE_SUFFIX = '... (line truncated to 2000 chars)';

export interface FileReadOptions {
  offset?: number; // 1-based start line
  limit?: number;  // max lines to return
}

export interface FileReadResult {
  content: string;
  totalLines: number;
  isBinary: boolean;
}

export async function readFileWithHashlines(
  filePath: string,
  options: FileReadOptions = {}
): Promise<FileReadResult> {
  const buffer = await readFile(filePath);

  // Binary detection: check for null bytes in first 8KB
  const checkEnd = Math.min(buffer.length, BINARY_CHECK_BYTES);
  for (let i = 0; i < checkEnd; i++) {
    if (buffer[i] === 0) {
      return { content: 'Binary file, cannot display', totalLines: 0, isBinary: true };
    }
  }

  const text = buffer.toString('utf-8');
  const rawLines = text.split('\n');
  // Remove trailing empty line artifact from split
  if (rawLines.length > 0 && rawLines[rawLines.length - 1] === '') {
    rawLines.pop();
  }

  const totalLines = rawLines.length;
  const offset = options.offset ?? 1;
  const startIdx = Math.max(0, offset - 1);
  const endIdx = options.limit != null
    ? Math.min(totalLines, startIdx + options.limit)
    : totalLines;

  // Build hashline output for each line
  const lines: string[] = [];
  for (let i = startIdx; i < endIdx; i++) {
    const lineNumber = i + 1;
    const rawContent = rawLines[i] ?? '';

    // Handle long lines (skip hashline for truncated lines, matching Claude Code behavior)
    if (rawContent.length > MAX_LINE_LENGTH) {
      const truncated = rawContent.slice(0, MAX_LINE_LENGTH);
      lines.push(`${lineNumber}| ${truncated}${TRUNCATED_LINE_SUFFIX}`);
      continue;
    }

    const hash = await computeLineHash(lineNumber, rawContent);
    lines.push(formatHashLine(lineNumber, hash, rawContent));
  }

  return { content: lines.join('\n'), totalLines, isBinary: false };
}
