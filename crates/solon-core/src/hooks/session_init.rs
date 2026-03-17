/// SessionStart hook: detect project, write env vars, inject context.
/// Reads SessionStartInput from stdin, writes plain text context to stdout.
use anyhow::Result;
use serde_json::Value;
use std::path::PathBuf;

use crate::hooks::{
    build_compaction_context, detect_package_manager, detect_project_type, extract_task_list_id,
    get_coding_level_guidelines, get_coding_level_style_name, get_env, get_git_branch,
    get_git_remote_url, get_git_root, get_reports_path, get_str, is_hook_enabled, load_config,
    local_tz, normalize_path, resolve_naming_pattern, resolve_plan_path, write_context, write_env,
    write_env_forced, write_session_state, PlanResolution, SessionState,
};

pub fn run() -> Result<()> {
    if !is_hook_enabled("session-init") {
        std::process::exit(0);
    }

    let shadowed_cleanup = cleanup_orphaned_shadowed_skills();

    let input: Value = match crate::hooks::read_hook_input() {
        Ok(v) => v,
        Err(e) => {
            eprintln!("[session-init] Error: {}", e);
            std::process::exit(0);
        }
    };

    let env_file = get_env("CLAUDE_ENV_FILE").unwrap_or_default();
    let source = {
        let s = get_str(&input, "source");
        if s.is_empty() { "unknown" } else { s }.to_string()
    };
    let session_id = get_str(&input, "session_id").to_string();

    let cfg = load_config();

    let project_type = detect_project_type(&cfg.project.r#type);
    let package_manager = detect_package_manager(&cfg.project.package_manager);

    // Try sc binary first, fall back to internal resolver
    let resolved =
        try_resolve_with_sc(&session_id).unwrap_or_else(|| resolve_plan_path(&session_id, &cfg));

    // Persist session state
    if !session_id.is_empty() {
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
        let cwd = std::env::current_dir()
            .unwrap_or_default()
            .to_string_lossy()
            .to_string();
        write_session_state(
            &session_id,
            &SessionState {
                session_origin: cwd,
                active_plan: if active_plan.is_empty() {
                    None
                } else {
                    Some(active_plan)
                },
                suggested_plan: if suggested_plan.is_empty() {
                    None
                } else {
                    Some(suggested_plan)
                },
                timestamp: chrono::Utc::now().timestamp_millis(),
                source: source.clone(),
            },
        );
    }

    let base_dir = std::env::current_dir()
        .unwrap_or_default()
        .to_string_lossy()
        .to_string();
    let reports_path = get_reports_path(
        &resolved.path,
        &resolved.resolved_by,
        &cfg.plan,
        &cfg.paths,
        &base_dir,
    );
    let task_list_id = extract_task_list_id(&resolved);
    let git_branch = get_git_branch();
    let name_pattern = resolve_naming_pattern(&cfg.plan, &git_branch);

    let home = dirs_home_str();
    let user = first_of(&[
        &get_env("USERNAME").unwrap_or_default(),
        &get_env("USER").unwrap_or_default(),
        &get_env("LOGNAME").unwrap_or_default(),
    ]);
    let node_version = detect_node_version();
    let git_root = get_git_root();
    let git_url = get_git_remote_url();
    let timezone = local_tz();
    let python_version = get_python_version();

    write_session_env_vars(
        &env_file,
        &WriteEnvArgs {
            cfg: &cfg,
            session_id: &session_id,
            name_pattern: &name_pattern,
            resolved: &resolved,
            task_list_id: &task_list_id,
            reports_path: &reports_path,
            base_dir: &base_dir,
            project_type: &project_type,
            package_manager: &package_manager,
            node_version: &node_version,
            python_version: &python_version,
            git_root: &git_root,
            git_url: &git_url,
            git_branch: &git_branch,
            user: &user,
            timezone: &timezone,
            home: &home,
        },
    );

    let team = detect_agent_team();
    if !env_file.is_empty() {
        if let Some(ref t) = team {
            write_env(&env_file, "SL_AGENT_TEAM", &t.team_name);
            write_env(
                &env_file,
                "SL_AGENT_TEAM_MEMBERS",
                &t.member_count.to_string(),
            );
        }
    }

    write_session_context(WriteContextArgs {
        source: &source,
        project_type: &project_type,
        package_manager: &package_manager,
        cfg: &cfg,
        git_root: &git_root,
        base_dir: &base_dir,
        resolved: &resolved,
        shadowed_cleanup: &shadowed_cleanup,
        team: team.as_ref(),
        coding_level: cfg.coding_level,
        session_id: &session_id,
    });

    Ok(())
}

struct WriteEnvArgs<'a> {
    cfg: &'a crate::hooks::SlConfig,
    session_id: &'a str,
    name_pattern: &'a str,
    resolved: &'a PlanResolution,
    task_list_id: &'a str,
    reports_path: &'a str,
    base_dir: &'a str,
    project_type: &'a str,
    package_manager: &'a str,
    node_version: &'a str,
    python_version: &'a str,
    git_root: &'a str,
    git_url: &'a str,
    git_branch: &'a str,
    user: &'a str,
    timezone: &'a str,
    home: &'a str,
}

