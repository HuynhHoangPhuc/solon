/// StatusLine hook: render Claude Code status display from JSON stdin.
/// Falls back to "dir" on any error so Claude Code always gets a valid line.
use anyhow::Result;
use serde_json::Value;
use std::io::{self, Read};

/// autocompact buffer: 22.5% of 200k context (matches TS constant).
const AUTOCOMPACT_BUFFER: i64 = 45_000;
const USAGE_CACHE_FILE: &str = "sl-usage-limits-cache.json";

pub fn run() -> Result<()> {
    let mut buf = String::new();
    if io::stdin().read_to_string(&mut buf).is_err() || buf.trim().is_empty() {
        print_fallback();
        return Ok(());
    }
    if render_statusline(&buf).is_err() {
        print_fallback();
    }
    Ok(())
}

fn print_fallback() {
    let cwd = std::env::current_dir()
        .map(|p| p.to_string_lossy().to_string())
        .unwrap_or_else(|_| "unknown".to_string());
    println!("{}", crate::hooks::collapse_home(&cwd));
}

fn render_statusline(raw: &str) -> Result<()> {
    let data: Value = serde_json::from_str(raw)?;

    // Working directory
    let raw_dir = data
        .get("workspace")
        .and_then(|ws| ws.get("current_dir"))
        .and_then(|v| v.as_str())
        .or_else(|| data.get("cwd").and_then(|v| v.as_str()))
        .map(|s| s.to_string())
        .unwrap_or_else(|| {
            std::env::current_dir()
                .map(|p| p.to_string_lossy().to_string())
                .unwrap_or_default()
        });
    let current_dir = crate::hooks::collapse_home(&raw_dir);

    // Model name
    let model_name = data
        .get("model")
        .and_then(|m| m.get("display_name"))
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty())
        .unwrap_or("Claude")
        .to_string();

    // Git info (using exec_safe with cache handled externally)
    let git_info = get_git_info(&raw_dir);

    // Context window calculation
    // Priority: total_input+output tokens (cumulative) > used_percentage (native) > current_usage (last call)
    let mut context_percent: i64 = 0;
    let mut total_tokens: i64 = 0;
    let mut context_size: i64 = 0;
    if let Some(cw) = data.get("context_window").and_then(|v| v.as_object()) {
        context_size = json_i64(cw.get("context_window_size")).unwrap_or(0);
        // Try cumulative totals first (most accurate for session-wide view)
        let total_input = json_i64(cw.get("total_input_tokens")).unwrap_or(0);
        let total_output = json_i64(cw.get("total_output_tokens")).unwrap_or(0);
        if context_size > 0 && (total_input + total_output) > 0 {
            total_tokens = total_input + total_output;
            let compact_threshold = get_compact_threshold(context_size);
            let raw_pct = total_tokens as f64 / compact_threshold * 100.0;
            context_percent = raw_pct.round().clamp(0.0, 100.0) as i64;
        } else if let Some(pct) = json_i64(cw.get("used_percentage")) {
            // Fallback: use Claude Code's pre-calculated percentage
            context_percent = pct;
        } else if context_size > AUTOCOMPACT_BUFFER {
            // Last resort: current_usage from last API call
            if let Some(usage) = cw.get("current_usage").and_then(|v| v.as_object()) {
                let inp = json_i64(usage.get("input_tokens")).unwrap_or(0);
                let cache_create = json_i64(usage.get("cache_creation_input_tokens")).unwrap_or(0);
                let cache_read = json_i64(usage.get("cache_read_input_tokens")).unwrap_or(0);
                total_tokens = inp + cache_create + cache_read;
                let raw_pct =
                    (total_tokens + AUTOCOMPACT_BUFFER) as f64 / context_size as f64 * 100.0;
                context_percent = raw_pct.round().min(100.0) as i64;
            }
        }
    }

    // Write context cache for other hooks (preemptive-compaction, etc.)
    let session_id = data
        .get("session_id")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string();
    if !session_id.is_empty() && context_size > 0 {
        write_context_cache(
            &session_id,
            context_percent,
            total_tokens,
            context_size,
            &data,
        );
    }

    // Parse transcript
    let transcript_path = data
        .get("transcript_path")
        .and_then(|v| v.as_str())
        .unwrap_or("");
    let transcript = parse_transcript(transcript_path);

    // Usage limits cache
    let (session_text, usage_percent) = read_usage_limits_cache();

    // Lines changed
    let lines_added = data
        .get("cost")
        .and_then(|c| json_i64(c.get("total_lines_added")))
        .unwrap_or(0);
    let lines_removed = data
        .get("cost")
        .and_then(|c| json_i64(c.get("total_lines_removed")))
        .unwrap_or(0);

    // Build render context
    let ctx = RenderContext {
        model_name,
        current_dir,
        git_branch: git_info
            .as_ref()
            .map(|g| g.branch.clone())
            .unwrap_or_default(),
        git_unstaged: git_info.as_ref().map(|g| g.unstaged).unwrap_or(0),
        git_staged: git_info.as_ref().map(|g| g.staged).unwrap_or(0),
        git_ahead: git_info.as_ref().map(|g| g.ahead).unwrap_or(0),
        git_behind: git_info.as_ref().map(|g| g.behind).unwrap_or(0),
        context_percent,
        session_text,
        usage_percent,
        lines_added,
        lines_removed,
        transcript,
    };

    // Load config for statusline mode
    let cfg = crate::hooks::load_config();
    let mode = if cfg.statusline.is_empty() {
        "full".to_string()
    } else {
        cfg.statusline.clone()
    };

    let lines = match mode.as_str() {
        "none" => vec!["".to_string()],
        "minimal" => render_minimal(&ctx),
        "compact" => render_compact(&ctx),
        _ => render_full(&ctx),
    };
    for line in lines {
        println!("{}", line);
    }
    Ok(())
}

