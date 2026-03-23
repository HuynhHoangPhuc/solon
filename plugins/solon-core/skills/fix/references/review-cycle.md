# Review Cycle

Load `../../references/shared/verification-protocol.md` for review cycle logic (Autonomous, Human-in-the-Loop, Quick mode thresholds, critical issues list).

## Fix-Specific Display Format

When presenting findings in Human-in-the-Loop mode, use this display:

```
┌─────────────────────────────────────┐
│ Review: [score]/10                  │
├─────────────────────────────────────┤
│ Critical ([N]): [list]              │
│ Warnings ([N]): [list]              │
│ Suggestions ([N]): [list]           │
└─────────────────────────────────────┘
```
