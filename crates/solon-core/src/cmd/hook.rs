/// Hook subcommand dispatcher: routes hook event names to their handler modules.
/// Each handler reads JSON from stdin, processes logic, writes JSON to stdout.
/// Exit 0 = allow/continue, exit 2 = explicit block.
use anyhow::Result;
use clap::Subcommand;

#[derive(Subcommand)]
pub enum HookCommand {
    /// PostToolUse: detect AI-generated comment patterns in code edits
    CommentSlopChecker,
    /// SubagentStop: remind to run /sl:ship after plan agent
    ShipReminder,
    /// PreToolUse(Write): inject file naming guidance as allow response
    DescriptiveName,
    /// UserPromptSubmit: inject full dev rules reminder (throttled)
    DevRules,
    /// UserPromptSubmit: classify intent and inject compact strategy guidance
    IntentGate,
    /// PostToolUse(Edit/Write/MultiEdit): track edits, remind to run code-simplifier
    PostEdit,
    /// PostToolUse: warn when context window is near capacity
    PreemptiveCompaction,
    /// PreToolUse: block access to privacy-sensitive files
    PrivacyBlock,
    /// PreToolUse: block .slignore-listed paths and overly broad glob patterns
    ScoutBlock,
    /// SessionStart: detect project, resolve plan path, inject context
    SessionInit,
    /// StatusLine: render Claude Code status display
    Statusline {
        /// Dump raw stdin JSON to temp file for debugging
        #[arg(long, default_value_t = false)]
        debug: bool,
    },
    /// SubagentStart: build compact context block for subagents
    SubagentInit,
    /// TaskCompleted: log task completion and inject progress context
    TaskCompleted,
    /// SubagentStart: inject team peer list and task context for team agents
    TeamContext,
    /// TeammateIdle: inject available task context when teammate goes idle
    TeammateIdle,
    /// UserPromptSubmit: remind about incomplete plan todos
    TodoContinuationEnforcer,
    /// PostToolUse: truncate large tool outputs to save context window space
    ToolOutputTruncation,
    /// UserPromptSubmit/PostToolUse: fetch and cache Anthropic usage limits
    UsageAwareness,
    /// Print version string
    Version,
    /// SubagentStop: extract and accumulate learnings from transcript
    WisdomAccumulator,
}

pub fn run(cmd: HookCommand) -> Result<()> {
    match cmd {
        HookCommand::CommentSlopChecker => crate::hooks::comment_slop_checker::run(),
        HookCommand::ShipReminder => crate::hooks::ship_reminder::run(),
        HookCommand::DescriptiveName => crate::hooks::descriptive_name::run(),
        HookCommand::DevRules => crate::hooks::dev_rules::run(),
        HookCommand::IntentGate => crate::hooks::intent_gate::run(),
        HookCommand::PostEdit => crate::hooks::post_edit::run(),
        HookCommand::PreemptiveCompaction => crate::hooks::preemptive_compaction::run(),
        HookCommand::PrivacyBlock => crate::hooks::privacy_block::run(),
        HookCommand::ScoutBlock => crate::hooks::scout_block::run(),
        HookCommand::SessionInit => crate::hooks::session_init::run(),
        HookCommand::Statusline { debug } => crate::hooks::statusline::run(debug),
        HookCommand::SubagentInit => crate::hooks::subagent_init::run(),
        HookCommand::TaskCompleted => crate::hooks::task_completed::run(),
        HookCommand::TeamContext => crate::hooks::team_context::run(),
        HookCommand::TeammateIdle => crate::hooks::teammate_idle::run(),
        HookCommand::TodoContinuationEnforcer => crate::hooks::todo_continuation_enforcer::run(),
        HookCommand::ToolOutputTruncation => crate::hooks::tool_output_truncation::run(),
        HookCommand::UsageAwareness => crate::hooks::usage_awareness::run(),
        HookCommand::Version => crate::hooks::version::run(),
        HookCommand::WisdomAccumulator => crate::hooks::wisdom_accumulator::run(),
    }
}