/// Compact threshold: the token count at which autocompaction triggers.
/// Research-based defaults: 200k→77.5%, 1M→33%.
fn get_compact_threshold(context_size: i64) -> f64 {
    match context_size {
        200_000 => 155_000.0,
        1_000_000 => 330_000.0,
        s if s >= 1_000_000 => s as f64 * 0.33,
        s => s as f64 * 0.775, // default ~77.5% for standard windows
    }
}

// ── Render context ────────────────────────────────────────────────────────────

struct RenderContext {
    model_name: String,
    current_dir: String,
    git_branch: String,
    git_unstaged: i64,
    git_staged: i64,
    git_ahead: i64,
    git_behind: i64,
    context_percent: i64,
    session_text: String,
    usage_percent: Option<i64>,
    lines_added: i64,
    lines_removed: i64,
    transcript: TranscriptData,
}

fn build_usage_string(ctx: &RenderContext) -> String {
    if ctx.session_text.is_empty() || ctx.session_text == "N/A" {
        return String::new();
    }
    let mut s = ctx.session_text.replace(" until reset", " left");
    if let Some(up) = ctx.usage_percent {
        s.push_str(&format!(" ({}%)", up));
    }
    s
}

fn render_session_lines(ctx: &RenderContext) -> Vec<String> {
    let term_width = get_terminal_width();
    let threshold = (term_width * 85) / 100;

    let dir_part = format!("📁 {}", ctx.current_dir);

    let branch_part = if !ctx.git_branch.is_empty() {
        let mut indicators: Vec<String> = Vec::new();
        if ctx.git_unstaged > 0 {
            indicators.push(format!("{}", ctx.git_unstaged));
        }
        if ctx.git_staged > 0 {
            indicators.push(format!("+{}", ctx.git_staged));
        }
        if ctx.git_ahead > 0 {
            indicators.push(format!("{}↑", ctx.git_ahead));
        }
        if ctx.git_behind > 0 {
            indicators.push(format!("{}↓", ctx.git_behind));
        }
        let branch = format!("🌿 {}", ctx.git_branch);
        if indicators.is_empty() {
            branch
        } else {
            let ind_str =
                crate::hooks::colorize(&format!("({})", indicators.join(", ")), "\x1b[33m");
            format!("{} {}", branch, ind_str)
        }
    } else {
        String::new()
    };

    let location_part = if !branch_part.is_empty() {
        format!("{}  {}", dir_part, branch_part)
    } else {
        dir_part.clone()
    };

    let mut session_part = format!("🤖 {}", ctx.model_name);
    if ctx.context_percent > 0 {
        session_part.push_str(&format!(
            "  {} {}%",
            crate::hooks::colored_bar(ctx.context_percent, 12),
            ctx.context_percent
        ));
    }
    {
        let usage_str = build_usage_string(ctx);
        if !usage_str.is_empty() {
            session_part.push_str(&format!("  ⌛ {}", usage_str.replace(")", " used)")));
        }
    }

    let stats_part = if ctx.lines_added > 0 || ctx.lines_removed > 0 {
        format!(
            "📝 {} {}",
            crate::hooks::colorize(&format!("+{}", ctx.lines_added), "\x1b[32m"),
            crate::hooks::colorize(&format!("-{}", ctx.lines_removed), "\x1b[31m"),
        )
    } else {
        String::new()
    };

    let stats_len = visible_length(&stats_part);
    let all_one_line = format!("{}  {}  {}", session_part, location_part, stats_part);
    let session_location = format!("{}  {}", session_part, location_part);

    let mut lines: Vec<String> = Vec::new();
    if visible_length(&all_one_line) <= threshold && stats_len > 0 {
        lines.push(all_one_line);
    } else if visible_length(&session_location) <= threshold {
        lines.push(session_location);
        if stats_len > 0 {
            lines.push(stats_part);
        }
    } else if visible_length(&session_part) <= threshold {
        lines.push(session_part);
        lines.push(location_part);
        if stats_len > 0 {
            lines.push(stats_part);
        }
    } else {
        lines.push(session_part);
        lines.push(dir_part);
        if !branch_part.is_empty() {
            lines.push(branch_part);
        }
        if stats_len > 0 {
            lines.push(stats_part);
        }
    }
    lines
}

