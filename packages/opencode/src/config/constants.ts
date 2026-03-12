export const DEFAULT_MAX_TOKENS = 50_000;
export const WEBFETCH_MAX_TOKENS = 10_000;
export const TRUNCATABLE_TOOLS = new Set([
  "grep",
  "glob",
  "bash",
  "interactive_bash",
  "lsp_diagnostics",
  "webfetch",
]);
