/// SubagentStart hook: inject compact context for subagents.
/// Writes JSON hookSpecificOutput with additionalContext to stdout.
use anyhow::Result;
use serde_json::Value;

use crate::hooks::{
    extract_task_list_id, get_env, get_reports_path, get_str, is_hook_enabled, load_config,
    normalize_path, read_wisdom, resolve_naming_pattern, resolve_plan_path, resolve_skills_venv,
};

pub fn run() -> Result<()> {
    if !is_hook_enabled("subagent-init") {
        std::process::exit(0);
    }

    let input: Value = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(_) => std::process::exit(0),
    };

    let agent_type = {
        let s = get_str(&input, "agent_type");
        if s.is_empty() { "unknown" } else { s }.to_string()
    };
    let agent_id = {
        let s = get_str(&input, "agent_id");
        if s.is_empty() { "unknown" } else { s }.to_string()
    };

    // Exit early if no meaningful payload
    if agent_type == "unknown" && agent_id == "unknown" && get_str(&input, "session_id").is_empty()
    {
        std::process::exit(0);
    }

    let cfg = load_config();

    let effective_cwd = {
        let c = get_str(&input, "cwd").trim().to_string();
        if c.is_empty() {
            std::env::current_dir()
                .unwrap_or_default()
                .to_string_lossy()
                .to_string()
        } else {
            c
        }
    };

    let session_id = {
        let s = get_str(&input, "session_id").to_string();
        if s.is_empty() {
            get_env("SL_SESSION_ID").unwrap_or_default()
        } else {
            s
        }
    };

    let resolved = resolve_plan_path(&session_id, &cfg);
    let reports_path = get_reports_path(
        &resolved.path,
        &resolved.resolved_by,
        &cfg.plan,
        &cfg.paths,
        &effective_cwd,
    );
    let task_list_id = extract_task_list_id(&resolved);

    let plans_dir = normalize_path(&cfg.paths.plans);
    let plans_path = format!(
        "{}/{}",
        &effective_cwd,
        if plans_dir.is_empty() {
            "plans"
        } else {
            &plans_dir
        }
    );
    let docs_dir = normalize_path(&cfg.paths.docs);
    let docs_path = format!(
        "{}/{}",
        &effective_cwd,
        if docs_dir.is_empty() {
            "docs"
        } else {
            &docs_dir
        }
    );

    let name_pattern = resolve_naming_pattern(&cfg.plan, "");

    let active_plan = if resolved.resolved_by == "session" {
        resolved.path.clone()
    } else {
        String::new()
    };
    let suggested_plan = if resolved.resolved_by == "branch" {
        resolved.path.clone()
    } else {
        String::new()
    };

    let skills_venv = resolve_skills_venv();

    let thinking_lang = cfg
        .locale
        .thinking_language
        .as_deref()
        .unwrap_or("")
        .to_string();
    let response_lang = cfg
        .locale
        .response_language
        .as_deref()
        .unwrap_or("")
        .to_string();
    let effective_thinking = if thinking_lang.is_empty() && !response_lang.is_empty() {
        "en".to_string()
    } else {
        thinking_lang.clone()
    };

    let mut lines: Vec<String> = Vec::new();
    lines.push(format!("## Subagent: {}", agent_type));
    lines.push(format!("ID: {} | CWD: {}", agent_id, effective_cwd));
    lines.push(String::new());

    lines.push("## Context".to_string());
    if !active_plan.is_empty() {
        lines.push(format!("- Plan: {}", active_plan));
        if !task_list_id.is_empty() {
            lines.push(format!(
                "- Task List: {} (shared with session)",
                task_list_id
            ));
        }
    } else if !suggested_plan.is_empty() {
        lines.push(format!("- Plan: none | Suggested: {}", suggested_plan));
    } else {
        lines.push("- Plan: none".to_string());
    }
    lines.push(format!("- Reports: {}", reports_path));
    lines.push(format!("- Paths: {}/ | {}/", plans_path, docs_path));

    // Inject workflow progress from sc binary
    if !active_plan.is_empty() {
        if let Some(progress) = fetch_workflow_progress(&active_plan) {
            lines.push(format!("- {}", progress));
        }
    }
    lines.push(String::new());

    let has_thinking = !effective_thinking.is_empty() && effective_thinking != response_lang;
    if has_thinking || !response_lang.is_empty() {
        lines.push("## Language".to_string());
        if has_thinking {
            lines.push(format!(
                "- Thinking: Use {} for reasoning (logic, precision).",
                effective_thinking
            ));
        }
        if !response_lang.is_empty() {
            lines.push(format!(
                "- Response: Respond in {} (natural, fluent).",
                response_lang
            ));
        }
        lines.push(String::new());
    }

    lines.push("## Rules".to_string());
    lines.push(format!("- Reports → {}", reports_path));
    lines.push("- YAGNI / KISS / DRY".to_string());
    lines.push("- Concise, list unresolved Qs at end".to_string());
    if !skills_venv.is_empty() {
        lines.push(format!(
            "- Python scripts in .claude/skills/: Use `{}`",
            skills_venv
        ));
        lines.push("- Never use global pip install".to_string());
    }

    lines.push(String::new());
    lines.push("## Naming".to_string());
    lines.push(format!(
        "- Report: {}",
        format!("{}{}-{}.md", reports_path, agent_type, name_pattern)
    ));
    lines.push(format!("- Plan dir: {}/{}/", plans_path, name_pattern));

    if cfg.trust.enabled {
        if let Some(ref pass) = cfg.trust.passphrase {
            if !pass.is_empty() {
                lines.push(String::new());
                lines.push("## Trust Verification".to_string());
                lines.push(format!("Passphrase: \"{}\"", pass));
            }
        }
    }

    // Inject prior learnings
    let wisdom = read_wisdom(&active_plan, &session_id, 15);
    if !wisdom.is_empty() {
        lines.push(String::new());
        lines.push("## Prior Learnings".to_string());
        lines.push(wisdom);
    }

    // Per-agent context prefix
    if let Some(ref subagent_cfg) = cfg.subagent {
        if let Some(agent_cfg) = subagent_cfg.agents.get(&agent_type) {
            if !agent_cfg.context_prefix.is_empty() {
                lines.push(String::new());
                lines.push("## Agent Instructions".to_string());
                lines.push(agent_cfg.context_prefix.clone());
            }
        }
    }

    let output = serde_json::json!({
        "hookSpecificOutput": {
            "hookEventName": "SubagentStart",
            "additionalContext": lines.join("\n")
        }
    });
    println!("{}", serde_json::to_string(&output)?);
    Ok(())
}

fn fetch_workflow_progress(plan_dir: &str) -> Option<String> {
    let sc = crate::hooks::find_sc_binary()?;
    let output = std::process::Command::new(&sc)
        .args(["workflow", "status", plan_dir])
        .output()
        .ok()?;
    if !output.status.success() {
        return None;
    }
    let text = String::from_utf8_lossy(&output.stdout).trim().to_string();
    if text.is_empty() {
        return None;
    }
    let v: Value = serde_json::from_str(&text).ok()?;
    let total = v.get("phases")?.get("total")?.as_i64()?;
    if total == 0 {
        return None;
    }
    let completed = v.get("phases")?.get("completed")?.as_i64().unwrap_or(0);
    let progress = v.get("progress")?.as_i64().unwrap_or(0);
    Some(format!(
        "Plan progress: {}% ({}/{} phases complete)",
        progress, completed, total
    ))
}