fn render_full(ctx: &RenderContext) -> Vec<String> {
    let mut lines = render_session_lines(ctx);
    lines.extend(render_agents_lines(&ctx.transcript));
    if let Some(todo_line) = render_todos_line(&ctx.transcript) {
        lines.push(todo_line);
    }
    lines
}

fn render_compact(ctx: &RenderContext) -> Vec<String> {
    let mut line1 = format!("🤖 {}", ctx.model_name);
    if ctx.context_percent > 0 {
        line1.push_str(&format!(
            "  {} {}%",
            crate::hooks::colored_bar(ctx.context_percent, 12),
            ctx.context_percent
        ));
    }
    {
        let usage_str = build_usage_string(ctx);
        if !usage_str.is_empty() {
            line1.push_str(&format!("  ⌛ {}", usage_str));
        }
    }
    let mut line2 = format!("📁 {}", ctx.current_dir);
    if !ctx.git_branch.is_empty() {
        line2.push_str(&format!("  🌿 {}", ctx.git_branch));
    }
    vec![line1, line2]
}

fn render_minimal(ctx: &RenderContext) -> Vec<String> {
    let mut parts = vec![format!("🤖 {}", ctx.model_name)];
    if ctx.context_percent > 0 {
        let battery = if ctx.context_percent > 70 {
            crate::hooks::colorize("🔋", "\x1b[31m")
        } else {
            "🔋".to_string()
        };
        parts.push(format!("{} {}%", battery, ctx.context_percent));
    }
    {
        let usage_str = build_usage_string(ctx);
        if !usage_str.is_empty() {
            parts.push(format!("⏰ {}", usage_str));
        }
    }
    if !ctx.git_branch.is_empty() {
        parts.push(format!("🌿 {}", ctx.git_branch));
    }
    parts.push(format!("📁 {}", ctx.current_dir));
    vec![parts.join("  ")]
}

