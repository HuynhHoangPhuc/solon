/// Plan naming utilities: slug sanitization, date formatting, naming pattern resolution.
/// Ported from Go solon-core/internal/plan/naming.go
use chrono::Local;
use regex::Regex;
use solon_common::PlanConfig;

/// Remove invalid filename chars, convert to kebab-case, truncate at 100.
pub fn sanitize_slug(slug: &str) -> String {
    if slug.is_empty() {
        return String::new();
    }
    // Replace invalid filename chars with hyphens (matches Go behaviour of
    // first stripping then converting non-alnum — but we use hyphen to
    // preserve word boundaries across slashes and colons)
    let invalid = Regex::new(r#"[<>:"/\\|?*\x00-\x1f\x7f]"#).unwrap();
    let s = invalid.replace_all(slug, "-");
    // Replace remaining non-alphanumeric-hyphen chars with hyphen
    let non_alnum = Regex::new(r"[^a-zA-Z0-9\-]").unwrap();
    let s = non_alnum.replace_all(&s, "-");
    // Collapse multiple hyphens
    let multi_hyphen = Regex::new(r"-+").unwrap();
    let s = multi_hyphen.replace_all(&s, "-");
    let s = s.trim_matches('-').to_string();
    if s.len() > 100 {
        s[..100].to_string()
    } else {
        s
    }
}

/// Format a date pattern using current local time.
/// Supported tokens: YYYY, YY, MM, DD, HH, mm, ss
pub fn format_date(format: &str) -> String {
    let now = Local::now();
    format
        .replace("YYYY", &format!("{:04}", now.format("%Y")))
        .replace("YY", &format!("{:02}", now.format("%y")))
        .replace("MM", &format!("{:02}", now.format("%m")))
        .replace("DD", &format!("{:02}", now.format("%d")))
        .replace("HH", &format!("{:02}", now.format("%H")))
        .replace("mm", &format!("{:02}", now.format("%M")))
        .replace("ss", &format!("{:02}", now.format("%S")))
}

/// Extract an issue number from a branch name.
pub fn extract_issue_from_branch(branch: &str) -> String {
    if branch.is_empty() {
        return String::new();
    }
    let patterns = [
        Regex::new(r"(?i)(?:issue|gh|fix|feat|bug)[/\-]?(\d+)").unwrap(),
        Regex::new(r"[/\-](\d+)[/\-]").unwrap(),
        Regex::new(r"#(\d+)").unwrap(),
    ];
    for re in &patterns {
        if let Some(cap) = re.captures(branch) {
            if let Some(m) = cap.get(1) {
                return m.as_str().to_string();
            }
        }
    }
    String::new()
}

/// Format an issue ID with the configured prefix.
pub fn format_issue_id(issue_id: &str, plan_config: &PlanConfig) -> String {
    if issue_id.is_empty() {
        return String::new();
    }
    if let Some(prefix) = &plan_config.issue_prefix {
        if !prefix.is_empty() {
            return format!("{}{}", prefix, issue_id);
        }
    }
    format!("#{}", issue_id)
}

/// Resolve the naming pattern with date and optional issue substituted.
/// Keeps {slug} as a placeholder for callers to substitute.
pub fn resolve_naming_pattern(plan_config: &PlanConfig, git_branch: &str) -> String {
    let formatted_date = format_date(&plan_config.date_format);
    let issue_id = extract_issue_from_branch(git_branch);

    let full_issue = if !issue_id.is_empty() {
        if let Some(prefix) = &plan_config.issue_prefix {
            if !prefix.is_empty() {
                format!("{}{}", prefix, issue_id)
            } else {
                String::new()
            }
        } else {
            String::new()
        }
    } else {
        String::new()
    };

    let mut pattern = plan_config.naming_format.clone();
    pattern = pattern.replace("{date}", &formatted_date);

    if !full_issue.is_empty() {
        pattern = pattern.replace("{issue}", &full_issue);
    } else {
        // Remove {issue} placeholder and surrounding hyphens
        let re_issue = Regex::new(r"-?\{issue\}-?").unwrap();
        pattern = re_issue.replace_all(&pattern, "-").to_string();
        let re_multi = Regex::new(r"--+").unwrap();
        pattern = re_multi.replace_all(&pattern, "-").to_string();
    }

    pattern = pattern.trim_start_matches('-').to_string();
    pattern = pattern.trim_end_matches('-').to_string();

    // Clean up hyphens around {slug}
    let re_pre = Regex::new(r"-+(\{slug\})").unwrap();
    pattern = re_pre.replace_all(&pattern, "-$1").to_string();
    let re_post = Regex::new(r"(\{slug\})-+").unwrap();
    pattern = re_post.replace_all(&pattern, "$1-").to_string();
    let re_multi = Regex::new(r"--+").unwrap();
    pattern = re_multi.replace_all(&pattern, "-").to_string();

    pattern
}

/// Build the full plan directory name with slug substituted.
pub fn build_plan_dir_name(plan_config: &PlanConfig, git_branch: &str, slug: &str) -> String {
    let pattern = resolve_naming_pattern(plan_config, git_branch);
    let sanitized = if slug.is_empty() {
        "untitled".to_string()
    } else {
        let s = sanitize_slug(slug);
        if s.is_empty() {
            "untitled".to_string()
        } else {
            s
        }
    };
    pattern.replace("{slug}", &sanitized)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sanitize_slug_basic() {
        assert_eq!(sanitize_slug("my feature plan"), "my-feature-plan");
        assert_eq!(sanitize_slug("hello--world"), "hello-world");
        assert_eq!(sanitize_slug(""), "");
    }

    #[test]
    fn test_sanitize_slug_strips_invalid() {
        assert_eq!(sanitize_slug("feat/my:feature"), "feat-my-feature");
    }

    #[test]
    fn test_sanitize_slug_truncates_at_100() {
        let long = "a".repeat(120);
        assert_eq!(sanitize_slug(&long).len(), 100);
    }

    #[test]
    fn test_extract_issue_from_branch() {
        assert_eq!(extract_issue_from_branch("feat/issue-42-my-feature"), "42");
        assert_eq!(extract_issue_from_branch("fix/123-bug"), "123");
        assert_eq!(extract_issue_from_branch("main"), "");
    }

    #[test]
    fn test_build_plan_dir_name_no_branch() {
        let cfg = PlanConfig::default();
        let name = build_plan_dir_name(&cfg, "", "my-feature");
        assert!(name.contains("my-feature"));
        // Should not contain {slug} literal
        assert!(!name.contains("{slug}"));
    }
}
