// Hook I/O helpers: stdin reading, stdout writing, error blocking
import * as fs from "node:fs";
import type { HookInput, HookOutput } from "./types.ts";

/** Read JSON from stdin asynchronously (works in both Node and Bun) */
export async function readInput<T extends HookInput>(): Promise<T> {
  const chunks: Buffer[] = [];
  for await (const chunk of process.stdin) {
    chunks.push(chunk as Buffer);
  }
  return JSON.parse(Buffer.concat(chunks).toString()) as T;
}

/** Read stdin synchronously (for hooks that use sync I/O) */
export function readInputSync<T extends HookInput>(): T {
  const stdin = fs.readFileSync(0, "utf-8").trim();
  if (!stdin) throw new Error("Empty stdin");
  return JSON.parse(stdin) as T;
}

/** Write JSON output to stdout */
export function writeOutput(output: HookOutput): void {
  process.stdout.write(JSON.stringify(output));
}

/** Write plain text to stdout (for SessionStart/UserPromptSubmit context) */
export function writeContext(text: string): void {
  process.stdout.write(text);
}

/** Exit with blocking error (stderr message shown to Claude, exit 2) */
export function block(message: string): never {
  process.stderr.write(message);
  process.exit(2);
}

/** Log to stderr (visible in hook output but non-blocking) */
export function log(hookName: string, message: string): void {
  process.stderr.write(`[${hookName}] ${message}\n`);
}

/** Wrap async hook in try-catch with fail-open behavior */
export async function runHook<T extends HookInput>(
  hookName: string,
  handler: (input: T) => Promise<void>
): Promise<void> {
  try {
    const input = await readInput<T>();
    await handler(input);
  } catch (err) {
    process.stderr.write(
      `[${hookName}] ${err instanceof Error ? err.message : String(err)}\n`
    );
    process.exit(0); // fail-open
  }
}

/** Wrap sync hook in try-catch with fail-open behavior */
export function runHookSync<T extends HookInput>(
  hookName: string,
  handler: (input: T) => void
): void {
  try {
    const input = readInputSync<T>();
    handler(input);
  } catch (err) {
    process.stderr.write(
      `[${hookName}] ${err instanceof Error ? err.message : String(err)}\n`
    );
    process.exit(0);
  }
}
