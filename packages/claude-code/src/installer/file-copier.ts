import { copyFile, mkdir, readdir, stat } from "node:fs/promises";
import { join } from "node:path";

export async function copyDirectory(
  src: string,
  dest: string,
): Promise<string[]> {
  const copied: string[] = [];
  await mkdir(dest, { recursive: true });
  const entries = await readdir(src, { withFileTypes: true });
  for (const entry of entries) {
    const srcPath = join(src, entry.name);
    const destPath = join(dest, entry.name);
    if (entry.isDirectory()) {
      const subCopied = await copyDirectory(srcPath, destPath);
      copied.push(...subCopied);
    } else {
      await copyFile(srcPath, destPath);
      copied.push(destPath);
    }
  }
  return copied;
}

export async function copyFileIfExists(
  src: string,
  dest: string,
): Promise<boolean> {
  try {
    await stat(src);
    await copyFile(src, dest);
    return true;
  } catch {
    return false;
  }
}
