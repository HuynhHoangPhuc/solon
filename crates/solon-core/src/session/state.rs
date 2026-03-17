/// Per-session state persisted to /tmp/sl-session-{id}.json.
/// Atomic writes via temp file + rename to prevent corruption.
/// Ported from Go solon-core/internal/session/state.go
use solon_common::SessionState;
use std::path::PathBuf;

/// Returns the temp file path for a session ID.
pub fn get_session_temp_path(session_id: &str) -> PathBuf {
    let tmp = std::env::temp_dir();
    tmp.join(format!("sl-session-{}.json", session_id))
}

/// Reads session state from the temp file.
/// Returns None if session_id is empty, file missing, or parse fails.
pub fn read_session_state(session_id: &str) -> Option<SessionState> {
    if session_id.is_empty() {
        return None;
    }
    let path = get_session_temp_path(session_id);
    let data = std::fs::read_to_string(path).ok()?;
    serde_json::from_str(&data).ok()
}

/// Atomically writes session state to the temp file. Returns true on success.
pub fn write_session_state(session_id: &str, state: &SessionState) -> bool {
    if session_id.is_empty() {
        return false;
    }
    let target = get_session_temp_path(session_id);
    let tmp_path = format!("{}.{:x}", target.display(), rand_suffix());

    let data = match serde_json::to_string_pretty(state) {
        Ok(d) => d,
        Err(_) => return false,
    };

    if std::fs::write(&tmp_path, data).is_err() {
        return false;
    }

    if std::fs::rename(&tmp_path, &target).is_err() {
        let _ = std::fs::remove_file(&tmp_path);
        return false;
    }
    true
}

fn rand_suffix() -> u64 {
    use std::time::{SystemTime, UNIX_EPOCH};
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.subsec_nanos() as u64)
        .unwrap_or(12345)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_empty_session_id_returns_none() {
        assert!(read_session_state("").is_none());
    }

    #[test]
    fn test_session_path_format() {
        let p = get_session_temp_path("abc123");
        assert!(p.to_string_lossy().contains("sl-session-abc123.json"));
    }

    #[test]
    fn test_write_and_read_session() {
        let id = "test-write-read-session-rs";
        let state = SessionState {
            session_origin: "/tmp/test".to_string(),
            active_plan: Some("plans/my-plan".to_string()),
            suggested_plan: None,
            timestamp: 1234567890,
            source: "test".to_string(),
        };
        assert!(write_session_state(id, &state));
        let loaded = read_session_state(id).unwrap();
        assert_eq!(loaded.active_plan, Some("plans/my-plan".to_string()));
        // cleanup
        let _ = std::fs::remove_file(get_session_temp_path(id));
    }
}
