// Hashline
export {
  computeLineHash,
  HASHLINE_DICT,
  NIBBLE_STR,
} from "./hashline/hash-computation.js";
export { formatHashLine } from "./hashline/hash-line-formatter.js";
export {
  parseLineRef,
  normalizeLineRef,
  HASHLINE_REF_PATTERN,
} from "./hashline/hash-line-parser.js";
export {
  validateLineRefs,
  HashlineMismatchError,
} from "./hashline/hash-line-validator.js";
export { resolveHashlineEdit } from "./hashline/hash-line-edit-resolver.js";

// Truncation
export { estimateTokens } from "./truncation/token-estimator.js";
export { truncateOutput } from "./truncation/output-truncator.js";
export {
  DEFAULT_MAX_TOKENS,
  WEBFETCH_MAX_TOKENS,
  CHARS_PER_TOKEN,
  DEFAULT_PRESERVE_HEADER_LINES,
} from "./truncation/truncation-constants.js";

// Scout-block
export { checkBroadPattern } from "./scout-block/broad-pattern-detector.js";
export type { BroadPatternResult } from "./scout-block/broad-pattern-detector.js";
export {
  isBlockedDirectory,
  loadSolonIgnore,
} from "./scout-block/directory-blocklist.js";
export { checkScoutBlock } from "./scout-block/scout-block-checker.js";
export type { ScoutBlockResult } from "./scout-block/scout-block-checker.js";
