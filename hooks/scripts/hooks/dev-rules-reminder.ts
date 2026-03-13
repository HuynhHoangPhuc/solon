// UserPromptSubmit: Inject dev rules reminder context (throttled by transcript check)
import { isHookEnabled } from "../lib/config-loader.ts";
import { buildReminderContext, wasRecentlyInjected } from "../lib/context-builder.ts";
import { readInputSync, writeContext } from "../lib/hook-io.ts";
import type { UserPromptSubmitInput } from "../lib/types.ts";

if (!isHookEnabled("dev-rules-reminder")) process.exit(0);

try {
  const input = readInputSync<UserPromptSubmitInput>();
  if (wasRecentlyInjected(input.transcript_path)) process.exit(0);

  const sessionId = input.session_id || process.env.SL_SESSION_ID || null;
  const baseDir = process.cwd();
  const { content } = buildReminderContext({ sessionId, baseDir });
  writeContext(content);
} catch {
  process.exit(0);
}
