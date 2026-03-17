use crate::hooks::{
    get_env, get_reports_path, get_str, is_hook_enabled, load_config, normalize_path,
    read_context_percent, resolve_naming_pattern, resolve_plan_path, resolve_skills_venv,
    write_context,
};
/// UserPromptSubmit hook: inject dev rules reminder (throttled by transcript check).
use anyhow::Result;

pub fn run() -> Result<()> {
    if !is_hook_enabled("dev-rules-reminder") {
        std::process::exit(0);
    }

    let input = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(e) => {
            eprintln!("[dev-rules] {}", e);
            std::process::exit(0);
        }
    };

    let transcript_path = get_str(&input, "transcript_path").to_string();
    if was_recently_injected(&transcript_path) {
        std::process::exit(0);
    }

    let session_id = {
        let s = get_str(&input, "session_id").to_string();
        if s.is_empty() {
            get_env("SL_SESSION_ID").unwrap_or_default()
        } else {
            s
        }
    };

    let base_dir = std::env::current_dir()
        .unwrap_or_default()
        .to_string_lossy()
        .to_string();
    let content = build_reminder_context(&session_id, &base_dir);
    write_context(&content);
    Ok(())
}

fn was_recently_injected(transcript_path: &str) -> bool {
    if transcript_path.is_empty() {
        return false;
    }
    let data = match std::fs::read_to_string(transcript_path) {
        Ok(d) => d,
        Err(_) => return false,
    };
    let lines: Vec<&str> = data.lines().collect();
    let start = if lines.len() > 150 {
        lines.len() - 150
    } else {
        0
    };
    lines[start..]
        .iter()
        .any(|l| l.contains("[IMPORTANT] Consider Modularization"))
}

