// session-init helper functions: shadowed skills cleanup + agent team detection
import * as fs from "node:fs";
import * as path from "node:path";
import * as os from "node:os";

interface ShadowedCleanupResult {
  restored: string[];
  skipped: string[];
  kept: string[];
}

interface TeamInfo {
  teamName: string;
  memberCount: number;
}

/** One-time cleanup for orphaned .shadowed/ dirs from disabled skill-dedup hook (Issue #422) */
export function cleanupOrphanedShadowedSkills(): ShadowedCleanupResult {
  const shadowedDir = path.join(process.cwd(), ".claude", "skills", ".shadowed");
  if (!fs.existsSync(shadowedDir)) return { restored: [], skipped: [], kept: [] };

  const skillsDir = path.join(process.cwd(), ".claude", "skills");
  const restored: string[] = [];
  const skipped: string[] = [];
  const kept: string[] = [];

  try {
    const entries = fs.readdirSync(shadowedDir, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      const src = path.join(shadowedDir, entry.name);
      const dest = path.join(skillsDir, entry.name);
      try {
        if (!fs.existsSync(dest)) {
          fs.renameSync(src, dest);
          restored.push(entry.name);
        } else {
          const orphanedSkill = path.join(src, "SKILL.md");
          const localSkill = path.join(dest, "SKILL.md");
          if (fs.existsSync(orphanedSkill) && fs.existsSync(localSkill)) {
            if (fs.readFileSync(orphanedSkill, "utf8") === fs.readFileSync(localSkill, "utf8")) {
              fs.rmSync(src, { recursive: true, force: true });
              skipped.push(entry.name);
            } else {
              kept.push(entry.name); // Content differs — keep for manual review
            }
          } else {
            fs.rmSync(src, { recursive: true, force: true });
            skipped.push(entry.name);
          }
        }
      } catch (err) {
        process.stderr.write(`[session-init] Failed to process "${entry.name}": ${(err as Error).message}\n`);
      }
    }

    const manifestFile = path.join(shadowedDir, ".dedup-manifest.json");
    if (fs.existsSync(manifestFile)) fs.unlinkSync(manifestFile);
    const remaining = fs.readdirSync(shadowedDir);
    if (remaining.length === 0) fs.rmdirSync(shadowedDir);
  } catch (err) {
    process.stderr.write(`[session-init] Shadowed cleanup error: ${(err as Error).message}\n`);
  }

  return { restored, skipped, kept };
}

/** Detect if session is running inside an Agent Team by scanning ~/.claude/teams/ */
export function detectAgentTeam(): TeamInfo | null {
  try {
    const teamsDir = path.join(os.homedir(), ".claude", "teams");
    if (!fs.existsSync(teamsDir)) return null;

    for (const entry of fs.readdirSync(teamsDir, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const configPath = path.join(teamsDir, entry.name, "config.json");
      if (!fs.existsSync(configPath)) continue;
      try {
        const config = JSON.parse(fs.readFileSync(configPath, "utf8")) as { members?: unknown[] };
        if (config.members && config.members.length > 0) {
          return { teamName: entry.name, memberCount: config.members.length };
        }
      } catch {}
    }
    return null;
  } catch {
    return null;
  }
}
