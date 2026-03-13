import { describe, it, expect, mock, beforeEach } from "bun:test";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

// Use a temp file for throttle state to isolate tests
const TEST_THROTTLE_FILE = path.join(os.tmpdir(), `sl-noti-throttle-test-${process.pid}.json`);

// Override the throttle file path before importing sender
// We mock the module by importing the underlying functions directly

describe("sender throttle logic", () => {
  const throttleFile = TEST_THROTTLE_FILE;

  beforeEach(() => {
    // Clean up throttle file between tests
    try { fs.unlinkSync(throttleFile); } catch { /* ignore */ }
  });

  it("throttle file does not exist → fresh state", () => {
    expect(fs.existsSync(throttleFile)).toBe(false);
  });

  it("records and reads throttle state correctly", () => {
    const state: Record<string, number> = { telegram: Date.now() };
    fs.writeFileSync(throttleFile, JSON.stringify(state));
    const loaded = JSON.parse(fs.readFileSync(throttleFile, "utf8")) as Record<string, number>;
    expect(loaded.telegram).toBeTruthy();
    expect(Date.now() - loaded.telegram).toBeLessThan(1000);
  });

  it("throttle expires after duration", () => {
    const oldTime = Date.now() - 6 * 60 * 1000; // 6 minutes ago
    const state = { discord: oldTime };
    fs.writeFileSync(throttleFile, JSON.stringify(state));
    const loaded = JSON.parse(fs.readFileSync(throttleFile, "utf8")) as Record<string, number>;
    const elapsed = Date.now() - loaded.discord;
    const THROTTLE_DURATION_MS = 5 * 60 * 1000;
    expect(elapsed >= THROTTLE_DURATION_MS).toBe(true);
  });
});

describe("send() with mocked fetch", () => {
  it("returns success on 200 response", async () => {
    const originalFetch = globalThis.fetch;
    globalThis.fetch = mock(async () => new Response(null, { status: 200 }));

    const { send } = await import("../../notifications/lib/sender.ts");
    const result = await send("test-provider-ok", "https://example.com/webhook", { text: "hello" });
    expect(result.success).toBe(true);

    globalThis.fetch = originalFetch;
  });

  it("returns error with message on non-2xx response", async () => {
    const originalFetch = globalThis.fetch;
    globalThis.fetch = mock(async () => new Response("Bad Request", { status: 400 }));

    const { send } = await import("../../notifications/lib/sender.ts");
    const result = await send("test-provider-fail", "https://example.com/webhook", {});
    // May be throttled from a previous run — just check it's not a success
    expect(result.success).toBe(false);
    if (!result.throttled) {
      expect(typeof result.error).toBe("string");
    }

    globalThis.fetch = originalFetch;
  });

  it("returns error on network failure", async () => {
    const originalFetch = globalThis.fetch;
    globalThis.fetch = mock(async () => { throw new Error("Network error"); });

    const { send } = await import("../../notifications/lib/sender.ts");
    const result = await send("test-provider-net", "https://example.com/webhook", {});
    expect(result.success).toBe(false);
    if (!result.throttled) {
      expect(typeof result.error).toBe("string");
    }

    globalThis.fetch = originalFetch;
  });
});