fn write_session_env_vars(env_file: &str, a: &WriteEnvArgs) {
    if env_file.is_empty() {
        return;
    }
    let cfg = a.cfg;

    write_env(env_file, "SL_SESSION_ID", a.session_id);
    write_env(env_file, "SL_PLAN_NAMING_FORMAT", &cfg.plan.naming_format);
    write_env(env_file, "SL_PLAN_DATE_FORMAT", &cfg.plan.date_format);
    let issue_prefix = cfg.plan.issue_prefix.as_deref().unwrap_or("");
    write_env_forced(env_file, "SL_PLAN_ISSUE_PREFIX", issue_prefix);
    write_env(env_file, "SL_PLAN_REPORTS_DIR", &cfg.plan.reports_dir);
    write_env(env_file, "SL_NAME_PATTERN", a.name_pattern);

    let active_plan = if a.resolved.resolved_by == "session" {
        &a.resolved.path
    } else {
        ""
    };
    write_env_forced(env_file, "SL_ACTIVE_PLAN", active_plan);
    let suggested_plan = if a.resolved.resolved_by == "branch" {
        &a.resolved.path
    } else {
        ""
    };
    write_env_forced(env_file, "SL_SUGGESTED_PLAN", suggested_plan);

    if !a.task_list_id.is_empty() {
        write_env(env_file, "CLAUDE_CODE_TASK_LIST_ID", a.task_list_id);
    }
    write_env(env_file, "SL_GIT_ROOT", a.git_root);
    write_env(env_file, "SL_REPORTS_PATH", a.reports_path);

    let docs_path = PathBuf::from(a.base_dir).join(normalize_path(&cfg.paths.docs));
    let plans_path = PathBuf::from(a.base_dir).join(normalize_path(&cfg.paths.plans));
    write_env(env_file, "SL_DOCS_PATH", &docs_path.to_string_lossy());
    write_env(env_file, "SL_PLANS_PATH", &plans_path.to_string_lossy());
    write_env(env_file, "SL_PROJECT_ROOT", a.base_dir);
    write_env(env_file, "SL_PROJECT_TYPE", a.project_type);
    write_env_forced(env_file, "SL_PACKAGE_MANAGER", a.package_manager);
    write_env(env_file, "SL_NODE_VERSION", a.node_version);
    write_env(env_file, "SL_PYTHON_VERSION", a.python_version);
    write_env(env_file, "SL_OS_PLATFORM", std::env::consts::OS);
    write_env(env_file, "SL_GIT_URL", a.git_url);
    write_env(env_file, "SL_GIT_BRANCH", a.git_branch);
    write_env(env_file, "SL_USER", a.user);
    write_env(env_file, "SL_LOCALE", &get_env("LANG").unwrap_or_default());
    write_env(env_file, "SL_TIMEZONE", a.timezone);
    write_env(
        env_file,
        "SL_CLAUDE_SETTINGS_DIR",
        &format!("{}/.claude", a.home),
    );

    if let Some(ref tl) = cfg.locale.thinking_language {
        if !tl.is_empty() {
            write_env(env_file, "SL_THINKING_LANGUAGE", tl);
        }
    }
    if let Some(ref rl) = cfg.locale.response_language {
        if !rl.is_empty() {
            write_env(env_file, "SL_RESPONSE_LANGUAGE", rl);
        }
    }

    let v = &cfg.plan.validation;
    let mode = if v.mode.is_empty() { "prompt" } else { &v.mode };
    let min_q = if v.min_questions == 0 {
        3
    } else {
        v.min_questions
    };
    let max_q = if v.max_questions == 0 {
        8
    } else {
        v.max_questions
    };
    write_env(env_file, "SL_VALIDATION_MODE", mode);
    write_env(env_file, "SL_VALIDATION_MIN_QUESTIONS", &min_q.to_string());
    write_env(env_file, "SL_VALIDATION_MAX_QUESTIONS", &max_q.to_string());
    write_env(
        env_file,
        "SL_VALIDATION_FOCUS_AREAS",
        &v.focus_areas.join(","),
    );

    let coding_level = if cfg.coding_level < 0 {
        5
    } else {
        cfg.coding_level
    };
    write_env(env_file, "SL_CODING_LEVEL", &coding_level.to_string());
    write_env(
        env_file,
        "SL_CODING_LEVEL_STYLE",
        &get_coding_level_style_name(coding_level),
    );
}

