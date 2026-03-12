import { CHARS_PER_TOKEN } from './truncation-constants.js';

/**
 * Estimate token count for a string using a fixed chars-per-token ratio.
 * Fast approximation; not model-specific.
 */
export function estimateTokens(text: string): number {
  return Math.ceil(text.length / CHARS_PER_TOKEN);
}