// ── Agent/todo transcript rendering ──────────────────────────────────────────

fn render_agents_lines(transcript: &TranscriptData) -> Vec<String> {
    if transcript.agents.is_empty() {
        return Vec::new();
    }

    let mut sorted = transcript.agents.clone();
    sorted.sort_by_key(|a| a.start_ms);

    let running: Vec<TranscriptAgent> = sorted
        .iter()
        .filter(|a| a.status == "running")
        .cloned()
        .collect();
    let completed: Vec<TranscriptAgent> = sorted
        .iter()
        .filter(|a| a.status != "running")
        .cloned()
        .collect();
    let mut all_agents: Vec<TranscriptAgent> = running.clone();
    all_agents.extend(completed.clone());
    all_agents.sort_by_key(|a| a.start_ms);

    // Collapse consecutive same-type/status runs
    let mut collapsed: Vec<(String, String, usize, Option<TranscriptAgent>)> = Vec::new(); // (type, status, count, last)
    for a in &all_agents {
        let atype = if a.agent_type.is_empty() {
            "agent".to_string()
        } else {
            a.agent_type.clone()
        };
        if let Some(last) = collapsed.last_mut() {
            if last.0 == atype && last.1 == a.status {
                last.2 += 1;
                last.3 = Some(a.clone());
                continue;
            }
        }
        collapsed.push((atype, a.status.clone(), 1, Some(a.clone())));
    }

    // Show last 4 groups
    let to_show = if collapsed.len() > 4 {
        &collapsed[collapsed.len() - 4..]
    } else {
        &collapsed[..]
    };

    let flow_parts: Vec<String> = to_show
        .iter()
        .map(|(atype, status, count, _)| {
            let icon = if status == "running" {
                crate::hooks::colorize("●", "\x1b[33m")
            } else {
                crate::hooks::colorize("○", "\x1b[2m")
            };
            let suffix = if *count > 1 {
                format!(" ×{}", count)
            } else {
                String::new()
            };
            format!("{} {}{}", icon, atype, suffix)
        })
        .collect();

    let completed_count = completed.len();
    let flow_suffix = if completed_count > 2 {
        format!(
            " {}",
            crate::hooks::colorize(&format!("({} done)", completed_count), "\x1b[2m")
        )
    } else {
        String::new()
    };

    let mut lines = vec![format!("{}{}", flow_parts.join(" → "), flow_suffix)];

    let detail = running.first().or_else(|| completed.last());
    if let Some(agent) = detail {
        if !agent.description.is_empty() {
            let desc = if agent.description.len() > 50 {
                format!("{}...", &agent.description[..47])
            } else {
                agent.description.clone()
            };
            let elapsed = format_elapsed_ms(agent.start_ms, agent.end_ms);
            let icon = if agent.status == "running" {
                crate::hooks::colorize("▸", "\x1b[33m")
            } else {
                crate::hooks::colorize("▸", "\x1b[2m")
            };
            lines.push(format!(
                "   {} {} {}",
                icon,
                desc,
                crate::hooks::colorize(&format!("({})", elapsed), "\x1b[2m")
            ));
        }
    }
    lines
}

