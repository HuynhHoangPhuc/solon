// Wrapper for vendored ignore library (gitignore-spec pattern matching)
// ignore v5.3.0 - https://github.com/kaelzhang/node-ignore - MIT License
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);

interface IgnoreInstance {
  add(patterns: string | string[]): IgnoreInstance;
  ignores(path: string): boolean;
  filter(paths: string[]): string[];
  createFilter(): (path: string) => boolean;
}

type IgnoreFactory = () => IgnoreInstance;

const Ignore = require("./bundled/ignore.cjs") as IgnoreFactory;

export { Ignore };
export type { IgnoreInstance };
