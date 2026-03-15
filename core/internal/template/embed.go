// Package template provides embedded Go templates for plan scaffolding.
package template

import (
	"fmt"
	"strings"
)

// RenderPlan generates a plan.md from the embedded template.
func RenderPlan(slug, mode, dirName string) string {
	title := strings.ReplaceAll(slug, "-", " ")
	// Title case
	words := strings.Fields(title)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	title = strings.Join(words, " ")

	return fmt.Sprintf(`---
status: pending
priority: P1
effort: TBD
branch: TBD
tags: []
---

# %s

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
`, title)
}

// RenderPhase generates a phase-XX-*.md from the embedded template.
func RenderPhase(title string, totalPhases int) string {
	return fmt.Sprintf(`# Phase: %s

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
`, title)
}

// RenderRedTeam generates a red-team review prompt for a plan.
func RenderRedTeam(planDir string, phases []string) string {
	phaseList := ""
	for _, p := range phases {
		phaseList += fmt.Sprintf("- %s\n", p)
	}

	return fmt.Sprintf(`# Red-Team Review: %s

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

%s
## Instructions

For each persona, review the plan and provide:
1. **Critical Issues** — Must fix before implementation
2. **Warnings** — Should address but not blocking
3. **Suggestions** — Nice-to-have improvements

Rate overall plan quality: 1-10 with justification.
`, planDir, phaseList)
}