fn render_todos_line(transcript: &TranscriptData) -> Option<String> {
    if transcript.todos.is_empty() {
        return None;
    }
    let todos = &transcript.todos;
    let total = todos.len();
    let mut in_progress: Option<&TranscriptTodo> = None;
    let mut completed_count = 0usize;
    let mut pending_count = 0usize;
    for t in todos {
        match t.status.as_str() {
            "in_progress" => {
                if in_progress.is_none() {
                    in_progress = Some(t);
                }
            }
            "completed" => completed_count += 1,
            "pending" => pending_count += 1,
            _ => {}
        }
    }

    if let Some(ip) = in_progress {
        let display = {
            let raw = if !ip.active_form.is_empty() {
                &ip.active_form
            } else {
                &ip.content
            };
            if raw.len() > 50 {
                format!("{}...", &raw[..47])
            } else {
                raw.clone()
            }
        };
        return Some(format!(
            "{} {} {}",
            crate::hooks::colorize("▸", "\x1b[33m"),
            display,
            crate::hooks::colorize(
                &format!("({} done, {} pending)", completed_count, pending_count),
                "\x1b[2m"
            )
        ));
    }

    if completed_count == total && total > 0 {
        return Some(format!(
            "{} All {} todos complete",
            crate::hooks::colorize("✓", "\x1b[32m"),
            total
        ));
    }

    if pending_count > 0 {
        let next = todos.iter().find(|t| t.status == "pending");
        let next_task = next
            .map(|t| {
                if t.content.len() > 40 {
                    format!("{}...", &t.content[..37])
                } else {
                    t.content.clone()
                }
            })
            .unwrap_or_else(|| "Next task".to_string());
        return Some(format!(
            "{} Next: {} {}",
            crate::hooks::colorize("○", "\x1b[2m"),
            next_task,
            crate::hooks::colorize(
                &format!("({} done, {} pending)", completed_count, pending_count),
                "\x1b[2m"
            )
        ));
    }
    None
}

// ── Transcript parsing ────────────────────────────────────────────────────────

#[derive(Clone)]
struct TranscriptAgent {
    agent_type: String,
    description: String,
    status: String,
    start_ms: i64,
    end_ms: i64,
}

#[derive(Clone)]
struct TranscriptTodo {
    content: String,
    active_form: String,
    status: String,
}

struct TranscriptData {
    agents: Vec<TranscriptAgent>,
    todos: Vec<TranscriptTodo>,
}

fn parse_transcript(path: &str) -> TranscriptData {
    let mut result = TranscriptData {
        agents: Vec::new(),
        todos: Vec::new(),
    };
    if path.is_empty() {
        return result;
    }

    let data = match std::fs::read_to_string(path) {
        Ok(d) => d,
        Err(_) => return result,
    };

    let mut agent_map: std::collections::HashMap<String, TranscriptAgent> =
        std::collections::HashMap::new();
    let mut latest_todos: Vec<TranscriptTodo> = Vec::new();

    for line in data.lines() {
        if line.is_empty() {
            continue;
        }
        let entry: Value = match serde_json::from_str(line) {
            Ok(v) => v,
            Err(_) => continue,
        };
        let ts_ms = parse_time_ms(entry.get("timestamp"));

        let message = match entry.get("message").and_then(|m| m.as_object()) {
            Some(m) => m,
            None => continue,
        };
        let content = match message.get("content").and_then(|c| c.as_array()) {
            Some(c) => c,
            None => continue,
        };

        for block in content {
            let block_type = block.get("type").and_then(|t| t.as_str()).unwrap_or("");
            if block_type == "tool_use" {
                let id = block
                    .get("id")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                let name = block
                    .get("name")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                if id.is_empty() || name.is_empty() {
                    continue;
                }
                let input = block.get("input");

                match name.as_str() {
                    "Task" => {
                        let atype = input
                            .and_then(|i| i.get("subagent_type"))
                            .and_then(|v| v.as_str())
                            .unwrap_or("unknown")
                            .to_string();
                        let desc = input
                            .and_then(|i| i.get("description"))
                            .and_then(|v| v.as_str())
                            .unwrap_or("")
                            .to_string();
                        agent_map.insert(
                            id.clone(),
                            TranscriptAgent {
                                agent_type: atype,
                                description: desc,
                                status: "running".to_string(),
                                start_ms: ts_ms,
                                end_ms: 0,
                            },
                        );
                    }
                    "TodoWrite" => {
                        if let Some(todos) = input
                            .and_then(|i| i.get("todos"))
                            .and_then(|v| v.as_array())
                        {
                            latest_todos = todos
                                .iter()
                                .filter_map(|t| {
                                    let content = t.get("content")?.as_str()?.to_string();
                                    let status = t
                                        .get("status")
                                        .and_then(|v| v.as_str())
                                        .unwrap_or("")
                                        .to_string();
                                    let active_form = t
                                        .get("activeForm")
                                        .and_then(|v| v.as_str())
                                        .unwrap_or("")
                                        .to_string();
                                    Some(TranscriptTodo {
                                        content,
                                        status,
                                        active_form,
                                    })
                                })
                                .collect();
                        }
                    }
                    _ => {} // Non-agent tools are not tracked in statusline
                }
            } else if block_type == "tool_result" {
                let tool_use_id = block
                    .get("tool_use_id")
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                if tool_use_id.is_empty() {
                    continue;
                }
                if let Some(a) = agent_map.get_mut(&tool_use_id) {
                    a.status = "completed".to_string();
                    a.end_ms = ts_ms;
                }
            }
        }
    }

    // Keep last 10 agents
    let mut agents: Vec<TranscriptAgent> = agent_map.into_values().collect();
    agents.sort_by_key(|a| a.start_ms);
    if agents.len() > 10 {
        agents = agents[agents.len() - 10..].to_vec();
    }

    result.agents = agents;
    result.todos = latest_todos;
    result
}

