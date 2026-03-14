// SubagentStop(Plan): Remind to run /cook after plan agent finishes
import * as path from "node:path";
import { isHookEnabled } from "../lib/config-loader.ts";
import { readSessionState } from "../lib/session-state.ts";
import { readInputSync, writeContext } from "../lib/hook-io.ts";
import type { SubagentStopInput } from "../lib/types.ts";

if (!isHookEnabled("cook-after-plan-reminder")) process.exit(0);

try {
  readInputSync<SubagentStopInput>(); // consume stdin (not used)
  const sessionId = process.env.SL_SESSION_ID;
  let planPath: string | null = null;

  if (sessionId) {
    const state = readSessionState(sessionId);
    if (state?.activePlan) {
      planPath = state.activePlan;
      if (!path.isAbsolute(planPath) && state.sessionOrigin) {
        planPath = path.resolve(state.sessionOrigin, planPath);
      }
    }
  }

  writeContext("MUST invoke /cook --auto skill before implementing the plan\n");
  if (planPath) {
    writeContext(`Best Practice: Run /clear then /cook ${path.join(planPath, "plan.md")}\n`);
  } else {
    writeContext("Best Practice: Run /clear then /cook {full-absolute-path-to-plan.md}\n");
  }
} catch {
  process.exit(0);
}