fn build_reminder_context(session_id: &str, base_dir: &str) -> String {
    let cfg = load_config();
    let resolved = resolve_plan_path(session_id, &cfg);
    let reports_path = get_reports_path(
        &resolved.path,
        &resolved.resolved_by,
        &cfg.plan,
        &cfg.paths,
        base_dir,
    );
    let name_pattern = resolve_naming_pattern(&cfg.plan, &crate::hooks::get_git_branch());
    let skills_venv = resolve_skills_venv();

    let plans_rel = normalize_path(&cfg.paths.plans);
    let plans_rel = if plans_rel.is_empty() {
        "plans".to_string()
    } else {
        plans_rel
    };
    let docs_rel = normalize_path(&cfg.paths.docs);
    let docs_rel = if docs_rel.is_empty() {
        "docs".to_string()
    } else {
        docs_rel
    };

    let plans_path = format!("{}/{}", base_dir, plans_rel);
    let docs_path = format!("{}/{}", base_dir, docs_rel);
    let docs_max_loc = if cfg.docs.max_loc > 0 {
        cfg.docs.max_loc
    } else {
        800
    };

    let mut lines: Vec<String> = Vec::new();

    // Language section
    let thinking = cfg.locale.thinking_language.as_deref().unwrap_or("");
    let response = cfg.locale.response_language.as_deref().unwrap_or("");
    let eff_thinking = if thinking.is_empty() && !response.is_empty() {
        "en"
    } else {
        thinking
    };
    let has_thinking = !eff_thinking.is_empty() && eff_thinking != response;
    if has_thinking || !response.is_empty() {
        lines.push("## Language".to_string());
        if has_thinking {
            lines.push(format!(
                "- Thinking: Use {} for reasoning (logic, precision).",
                eff_thinking
            ));
        }
        if !response.is_empty() {
            lines.push(format!(
                "- Response: Respond in {} (natural, fluent).",
                response
            ));
        }
        lines.push(String::new());
    }

    // Session section
    let tz = crate::hooks::local_tz();
    let platform = std::env::consts::OS;
    let user = get_env("USER")
        .or_else(|| get_env("USERNAME"))
        .or_else(|| get_env("LOGNAME"))
        .unwrap_or_default();
    let locale = get_env("LANG").unwrap_or_default();
    let now = chrono::Local::now();
    lines.push("## Session".to_string());
    lines.push(format!(
        "- DateTime: {}",
        now.format("%-m/%-d/%Y, %-I:%M:%S %p")
    ));
    lines.push(format!("- CWD: {}", base_dir));
    lines.push(format!("- Timezone: {}", tz));
    lines.push(format!("- Working directory: {}", base_dir));
    lines.push(format!("- OS: {}", platform));
    lines.push(format!("- User: {}", user));
    lines.push(format!("- Locale: {}", locale));
    lines.push("- Memory usage: N/A (Rust runtime)".to_string());
    lines.push("- Spawning multiple subagents can cause performance issues, spawn and delegate tasks intelligently based on the available system resources.".to_string());
    lines.push("- Remember that each subagent only has 200K tokens in context window, spawn and delegate tasks intelligently to make sure their context windows don't get bloated.".to_string());
    lines.push("- IMPORTANT: Include these environment information when prompting subagents to perform tasks.".to_string());
    lines.push(String::new());

    // Context usage section
    if cfg.hooks.get("context-tracking").copied().unwrap_or(true) {
        if let Some(pct) = read_context_percent(session_id) {
            lines.push("## Current Session's Context".to_string());
            lines.push(format!("- Context: {}% used", pct));
            lines.push("- **NOTE:** Optimize the workflow for token efficiency".to_string());
            if pct >= 90 {
                lines.push("- **CRITICAL:** Context nearly full - consider compaction or being concise, update current phase's status before the compaction.".to_string());
            } else if pct >= 70 {
                lines.push("- **WARNING:** Context usage moderate - being concise and optimize token efficiency.".to_string());
            }
            lines.push(String::new());
        }
    }

    // Rules section
    let dev_rules_path = resolve_rules_path("development-rules.md");
    lines.push("## Rules".to_string());
    if !dev_rules_path.is_empty() {
        lines.push(format!(
            "- Read and follow development rules: \"{}\"",
            dev_rules_path
        ));
    }
    lines.push(format!(
        "- Markdown files are organized in: Plans → \"{}\" directory, Docs → \"{}\" directory",
        plans_path, docs_path
    ));
    lines.push(format!("- **IMPORTANT:** DO NOT create markdown files outside of \"{}\" or \"{}\" UNLESS the user explicitly requests it.", plans_path, docs_path));
    if !skills_venv.is_empty() {
        lines.push(format!(
            "- Python scripts in .claude/skills/: Use `{}`",
            skills_venv
        ));
    }
    lines.push("- When skills' scripts are failed to execute, always fix them and run again, repeat until success.".to_string());
    lines.push("- Follow **YAGNI (You Aren't Gonna Need It) - KISS (Keep It Simple, Stupid) - DRY (Don't Repeat Yourself)** principles".to_string());
    lines.push("- Sacrifice grammar for the sake of concision when writing reports.".to_string());
    lines.push("- In reports, list any unresolved questions at the end, if any.".to_string());
    lines.push(
        "- IMPORTANT: Ensure token consumption efficiency while maintaining high quality."
            .to_string(),
    );
    lines.push(String::new());

    // Modularization section
    lines.push("## **[IMPORTANT] Consider Modularization:**".to_string());
    lines.push("- Check existing modules before creating new".to_string());
    lines
        .push("- Analyze logical separation boundaries (functions, classes, concerns)".to_string());
    lines.push("- Prefer kebab-case for JS/TS/Python/shell; respect language conventions (C#/Java use PascalCase, Go/Rust use snake_case)".to_string());
    lines.push("- Write descriptive code comments".to_string());
    lines.push("- After modularization, continue with main task".to_string());
    lines.push("- When not to modularize: Markdown files, plain text files, bash scripts, configuration files, environment variables files, etc.".to_string());
    lines.push(String::new());

    // Paths section
    lines.push("## Paths".to_string());
    lines.push(format!(
        "Reports: {} | Plans: {}/ | Docs: {}/ | docs.maxLoc: {}",
        reports_path, plans_path, docs_path, docs_max_loc
    ));
    lines.push(String::new());

    // Plan context section
    let git_branch = crate::hooks::get_git_branch();
    let plan_line = match resolved.resolved_by.as_str() {
        "session" => format!("- Plan: {}", resolved.path),
        "branch" => format!("- Plan: none | Suggested: {}", resolved.path),
        _ => "- Plan: none".to_string(),
    };
    let v = &cfg.plan.validation;
    let vmode = if v.mode.is_empty() { "prompt" } else { &v.mode };
    let vmin = if v.min_questions == 0 {
        3
    } else {
        v.min_questions
    };
    let vmax = if v.max_questions == 0 {
        8
    } else {
        v.max_questions
    };
    lines.push("## Plan Context".to_string());
    lines.push(plan_line);
    lines.push(format!("- Reports: {}", reports_path));
    if !git_branch.is_empty() {
        lines.push(format!("- Branch: {}", git_branch));
    }
    lines.push(format!(
        "- Validation: mode={}, questions={}-{}",
        vmode, vmin, vmax
    ));
    lines.push(String::new());

    // Naming section
    lines.push("## Naming".to_string());
    lines.push(format!(
        "- Report: `{}{{}}-{}.md`",
        reports_path, name_pattern
    ));
    lines.push(format!("- Plan dir: `{}/{}/`", plans_path, name_pattern));
    lines.push("- Replace `{type}` with: agent name, report type, or context".to_string());
    lines.push("- Replace `{slug}` in pattern with: descriptive-kebab-slug".to_string());

    // Semantic compression (stub — just return as-is; full impl would compress)
    lines.join("\n")
}

fn resolve_rules_path(filename: &str) -> String {
    let cwd = std::env::current_dir().unwrap_or_default();
    let home = dirs::home_dir().unwrap_or_default();
    let local = cwd.join(".claude").join("rules").join(filename);
    let global = home.join(".claude").join("rules").join(filename);
    if local.exists() {
        return format!(".claude/rules/{}", filename);
    }
    if global.exists() {
        return format!("~/.claude/rules/{}", filename);
    }
    // Legacy location
    let local_leg = cwd.join(".claude").join("workflows").join(filename);
    let global_leg = home.join(".claude").join("workflows").join(filename);
    if local_leg.exists() {
        return format!(".claude/workflows/{}", filename);
    }
    if global_leg.exists() {
        return format!("~/.claude/workflows/{}", filename);
    }
    String::new()
}
