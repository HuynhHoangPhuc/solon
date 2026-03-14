// PreToolUse(Write): Inject file naming guidance as allow response
import { isHookEnabled } from "../lib/config-loader.ts";
import { writeOutput } from "../lib/hook-io.ts";
import type { PreToolUseOutput } from "../lib/types.ts";

if (!isHookEnabled("descriptive-name")) process.exit(0);

const guidance = `## File naming guidance:
- Skip this guidance if you are creating markdown or plain text files
- Prefer kebab-case for JS/TS/Python/shell (.js, .ts, .py, .sh) with descriptive names
- Respect language conventions: C#/Java/Kotlin/Swift use PascalCase (.cs, .java, .kt, .swift), Go/Rust use snake_case (.go, .rs)
- Other languages: follow their ecosystem's standard naming convention
- Goal: self-documenting names for LLM tools (Grep, Glob, Search)`;

writeOutput({
  hookSpecificOutput: {
    hookEventName: "PreToolUse",
    permissionDecision: "allow",
    additionalContext: guidance,
  } as PreToolUseOutput,
});
