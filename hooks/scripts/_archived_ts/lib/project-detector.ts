// Project and environment detection: type, package manager, framework, Python, git
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";
import { execSafe, execFileSafe } from "./exec-utils.ts";
import type { ProjectInfo } from "./types.ts";

// ── Python Detection ───────────────────────────────────────────────────────

function isValidPythonPath(p: string): boolean {
  if (!p || typeof p !== "string") return false;
  if (/[;&|`$(){}[\]<>!#*?]/.test(p)) return false;
  try {
    return fs.statSync(p).isFile();
  } catch {
    return false;
  }
}

function getPythonPaths(): string[] {
  const paths: string[] = [];
  if (process.env.PYTHON_PATH) paths.push(process.env.PYTHON_PATH);

  if (process.platform === "win32") {
    const localAppData = process.env.LOCALAPPDATA;
    const programFiles = process.env.ProgramFiles || "C:\\Program Files";
    const programFilesX86 = process.env["ProgramFiles(x86)"] || "C:\\Program Files (x86)";
    if (localAppData) {
      paths.push(path.join(localAppData, "Microsoft", "WindowsApps", "python.exe"));
      for (const ver of ["313", "312", "311", "310", "39"]) {
        paths.push(path.join(localAppData, "Programs", "Python", `Python${ver}`, "python.exe"));
      }
    }
    for (const ver of ["313", "312", "311", "310", "39"]) {
      paths.push(path.join(programFiles, `Python${ver}`, "python.exe"));
      paths.push(path.join(programFilesX86, `Python${ver}`, "python.exe"));
    }
  } else {
    paths.push("/usr/bin/python3", "/usr/local/bin/python3");
    paths.push("/opt/homebrew/bin/python3", "/opt/homebrew/bin/python");
    paths.push("/usr/bin/python", "/usr/local/bin/python");
  }

  return paths;
}

function findPythonBinary(): string | null {
  if (process.platform !== "win32") {
    const py3 = execSafe("which python3", { timeout: 500 });
    if (py3 && isValidPythonPath(py3)) return py3;
    const py = execSafe("which python", { timeout: 500 });
    if (py && isValidPythonPath(py)) return py;
  } else {
    const where = execSafe("where python", { timeout: 500 });
    if (where) {
      const first = where.split("\n")[0].trim();
      if (isValidPythonPath(first)) return first;
    }
  }
  for (const p of getPythonPaths()) {
    if (isValidPythonPath(p)) return p;
  }
  return null;
}

export function getPythonVersion(): string | null {
  const pythonPath = findPythonBinary();
  if (pythonPath) {
    const result = execFileSafe(pythonPath, ["--version"]);
    if (result) return result;
  }
  for (const cmd of ["python3", "python"]) {
    const result = execFileSafe(cmd, ["--version"]);
    if (result) return result;
  }
  return null;
}

// ── Git Detection ──────────────────────────────────────────────────────────

export function isGitRepo(startDir?: string): boolean {
  let dir: string;
  try {
    dir = startDir || process.cwd();
  } catch {
    return false;
  }
  const root = path.parse(dir).root;
  while (dir !== root) {
    if (fs.existsSync(path.join(dir, ".git"))) return true;
    dir = path.dirname(dir);
  }
  return fs.existsSync(path.join(root, ".git"));
}

export function getGitRemoteUrl(): string | null {
  if (!isGitRepo()) return null;
  return execSafe("git config --get remote.origin.url");
}

export function getGitBranch(): string | null {
  if (!isGitRepo()) return null;
  return execSafe("git branch --show-current");
}

export function getGitRoot(): string | null {
  if (!isGitRepo()) return null;
  return execSafe("git rev-parse --show-toplevel");
}

// ── Project Type Detection ─────────────────────────────────────────────────

export function detectProjectType(configOverride?: string): string {
  if (configOverride && configOverride !== "auto") return configOverride;
  if (fs.existsSync("pnpm-workspace.yaml") || fs.existsSync("lerna.json")) return "monorepo";
  if (fs.existsSync("package.json")) {
    try {
      const pkg = JSON.parse(fs.readFileSync("package.json", "utf8")) as Record<string, unknown>;
      if (pkg.workspaces) return "monorepo";
      if (pkg.main || pkg.exports) return "library";
    } catch {}
  }
  return "single-repo";
}

export function detectPackageManager(configOverride?: string): string | null {
  if (configOverride && configOverride !== "auto") return configOverride;
  if (fs.existsSync("bun.lockb")) return "bun";
  if (fs.existsSync("pnpm-lock.yaml")) return "pnpm";
  if (fs.existsSync("yarn.lock")) return "yarn";
  if (fs.existsSync("package-lock.json")) return "npm";
  return null;
}

export function detectFramework(configOverride?: string): string | null {
  if (configOverride && configOverride !== "auto") return configOverride;
  if (!fs.existsSync("package.json")) return null;
  try {
    const pkg = JSON.parse(fs.readFileSync("package.json", "utf8")) as Record<string, unknown>;
    const deps = { ...(pkg.dependencies as object), ...(pkg.devDependencies as object) } as Record<string, unknown>;
    if (deps["next"]) return "next";
    if (deps["nuxt"]) return "nuxt";
    if (deps["astro"]) return "astro";
    if (deps["@remix-run/node"] || deps["@remix-run/react"]) return "remix";
    if (deps["svelte"] || deps["@sveltejs/kit"]) return "svelte";
    if (deps["vue"]) return "vue";
    if (deps["react"]) return "react";
    if (deps["express"]) return "express";
    if (deps["fastify"]) return "fastify";
    if (deps["hono"]) return "hono";
  } catch {}
  return null;
}

// ── Coding Level ───────────────────────────────────────────────────────────

export function getCodingLevelStyleName(level: number): string {
  const map: Record<number, string> = {
    0: "coding-level-0-eli5",
    1: "coding-level-1-junior",
    2: "coding-level-2-mid",
    3: "coding-level-3-senior",
    4: "coding-level-4-lead",
    5: "coding-level-5-god",
  };
  return map[level] || "coding-level-5-god";
}

export function getCodingLevelGuidelines(level: number, configDir?: string): string | null {
  if (level === -1 || level === null || level === undefined) return null;
  const styleName = getCodingLevelStyleName(level);
  const basePath = configDir || path.join(process.cwd(), ".claude");
  const stylePath = path.join(basePath, "output-styles", `${styleName}.md`);
  try {
    if (!fs.existsSync(stylePath)) return null;
    const content = fs.readFileSync(stylePath, "utf8");
    return content.replace(/^---[\s\S]*?---\n*/, "").trim();
  } catch {
    return null;
  }
}

// ── Main Entry Points ──────────────────────────────────────────────────────

interface DetectOptions {
  configOverrides?: { type?: string; packageManager?: string; framework?: string };
}

export function detectProject(options: DetectOptions = {}): ProjectInfo {
  const { configOverrides = {} } = options;
  return {
    type: detectProjectType(configOverrides.type),
    packageManager: detectPackageManager(configOverrides.packageManager),
    framework: detectFramework(configOverrides.framework),
    pythonVersion: getPythonVersion(),
    nodeVersion: process.version,
    gitBranch: getGitBranch(),
    gitRoot: getGitRoot(),
    gitUrl: getGitRemoteUrl(),
    osPlatform: process.platform,
    user: process.env.USERNAME || process.env.USER || process.env.LOGNAME || os.userInfo().username,
    locale: process.env.LANG || "",
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  };
}

export function buildStaticEnv(configDir?: string): Record<string, string | null> {
  return {
    nodeVersion: process.version,
    pythonVersion: getPythonVersion(),
    osPlatform: process.platform,
    gitUrl: getGitRemoteUrl(),
    gitBranch: getGitBranch(),
    gitRoot: getGitRoot(),
    user: process.env.USERNAME || process.env.USER || process.env.LOGNAME || os.userInfo().username,
    locale: process.env.LANG || "",
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    configDir: configDir || path.join(process.cwd(), ".claude"),
  };
}