struct WriteContextArgs<'a> {
    source: &'a str,
    project_type: &'a str,
    package_manager: &'a str,
    cfg: &'a crate::hooks::SlConfig,
    git_root: &'a str,
    base_dir: &'a str,
    resolved: &'a PlanResolution,
    shadowed_cleanup: &'a ShadowedCleanupResult,
    team: Option<&'a TeamInfo>,
    coding_level: i32,
    session_id: &'a str,
}

fn write_session_context(a: WriteContextArgs) {
    let mut parts = vec![format!(
        "Project: {}",
        if a.project_type.is_empty() {
            "unknown"
        } else {
            a.project_type
        }
    )];
    if !a.package_manager.is_empty() {
        parts.push(format!("PM: {}", a.package_manager));
    }
    parts.push(format!("Plan naming: {}", a.cfg.plan.naming_format));
    if !a.git_root.is_empty() && a.git_root != a.base_dir {
        parts.push(format!("Root: {}", a.git_root));
    }
    if !a.resolved.path.is_empty() {
        if a.resolved.resolved_by == "session" {
            parts.push(format!("Plan: {}", a.resolved.path));
        } else {
            parts.push(format!("Suggested: {}", a.resolved.path));
        }
    }
    write_context(&format!("Session {}. {}\n", a.source, parts.join(" | ")));

    let sc = a.shadowed_cleanup;
    if !sc.restored.is_empty() || !sc.kept.is_empty() {
        write_context(
            "\n[!] SKILL-DEDUP CLEANUP (Issue #422): Recovered orphaned .shadowed/ directory.\n",
        );
        if !sc.restored.is_empty() {
            write_context(&format!(
                "Restored {} skill(s): {}\n",
                sc.restored.len(),
                sc.restored.join(", ")
            ));
        }
        if !sc.kept.is_empty() {
            write_context(&format!(
                "[!] Kept {} for review (content differs): {}\n",
                sc.kept.len(),
                sc.kept.join(", ")
            ));
        }
    }

    if let Some(t) = a.team {
        write_context(&format!(
            "[i] Agent Team detected: \"{}\" ({} members)\n",
            t.team_name, t.member_count
        ));
    }

    if !a.git_root.is_empty() && a.git_root != a.base_dir {
        write_context("Subdirectory mode: Plans/docs will be created in current directory\n");
        write_context(&format!("   Git root: {}\n", a.git_root));
    }

    if a.source == "compact" {
        write_context("\nCONTEXT COMPACTED - APPROVAL STATE CHECK:\nIf you were waiting for user approval via AskUserQuestion, you MUST re-confirm before proceeding.\n");

        if crate::hooks::is_hook_enabled("compaction-context-preservation") {
            let plan_path = get_env("SL_ACTIVE_PLAN").unwrap_or_default();
            let plan_path = if plan_path.is_empty() && a.resolved.resolved_by == "session" {
                a.resolved.path.clone()
            } else {
                plan_path
            };
            let recovery = build_compaction_context(&plan_path, a.session_id);
            if !recovery.is_empty() {
                write_context(&format!("\n{}\n", recovery));
            }
        }
    }

    let guidelines = get_coding_level_guidelines(a.coding_level);
    if !guidelines.is_empty() {
        write_context(&format!("\n{}\n", guidelines));
    }

    if !a.cfg.assertions.is_empty() {
        write_context("\nUser Assertions:\n");
        for (i, assertion) in a.cfg.assertions.iter().enumerate() {
            write_context(&format!("  {}. {}\n", i + 1, assertion));
        }
    }
}

// ── Helpers ──────────────────────────────────────────────────────────────────

struct ShadowedCleanupResult {
    restored: Vec<String>,
    skipped: Vec<String>,
    kept: Vec<String>,
}