fn parse_time_ms(v: Option<&Value>) -> i64 {
    let s = match v.and_then(|v| v.as_str()) {
        Some(s) => s,
        None => return 0,
    };
    if let Ok(t) = chrono::DateTime::parse_from_rfc3339(s) {
        return t.timestamp_millis();
    }
    0
}

// ── Context cache ─────────────────────────────────────────────────────────────

fn write_context_cache(session_id: &str, percent: i64, tokens: i64, size: i64, data: &Value) {
    let path = std::env::temp_dir().join(format!("sl-context-{}.json", session_id));
    let usage = data
        .get("context_window")
        .and_then(|cw| cw.get("current_usage"))
        .cloned()
        .unwrap_or(Value::Null);
    let payload = serde_json::json!({
        "percent": percent,
        "tokens": tokens,
        "size": size,
        "usage": usage,
        "timestamp": chrono::Utc::now().timestamp_millis(),
    });
    if let Ok(b) = serde_json::to_string(&payload) {
        let _ = std::fs::write(path, b);
    }
}

// ── Usage limits cache ────────────────────────────────────────────────────────

fn read_usage_limits_cache() -> (String, Option<i64>) {
    let path = std::env::temp_dir().join(USAGE_CACHE_FILE);
    let data = match std::fs::read_to_string(&path) {
        Ok(d) => d,
        Err(_) => return (String::new(), None),
    };
    let cache: Value = match serde_json::from_str(&data) {
        Ok(v) => v,
        Err(_) => return (String::new(), None),
    };

    if cache.get("status").and_then(|s| s.as_str()) == Some("unavailable") {
        return ("N/A".to_string(), None);
    }

    let five_hour = match cache.get("data").and_then(|d| d.get("five_hour")) {
        Some(v) => v,
        None => return (String::new(), None),
    };

    let usage_percent = five_hour
        .get("utilization")
        .and_then(|v| v.as_f64())
        .map(|f| f.round() as i64);

    let session_text = five_hour
        .get("resets_at")
        .and_then(|v| v.as_str())
        .filter(|s| !s.is_empty())
        .and_then(|reset_at| {
            let remaining = parse_reset_remaining(reset_at);
            if remaining > 0 && remaining < 18000 {
                let rh = remaining / 3600;
                let rm = (remaining % 3600) / 60;
                Some(format!("{}h {}m until reset", rh, rm))
            } else {
                None
            }
        })
        .unwrap_or_default();

    (session_text, usage_percent)
}

