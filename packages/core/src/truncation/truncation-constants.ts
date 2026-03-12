/** Default maximum tokens for general tool output */
export const DEFAULT_MAX_TOKENS = 50_000;

/** Stricter token cap for WebFetch output */
export const WEBFETCH_MAX_TOKENS = 10_000;

/** Average chars per token (rough estimate for English/code) */
export const CHARS_PER_TOKEN = 4;

/** Number of header lines to always preserve when truncating */
export const DEFAULT_PRESERVE_HEADER_LINES = 3;
