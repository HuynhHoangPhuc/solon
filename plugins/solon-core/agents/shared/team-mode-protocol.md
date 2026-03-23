# Team Mode Protocol

Standard behavior when an agent is spawned as a teammate within an Agent Team.

## On Start

1. Check `TaskList` then claim your assigned or next unblocked task via `TaskUpdate`
2. Read full task description via `TaskGet` before starting work
3. Follow the role constraint defined in your agent file (the "Role constraint:" line after the team-mode reference)

## During Work

- Respect file ownership boundaries stated in task description
- Only 1 task `in_progress` at a time
- Mark task complete IMMEDIATELY after finishing

## On Completion

4. `TaskUpdate(status: "completed")` then `SendMessage` summary/deliverables to lead

## On Shutdown Request

5. When receiving `shutdown_request`: approve via `SendMessage(type: "shutdown_response")` unless mid-critical-operation

## Coordination

6. Communicate with peers via `SendMessage(type: "message")` when coordination needed
