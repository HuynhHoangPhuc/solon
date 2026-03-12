/**
 * Format a line with its hashline prefix for display in read output.
 * Output: "{lineNumber}#{hash}|{content}"
 */
export function formatHashLine(lineNumber: number, hash: string, content: string): string {
  return `${lineNumber}#${hash}|${content}`;
}
