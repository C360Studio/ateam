# Change Proposal: `spec-approval-142`

Approved-by:

## Why

Non-trivial work today goes from claim to code with no recorded owner approval of what is being built, and proposals grow into design documents nobody reads before the code lands (#142, epic #180). A change needs a visible, parseable approval and a hold that keeps an agent from running past it.

## What Changes

- `proposal.md` carries an `Approved-by:` line, empty at creation; the owner fills it as `<name> (<YYYY-MM-DD>)`. `ScanOpenSpecQueue` exposes the value as `approved_by` (`internal/state/openspec.go`).
- The change's first task carries `Hold: OPEN-Q proposal awaits owner approval` from creation; the owner removes it in the same edit that fills `Approved-by:`. The existing hold parsing keeps such a change open in `task spec:queue`.
- One-page rule: `proposal.md` stays at or under 60 lines (why, what changes, scope/non-goals, Definition of Done); detail goes in `tasks.md` or `design.md`. `task spec:check` warns, without failing, past 60.
- `.agents/protocol.md` § 2 Clarify: the default Stop boundary for non-trivial work is "pause for proposal approval"; an agent with that boundary stops at the hold and hands off. § 2 Close: the hold is gone before the archive move.
- One `review` row in `AGENTS.md` § 3 and `.agents/rules-index.md`, mirrored in the `ateam init` templates; harness spec scenarios for approval and the length warning.

## Non-goals

- The UI read view and Approve action (#181).
- The merge-check note for an unfilled `Approved-by:` (#182; `scripts/merge-check.sh` is a gate file).
- Validating the `<name> (<YYYY-MM-DD>)` shape or who filled it: the hold, removed by the owner, is the gate.

## Definition of Done

- `task verify` green on a clean tree; `ateam doctor` parity green.
- Tests: `TestProposalApprovedByHeader`, `TestOpenQApprovalHoldKeepsChangeOpen`, `TestProposalLengthWarning` in `internal/state/openspec_test.go`, each shown able to fail by `task mutate:check`.
- This proposal is itself approved by the owner through the header and hold it introduces before the change is archived.