fn cleanup_orphaned_shadowed_skills() -> ShadowedCleanupResult {
    let cwd = std::env::current_dir().unwrap_or_default();
    let shadowed_dir = cwd.join(".claude").join("skills").join(".shadowed");
    let skills_dir = cwd.join(".claude").join("skills");

    if !shadowed_dir.exists() {
        return ShadowedCleanupResult {
            restored: vec![],
            skipped: vec![],
            kept: vec![],
        };
    }

    let mut result = ShadowedCleanupResult {
        restored: vec![],
        skipped: vec![],
        kept: vec![],
    };
    let entries = match std::fs::read_dir(&shadowed_dir) {
        Ok(e) => e,
        Err(_) => return result,
    };

    for entry in entries.flatten() {
        if !entry.path().is_dir() {
            continue;
        }
        let name = entry.file_name().to_string_lossy().to_string();
        let src = shadowed_dir.join(&name);
        let dest = skills_dir.join(&name);

        if !dest.exists() {
            if std::fs::rename(&src, &dest).is_ok() {
                result.restored.push(name);
            } else {
                eprintln!("[session-init] Failed to restore \"{}\"", name);
            }
            continue;
        }

        let orphan_skill = src.join("SKILL.md");
        let local_skill = dest.join("SKILL.md");
        let orphan_data = std::fs::read_to_string(&orphan_skill);
        let local_data = std::fs::read_to_string(&local_skill);

        match (orphan_data, local_data) {
            (Ok(od), Ok(ld)) if od == ld => {
                let _ = std::fs::remove_dir_all(&src);
                result.skipped.push(name);
            }
            (Ok(_), Ok(_)) => {
                result.kept.push(name);
            }
            _ => {
                let _ = std::fs::remove_dir_all(&src);
                result.skipped.push(name);
            }
        }
    }

    let manifest = shadowed_dir.join(".dedup-manifest.json");
    let _ = std::fs::remove_file(&manifest);
    if let Ok(remaining) = std::fs::read_dir(&shadowed_dir) {
        if remaining.count() == 0 {
            let _ = std::fs::remove_dir(&shadowed_dir);
        }
    }

    result
}

struct TeamInfo {
    team_name: String,
    member_count: usize,
}

fn detect_agent_team() -> Option<TeamInfo> {
    let home = dirs_home_path();
    let teams_dir = home.join(".claude").join("teams");
    let entries = std::fs::read_dir(&teams_dir).ok()?;
    for entry in entries.flatten() {
        if !entry.path().is_dir() {
            continue;
        }
        let config_path = entry.path().join("config.json");
        let data = std::fs::read_to_string(&config_path).ok()?;
        let cfg: Value = serde_json::from_str(&data).ok()?;
        if let Some(members) = cfg.get("members").and_then(|m| m.as_array()) {
            if !members.is_empty() {
                return Some(TeamInfo {
                    team_name: entry.file_name().to_string_lossy().to_string(),
                    member_count: members.len(),
                });
            }
        }
    }
    None
}

fn try_resolve_with_sc(session_id: &str) -> Option<PlanResolution> {
    let sc = crate::hooks::find_sc_binary()?;
    let env_prefix = if !session_id.is_empty() {
        format!("SL_SESSION_ID={} ", session_id)
    } else {
        String::new()
    };
    let output = crate::hooks::exec_safe(&format!("{}{}  plan resolve", env_prefix, sc), "", 5000);
    if output.is_empty() {
        return None;
    }
    let v: Value = serde_json::from_str(&output).ok()?;
    let path = v.get("path")?.as_str()?.to_string();
    if path.is_empty() {
        return None;
    }
    let resolved_by = v
        .get("resolvedBy")
        .and_then(|r| r.as_str())
        .unwrap_or("")
        .to_string();
    Some(PlanResolution { path, resolved_by })
}

fn detect_node_version() -> String {
    let out = std::process::Command::new("node").arg("--version").output();
    match out {
        Ok(o) if o.status.success() => String::from_utf8_lossy(&o.stdout).trim().to_string(),
        _ => "N/A".to_string(),
    }
}

fn get_python_version() -> String {
    for cmd in &["python3", "python"] {
        if let Ok(o) = std::process::Command::new(cmd).arg("--version").output() {
            if o.status.success() {
                let v = String::from_utf8_lossy(&o.stdout).trim().to_string();
                let v2 = String::from_utf8_lossy(&o.stderr).trim().to_string();
                let result = if !v.is_empty() { v } else { v2 };
                if !result.is_empty() {
                    return result;
                }
            }
        }
    }
    String::new()
}

fn first_of(vals: &[&str]) -> String {
    vals.iter()
        .find(|v| !v.is_empty())
        .map(|s| s.to_string())
        .unwrap_or_default()
}

fn dirs_home_str() -> String {
    dirs_home_path().to_string_lossy().to_string()
}

fn dirs_home_path() -> std::path::PathBuf {
    dirs::home_dir().unwrap_or_else(|| std::path::PathBuf::from("."))
}