fn parse_reset_remaining(reset_at: &str) -> i64 {
    if let Ok(t) = chrono::DateTime::parse_from_rfc3339(reset_at) {
        let now = chrono::Utc::now();
        let diff = t.with_timezone(&chrono::Utc).signed_duration_since(now);
        let secs = diff.num_seconds();
        if secs > 0 {
            return secs;
        }
    }
    0
}

// ── Git info ──────────────────────────────────────────────────────────────────

struct GitInfo {
    branch: String,
    unstaged: i64,
    staged: i64,
    ahead: i64,
    behind: i64,
}

fn get_git_info(cwd: &str) -> Option<GitInfo> {
    let mut branch = crate::hooks::exec_safe("git branch --show-current", cwd, 3000);
    // Fallback: detached HEAD or older git versions
    if branch.is_empty() {
        branch = crate::hooks::exec_safe("git rev-parse --short HEAD", cwd, 3000);
    }
    if branch.is_empty() {
        return None;
    }

    let status_out = crate::hooks::exec_safe("git status --porcelain", cwd, 3000);
    let mut unstaged = 0i64;
    let mut staged = 0i64;
    for line in status_out.lines() {
        if line.len() >= 2 {
            let x = &line[..1];
            let y = &line[1..2];
            if x != " " && x != "?" {
                staged += 1;
            }
            if y != " " && y != "?" {
                unstaged += 1;
            }
        }
    }

    let ab_out = crate::hooks::exec_safe(
        "git rev-list --count --left-right @{upstream}...HEAD 2>/dev/null",
        cwd,
        3000,
    );
    let mut ahead = 0i64;
    let mut behind = 0i64;
    let parts: Vec<&str> = ab_out.split_whitespace().collect();
    if parts.len() == 2 {
        behind = parts[0].parse().unwrap_or(0);
        ahead = parts[1].parse().unwrap_or(0);
    }

    Some(GitInfo {
        branch,
        unstaged,
        staged,
        ahead,
        behind,
    })
}

// ── Utility helpers ───────────────────────────────────────────────────────────

fn json_i64(v: Option<&Value>) -> Option<i64> {
    match v? {
        Value::Number(n) => n.as_i64(),
        _ => None,
    }
}

/// Visible column width: strips ANSI codes, counts emoji as 2 columns.
fn visible_length(s: &str) -> usize {
    if s.is_empty() {
        return 0;
    }
    // Strip ANSI escape sequences
    let no_ansi = {
        let re = regex::Regex::new(r"\x1b\[[0-9;]*m").unwrap();
        re.replace_all(s, "").to_string()
    };
    let mut width = 0usize;
    for c in no_ansi.chars() {
        let cp = c as u32;
        if (0x1f300..=0x1f9ff).contains(&cp)
            || (0x2600..=0x26ff).contains(&cp)
            || (0x2700..=0x27bf).contains(&cp)
        {
            width += 2;
        } else {
            width += 1;
        }
    }
    width
}

fn get_terminal_width() -> usize {
    if let Ok(col) = std::env::var("COLUMNS") {
        if let Ok(n) = col.parse::<usize>() {
            if n > 0 {
                return n;
            }
        }
    }
    120
}

fn format_elapsed_ms(start_ms: i64, end_ms: i64) -> String {
    if start_ms == 0 {
        return "0s".to_string();
    }
    let end = if end_ms == 0 {
        chrono::Utc::now().timestamp_millis()
    } else {
        end_ms
    };
    let ms = end - start_ms;
    if ms < 0 || ms < 1000 {
        return "<1s".to_string();
    }
    if ms < 60_000 {
        return format!("{}s", ms / 1000);
    }
    let mins = ms / 60_000;
    let secs = (ms % 60_000) / 1000;
    format!("{}m {}s", mins, secs)
}
