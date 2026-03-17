use crate::hooks::{get_env, is_hook_enabled, read_session_state, write_context};
/// SubagentStop(Plan): remind to run /cook after plan agent finishes.
use anyhow::Result;

pub fn run() -> Result<()> {
    if !is_hook_enabled("cook-after-plan-reminder") {
        std::process::exit(0);
    }

    // Consume stdin (not used)
    let _ = crate::hooks::read_hook_input();

    let session_id = get_env("SL_SESSION_ID").unwrap_or_default();
    let plan_path = if !session_id.is_empty() {
        read_session_state(&session_id)
            .and_then(|s| s.active_plan)
            .map(|p| {
                if std::path::Path::new(&p).is_absolute() {
                    p
                } else {
                    // resolve relative to session origin
                    read_session_state(&session_id)
                        .and_then(|s| {
                            if s.session_origin.is_empty() {
                                None
                            } else {
                                Some(format!("{}/{}", s.session_origin, p))
                            }
                        })
                        .unwrap_or(p)
                }
            })
            .unwrap_or_default()
    } else {
        String::new()
    };

    write_context("MUST invoke /solon:cook --auto skill before implementing the plan\n");
    if !plan_path.is_empty() {
        write_context(&format!(
            "Best Practice: Run /clear then /solon:cook {}/plan.md\n",
            plan_path
        ));
    } else {
        write_context(
            "Best Practice: Run /clear then /solon:cook {full-absolute-path-to-plan.md}\n",
        );
    }

    Ok(())
}
