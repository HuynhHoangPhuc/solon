import { describe, it, expect } from "bun:test";
import {
  isBuildCommand, isVenvExecutable, isVenvCreationCommand, isAllowedCommand,
  splitCompoundCommand, unwrapShellExecutor, checkScoutBlock,
} from "../../scout-block/scout-checker.ts";

describe("isBuildCommand", () => {
  it("allows npm build", () => expect(isBuildCommand("npm run build")).toBe(true));
  it("allows pnpm test", () => expect(isBuildCommand("pnpm test")).toBe(true));
  it("allows yarn lint", () => expect(isBuildCommand("yarn lint")).toBe(true));
  it("allows bun install", () => expect(isBuildCommand("bun install")).toBe(true));
  it("allows cargo build", () => expect(isBuildCommand("cargo build")).toBe(true));
  it("allows go build", () => expect(isBuildCommand("go build ./...")).toBe(true));
  it("allows make", () => expect(isBuildCommand("make all")).toBe(true));
  it("allows tsc", () => expect(isBuildCommand("tsc --noEmit")).toBe(true));
  it("allows docker", () => expect(isBuildCommand("docker build .")).toBe(true));
  it("rejects ls node_modules", () => expect(isBuildCommand("ls node_modules")).toBe(false));
  it("rejects cat dist/file.js", () => expect(isBuildCommand("cat dist/file.js")).toBe(false));
  it("allows pnpm --filter web run build", () => {
    expect(isBuildCommand("pnpm --filter web run build")).toBe(true);
  });
});

describe("isVenvExecutable", () => {
  it("allows .venv/bin/ commands", () => {
    expect(isVenvExecutable(".venv/bin/python3 script.py")).toBe(true);
    expect(isVenvExecutable(".venv/bin/pytest")).toBe(true);
  });
  it("allows venv/Scripts/ on Windows", () => {
    expect(isVenvExecutable(".venv\\Scripts\\python.exe")).toBe(true);
  });
  it("rejects .venv directory access", () => {
    expect(isVenvExecutable("ls .venv")).toBe(false);
  });
});

describe("isVenvCreationCommand", () => {
  it("allows python3 -m venv", () => expect(isVenvCreationCommand("python3 -m venv .venv")).toBe(true));
  it("allows python -m venv", () => expect(isVenvCreationCommand("python -m venv env")).toBe(true));
  it("allows uv venv", () => expect(isVenvCreationCommand("uv venv")).toBe(true));
  it("allows virtualenv", () => expect(isVenvCreationCommand("virtualenv .venv")).toBe(true));
  it("rejects pip install", () => expect(isVenvCreationCommand("pip install requests")).toBe(false));
});

describe("isAllowedCommand", () => {
  it("strips env prefix before checking", () => {
    expect(isAllowedCommand("NODE_ENV=production npm run build")).toBe(true);
  });
  it("strips sudo before checking", () => {
    expect(isAllowedCommand("sudo npm install")).toBe(true);
  });
  it("allows venv executable", () => {
    expect(isAllowedCommand(".venv/bin/pytest tests/")).toBe(true);
  });
  it("rejects rm -rf dist", () => {
    expect(isAllowedCommand("rm -rf dist")).toBe(false);
  });
});

describe("splitCompoundCommand", () => {
  it("splits on &&", () => {
    const result = splitCompoundCommand("npm install && npm run build");
    expect(result).toHaveLength(2);
    expect(result[0].trim()).toBe("npm install");
    expect(result[1].trim()).toBe("npm run build");
  });

  it("splits on ;", () => {
    const result = splitCompoundCommand("echo a; echo b; echo c");
    expect(result).toHaveLength(3);
  });

  it("splits on ||", () => {
    const result = splitCompoundCommand("npm test || echo failed");
    expect(result).toHaveLength(2);
  });

  it("returns single command unchanged", () => {
    const result = splitCompoundCommand("ls -la");
    expect(result).toHaveLength(1);
  });

  it("filters empty segments", () => {
    const result = splitCompoundCommand("  &&  ");
    expect(result).toHaveLength(0);
  });
});

describe("unwrapShellExecutor", () => {
  it("unwraps bash -c", () => {
    expect(unwrapShellExecutor('bash -c "ls dist"')).toBe("ls dist");
  });
  it("unwraps sh -c", () => {
    expect(unwrapShellExecutor("sh -c 'cat node_modules/x'")).toBe("cat node_modules/x");
  });
  it("returns original if not a shell executor", () => {
    expect(unwrapShellExecutor("npm run build")).toBe("npm run build");
  });
});

describe("checkScoutBlock", () => {
  it("allows src/ paths", () => {
    const result = checkScoutBlock({
      toolName: "Read",
      toolInput: { file_path: "src/main.ts" },
    });
    expect(result.blocked).toBe(false);
  });

  it("blocks node_modules path", () => {
    const result = checkScoutBlock({
      toolName: "Read",
      toolInput: { file_path: "node_modules/lodash/index.js" },
    });
    expect(result.blocked).toBe(true);
    expect(result.isBroadPattern).toBeFalsy();
  });

  it("marks allowed build command", () => {
    const result = checkScoutBlock({
      toolName: "Bash",
      toolInput: { command: "npm run build" },
    });
    expect(result.blocked).toBe(false);
    expect(result.isAllowedCommand).toBe(true);
  });

  it("blocks broad Glob pattern", () => {
    const result = checkScoutBlock({
      toolName: "Glob",
      toolInput: { pattern: "**/*.ts" },
    });
    expect(result.blocked).toBe(true);
    expect(result.isBroadPattern).toBe(true);
    expect(result.suggestions).toBeTruthy();
  });

  it("allows specific Glob pattern with src/ prefix", () => {
    const result = checkScoutBlock({
      toolName: "Glob",
      toolInput: { pattern: "src/**/*.ts" },
    });
    expect(result.blocked).toBe(false);
  });

  it("allows compound command where all sub-commands are build commands", () => {
    const result = checkScoutBlock({
      toolName: "Bash",
      toolInput: { command: "npm install && npm run build" },
    });
    expect(result.blocked).toBe(false);
    expect(result.isAllowedCommand).toBe(true);
  });
});
