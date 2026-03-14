// Gitignore-spec compliant pattern matching for .slignore files
import * as fs from "node:fs";
import * as path from "node:path";
import { Ignore } from "./ignore-wrapper.ts";
import type { IgnoreInstance } from "./ignore-wrapper.ts";

// Default patterns if .slignore doesn't exist or is empty
// Only includes directories with HEAVY file counts (1000+ files typical)
export const DEFAULT_PATTERNS: string[] = [
  // JavaScript/TypeScript - package dependencies & build outputs
  "node_modules", "dist", "build", ".next", ".nuxt",
  // Python - virtualenvs & cache
  "__pycache__", ".venv", "venv",
  // Go/PHP - vendor dependencies
  "vendor",
  // Rust/Java - compiled outputs
  "target",
  // Version control
  ".git",
  // Test coverage (can be large with reports)
  "coverage",
];

export interface Matcher {
  ig: IgnoreInstance;
  patterns: string[];
  original: string[];
}

export interface MatchResult {
  blocked: boolean;
  pattern?: string;
}

/** Load patterns from .slignore file, falling back to DEFAULT_PATTERNS */
export function loadPatterns(ckignorePath: string): string[] {
  if (!ckignorePath || !fs.existsSync(ckignorePath)) {
    return DEFAULT_PATTERNS;
  }

  try {
    const content = fs.readFileSync(ckignorePath, "utf-8");
    const patterns = content
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line && !line.startsWith("#"));

    return patterns.length > 0 ? patterns : DEFAULT_PATTERNS;
  } catch {
    return DEFAULT_PATTERNS;
  }
}

/** Create a matcher from patterns, normalizing to match anywhere in path tree */
export function createMatcher(patterns: string[]): Matcher {
  const ig = Ignore();
  const normalizedPatterns: string[] = [];

  for (const p of patterns) {
    if (p.startsWith("!")) {
      const inner = p.slice(1);
      if (inner.includes("/") || inner.includes("*")) {
        normalizedPatterns.push(p);
      } else {
        normalizedPatterns.push(`!**/${inner}`);
        normalizedPatterns.push(`!**/${inner}/**`);
      }
    } else {
      if (p.includes("/") || p.includes("*")) {
        normalizedPatterns.push(p);
      } else {
        normalizedPatterns.push(`**/${p}`);
        normalizedPatterns.push(`**/${p}/**`);
        normalizedPatterns.push(p);
        normalizedPatterns.push(`${p}/**`);
      }
    }
  }

  ig.add(normalizedPatterns);
  return { ig, patterns: normalizedPatterns, original: patterns };
}

/** Check if a path should be blocked */
export function matchPath(matcher: Matcher, testPath: string): MatchResult {
  if (!testPath || typeof testPath !== "string") return { blocked: false };

  let normalized = testPath.replace(/\\/g, "/");
  if (normalized.startsWith("./")) normalized = normalized.slice(2);
  while (normalized.startsWith("/")) normalized = normalized.slice(1);
  while (normalized.startsWith("../")) normalized = normalized.slice(3);
  if (!normalized) return { blocked: false };

  const blocked = matcher.ig.ignores(normalized);
  if (blocked) {
    return { blocked: true, pattern: findMatchingPattern(matcher.original, normalized) };
  }
  return { blocked: false };
}

/** Find which original pattern matched (for error messages) */
export function findMatchingPattern(originalPatterns: string[], testPath: string): string {
  for (const p of originalPatterns) {
    if (p.startsWith("!")) continue;

    const simplified = p.replace(/\*\*/g, "").replace(/\*/g, "");
    if (simplified && testPath.includes(simplified)) return p;

    const tempIg = Ignore();
    if (p.includes("/") || p.includes("*")) {
      tempIg.add(p);
    } else {
      tempIg.add([`**/${p}`, `**/${p}/**`, p, `${p}/**`]);
    }
    if (tempIg.ignores(testPath)) return p;
  }
  return originalPatterns.find((p) => !p.startsWith("!")) || "unknown";
}
