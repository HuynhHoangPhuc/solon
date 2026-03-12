import { estimateTokens } from './token-estimator.js';
import { DEFAULT_PRESERVE_HEADER_LINES } from './truncation-constants.js';

/**
 * Truncate tool output to fit within a token budget.
 * - Returns text unchanged if already within budget
 * - Always preserves first N header lines
 * - Appends a truncation notice with remaining line count
 */
export function truncateOutput(
  text: string,
  maxTokens: number,
  preserveHeaderLines: number = DEFAULT_PRESERVE_HEADER_LINES
): string {
  if (estimateTokens(text) <= maxTokens) return text;

  const lines = text.split('\n');

  // Edge case: fewer lines than header threshold — char-slice directly
  if (lines.length <= preserveHeaderLines) {
    const maxChars = maxTokens * 4;
    return (
      text.slice(0, maxChars) +
      '\n[Output truncated due to context window limit]'
    );
  }

  const headerLines = lines.slice(0, preserveHeaderLines);
  const bodyLines = lines.slice(preserveHeaderLines);

  // Count tokens used by headers (include newline cost)
  const headerText = headerLines.join('\n') + '\n';
  let usedTokens = estimateTokens(headerText);
  const noticeText = '\n\n[X more lines truncated due to context window limit]';
  const noticeTokens = estimateTokens(noticeText);
  const budget = maxTokens - usedTokens - noticeTokens;

  const kept: string[] = [];
  let remaining = budget;

  for (const line of bodyLines) {
    const cost = estimateTokens(line + '\n');
    if (remaining - cost < 0) break;
    kept.push(line);
    remaining -= cost;
  }

  const truncatedCount = bodyLines.length - kept.length;
  const notice = `\n\n[${truncatedCount} more lines truncated due to context window limit]`;

  return headerLines.join('\n') + '\n' + kept.join('\n') + notice;
}
