# Proposal: status-label-route-443
Approved-by: Coby Leuschke (2026-11-27) sha256:66acd21338b3

## Track

`design`: it widens what a judge may write on the forge (it may now remove a label), in the contracts `ateam init` ships to adopters. No code, CLI flag, gate file or spec Requirement changes; `ateam state next` and `queue-check` already do what the new route needs.

## Decisions you are making by approving

- **In hold-my-beer mode, an issue waiting on your decision goes to a judge instead of you.** The judge answers the question the issue asks, writes its answer on a line starting `Ruling:` in its comment, and removes the `status:needs-decision` label, so the issue becomes startable again. In default mode nothing changes: the coordinator still stops and asks you.
- **A blocked issue whose blockers have all closed goes to a judge too; any other blocked issue still waits for you.** The coordinator starts a judge only when queue health already reports that every issue the blocked one depends on is closed. The judge checks each one, quotes its state, and removes `status:blocked`. A blocker that is still open, or not named, stays your call and does not end the coordinator's run.
- **A judge may remove only those two labels, and never adds one.** `gate-change-approved` and `class:flake` can weaken a merge gate and stay yours, as do tags and releases. The coordinator still never touches labels, milestones or the order line in either mode.
- **A judge's answer reads as a ruling, never as yours.** It uses `Ruling:`, not `Owner ruling`, so queue health counts it as a recorded decision without claiming you made it, and flags the issue if the label was left on.
- **No spec Requirement changes.** The route is contract wording pinned by the init test; the harness spec stays untouched, which keeps this change clear of the four open PRs that edit it.

## Why

#443: in hold-my-beer mode the coordinator contract names neither status label under the judge's gates nor under the owner's, and the judge contract forbids the label write that would unpark the issue. Found reading the contracts at `27839c9` during the v0.4.7 dogfood (default mode, so the gap did not fire).

## What Changes

1. `.agents/contracts/coordinator.md` hold-my-beer paragraph: the two label routes above, and the blocked issue with an open or unnamed blocker staying with the owner.
2. `.agents/contracts/judge.md`: the label removal, the `Ruling:` line, the evidence for a closed blocker, and the never-add / owner-only labels.
3. `.agents/skills/coordinate/SKILL.md` § Hold-my-beer Mode: the issue and its queue-health finding in the evidence list.
4. `docs/model-review.md` § Hold-my-beer mode: the two labels in the judge's list and a *Labels* bullet.
5. The same text in `internal/bootstrap/templates.go`, and literals for it in `TestInitShipsJudgeContract`.

## Scope & Non-Goals

- **In-bounds:** the four files above, `internal/bootstrap/templates.go`, `internal/bootstrap/coordinator_test.go`.
- **Non-goals:** `openspec/specs/harness/spec.md`; any Go outside `internal/bootstrap`; letting the coordinator write labels; a judge-specific decision-line prefix; enabling `hold_my_beer` here; `docs/daily-use.md`, which already says gate files and releases stay the owner's.

## Definition of Done

- The repository files and what `ateam init` writes are byte-identical (existing check in `TestInitShipsJudgeContract`), and the new literals fail the test when removed (`task mutate:check` on `templates.go`).
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green.
