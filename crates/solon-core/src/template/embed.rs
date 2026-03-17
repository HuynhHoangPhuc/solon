/// Embedded plan and phase templates.
/// Ported from Go solon-core/internal/template/embed.go

/// Render a plan.md from the template.
pub fn render_plan(slug: &str, _mode: &str, _dir_name: &str) -> String {
    let title = to_title_case(slug);
    format!(
        r#"---
status: pending
priority: P1
effort: TBD
branch: TBD
tags: []
---

# {title}

## Summary

<!-- Brief description of this plan -->

## Architecture

<!-- High-level architecture overview -->

## Phases

| # | Phase | Status | Priority | Effort | Blocked By |
|---|-------|--------|----------|--------|------------|

<!-- Phase table populated by phase files -->

## Key Dependencies

<!-- List key dependencies -->

## Key Decisions

<!-- Document key architectural decisions -->

## Risks

| Risk | Mitigation |
|------|------------|

<!-- Identify risks and mitigations -->
"#
    )
}

/// Render a phase-XX-*.md from the template.
pub fn render_phase(title: &str, _total_phases: usize) -> String {
    format!(
        r#"# Phase: {title}

## Overview
- **Priority:** TBD
- **Status:** Pending
- **Effort:** TBD
- **Description:** <!-- Brief description -->

## Key Insights

<!-- Important findings from research -->

## Requirements

### Functional
<!-- Functional requirements -->

### Non-Functional
<!-- Non-functional requirements -->

## Architecture

<!-- System design, component interactions, data flow -->

## Related Code Files

### To reference:
<!-- Files to read/reference -->

### To create:
<!-- Files to create -->

### To modify:
<!-- Files to modify -->

## Implementation Steps

1. <!-- Step 1 -->

## TODO

- [ ] <!-- Task 1 -->

## Success Criteria

<!-- Definition of done -->

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
"#
    )
}

/// Render a red-team review prompt for a plan directory.
pub fn render_red_team(plan_dir: &str, phases: &[String]) -> String {
    let phase_list: String = phases.iter().map(|p| format!("- {}\n", p)).collect();

    format!(
        r#"# Red-Team Review: {plan_dir}

## Review Personas

### 1. Security Architect
Focus: Authentication, authorization, data protection, injection vectors, secrets management.
- Are there any unauthenticated endpoints?
- Is input validation comprehensive?
- Are secrets properly managed (no hardcoded values)?

### 2. Performance Engineer
Focus: Latency, throughput, resource usage, caching, database queries.
- Are there N+1 query patterns?
- Is caching strategy appropriate?
- Will this scale to 10x current load?

### 3. Reliability Engineer
Focus: Error handling, failure modes, recovery, observability, testing.
- What happens when external dependencies fail?
- Is error handling consistent and informative?
- Are there sufficient tests for edge cases?

### 4. DevOps Specialist
Focus: Deployment, CI/CD, infrastructure, monitoring, rollback.
- Can this be deployed with zero downtime?
- Are migrations reversible?
- Is observability sufficient for debugging in production?

## Plan Files to Review

{phase_list}
## Instructions

For each persona, review the plan and provide:
1. **Critical Issues** — Must fix before implementation
2. **Warnings** — Should address but not blocking
3. **Suggestions** — Nice-to-have improvements

Rate overall plan quality: 1-10 with justification.
"#
    )
}

/// Convert a kebab-case slug to Title Case.
fn to_title_case(slug: &str) -> String {
    slug.replace('-', " ")
        .split_whitespace()
        .map(|w| {
            let mut chars = w.chars();
            match chars.next() {
                None => String::new(),
                Some(c) => c.to_uppercase().to_string() + chars.as_str(),
            }
        })
        .collect::<Vec<_>>()
        .join(" ")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_render_plan_contains_title() {
        let out = render_plan("my-feature", "hard", "260317-0424-my-feature");
        assert!(out.contains("My Feature"));
        assert!(out.contains("status: pending"));
    }

    #[test]
    fn test_render_phase_contains_title() {
        let out = render_phase("Research", 3);
        assert!(out.contains("# Phase: Research"));
        assert!(out.contains("- [ ]"));
    }

    #[test]
    fn test_render_red_team_contains_personas() {
        let phases = vec!["phase-01-research.md".to_string()];
        let out = render_red_team("plans/my-plan", &phases);
        assert!(out.contains("Security Architect"));
        assert!(out.contains("phase-01-research.md"));
    }

    #[test]
    fn test_to_title_case() {
        assert_eq!(to_title_case("port-go-to-rust"), "Port Go To Rust");
        assert_eq!(to_title_case("single"), "Single");
    }
}
