# Verification Protocol

Shared review cycle and quality gate patterns for fix, ship, test, and review workflows.

## Review Cycle

### Autonomous Mode

```
cycle = 0
LOOP:
  1. Run code-reviewer → score, critical_count, warnings, suggestions

  2. IF score >= 9.5 AND critical_count == 0:
     → "Review [score]/10 - Auto-approved"
     → PROCEED

  3. ELSE IF critical_count > 0 AND cycle < 3:
     → "Auto-fixing [N] critical issues (cycle [cycle+1]/3)"
     → Fix critical issues → re-run tests
     → cycle++, GOTO LOOP

  4. ELSE IF cycle >= 3:
     → ESCALATE to user via AskUserQuestion
     → Options: "Fix manually" / "Approve anyway" / "Abort"

  5. ELSE (score < 9.5, no critical):
     → "Review [score]/10 - Approved with [N] warnings"
     → PROCEED (warnings logged, not blocking)
```

### Human-in-the-Loop Mode

```
ALWAYS:
  1. Run code-reviewer → score, critical_count, warnings, suggestions
  2. Display findings (Critical, Warnings, Suggestions)
  3. AskUserQuestion:
     IF critical: "Fix critical" / "Fix all" / "Approve anyway" / "Abort"
     ELSE: "Approve" / "Fix warnings" / "Abort"
  4. Fix → re-test → re-review (max 3 cycles)
     Approve → proceed
     Abort → stop workflow
```

### Quick Mode

Same as Autonomous but:
- Lower threshold: score >= 8.5 acceptable
- Only 1 auto-fix cycle before escalate
- Focus: correctness, security, no regressions

## Critical Issues (Always Block)

These findings always block approval regardless of mode:
- Security vulnerabilities (XSS, SQL injection, OWASP)
- Performance bottlenecks (O(n^2) when O(n) possible)
- Architectural violations
- Data loss risks
- Breaking changes without migration

## Ship Review Gate

After each phase, self-check:
- All files in ownership list modified as expected
- Compile/lint passes
- Success criteria from phase file met
- No files outside ownership list modified

If any criterion fails: re-invoke agent with fix instructions before marking complete.

## Severity Levels

| Level | Label | Action Required |
|-------|-------|-----------------|
| Critical | Must fix before merge |
| Warning | Should fix, explain if skipping |
| Suggestion | Nice to have, no action required |
