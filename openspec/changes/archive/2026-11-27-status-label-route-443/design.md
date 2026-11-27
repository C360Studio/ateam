# Design: status-label-route-443

Drafted by claude (opus, high) in the developer run, under the design skill. Read-only on the deliverable until approval; the owner rules.

## Context

- `.agents/contracts/coordinator.md:5`: the coordinator never changes labels, milestones or an `Order:` line. `:7`: default mode stops a unit at an issue labelled `status:needs-decision` or `status:blocked`. `:9`: the hold-my-beer paragraph hands a judge proposals, design questions, developer questions, a third rejection and milestone scope or order, and keeps gate files and releases for the owner; neither status label is named on either side.
- `.agents/contracts/judge.md:5`: "Stay read-only except for your record and, on a milestone question, the edit it rules." `:9`: never apply `gate-change-approved`, tag or release.
- `.agents/skills/coordinate/SKILL.md:30`: the evidence list (proposal and design paths, review records, the question comment, milestone and issue numbers).
- `internal/bootstrap/templates.go:372`, `:382`, `:386`, `:423`: the shipped copies of the lines above. `internal/bootstrap/coordinator_test.go:88-92` fails when a shipped file differs from the repository's; `:94-117` pins literals.
- `docs/model-review.md:157`: the judge's gate list in prose.
- `internal/state/next.go:187-190`: `ateam state next` skips an issue carrying either label, so a label left on keeps the unit parked (the issue's first open question).
- `internal/state/forge.go:935-948`: `findDecisionLine` accepts a line starting `Ruling:` with no date; only `Owner decision`/`Owner ruling` need one (#421). `internal/state/queuecheck.go:237-239`: Q3 reports a `status:needs-decision` issue with a recorded decision (the issue's second open question).
- `internal/state/queuecheck.go:227-235`: `unread`, Q2 (no `Depends on:` line) and Q1 (every `Depends on:` issue closed) for `status:blocked`. `openspec/specs/harness/spec.md:1375` *Queue health checks* and `:1451` *Recorded decision lines* specify both; neither changes.
- `openspec/changes/archive/2026-11-27-archive-after-approval-412/.openspec.yaml`: `skip_specs: true` is the precedent for a contract-only change with no delta.

## Goals / Non-Goals

**Goals:** under `hold_my_beer`, a status-labelled issue has one written route, and a judge's ruling actually unparks it.
**Non-Goals:** coordinator label writes, new Go behaviour, a spec Requirement, enabling the mode.

## Decisions

### D1. `status:needs-decision`

- **A (recommended): the judge rules it and removes the label.** One sentence in each contract. Blast radius: the judge gains one forge write, on the issue it was asked about.
- **B: the judge rules, the label stays for the owner.** No new write, but `next` keeps skipping the issue, so the ruling changes nothing until the owner acts: the gap #443 reports.
- **C: the coordinator removes it after the ruling.** Breaks `coordinator.md:5` (routes, never rules) and forecloses keeping the coordinator write-free.
- *Adopter seam:* only with `hold_my_beer` on. If you do nothing, default mode is unchanged.

### D2. `status:blocked`

- **A (recommended): a judge only on a Q1 finding.** The coordinator starts a judge when `queue-check` reports every `Depends on:` issue closed; the judge confirms each state (an issue closed, a PR merged), quotes it, and removes the label. Q2, `unread`, or an open blocker stay owner gates, and do not end the run: other units continue, as in default mode.
- **B: a judge for every blocked issue (the issue's suggestion).** A still-open blocker comes back "cannot decide", and `coordinator.md:9` ends the whole run on "cannot decide", so a plain fact would stop the milestone. Also spends a judge run to learn what Q1 already says.
- **C: blocked stays owner-only in both modes.** No change, but a closed blocker still costs an owner prompt in a mode meant to avoid them.
- *Adopter seam:* the judge removes a label only on evidence a command printed; a team that labels for external blockers (no `Depends on:` line) keeps them owner gates.

### D3. Labels a judge never touches

- **A (recommended): remove only the label it was asked about, never add one.** `gate-change-approved` and `class:flake` feed `Gate files` and `merge:check`; tags and releases stay the owner's (`judge.md:9`).
- **B: let it add `status:blocked` when it finds a blocker.** Unrequested; a new write with no caller.

### D4. The ruling line

- **A (recommended): `Ruling: <answer>`.** Read today by `findDecisionLine`; never reads as the owner's, mirroring `Judged-by:` vs `Approved-by:`. If the label removal fails, Q3 reports the issue.
- **B: a new `Judge ruling (<date>):` prefix.** Needs a `forge.go` change and a spec scenario for no new caller.

### D5. Spec

- **A (recommended): no delta (`skip_specs: true`).** The route is instruction text; `TestInitShipsJudgeContract` pins it. Avoids `openspec/specs/harness/spec.md`, which PRs #434, #435, #436 and #442 edit.
- **B: a scenario under *Optional coordinator role*.** Puts the route in the spec, but archiving rewrites that Requirement block and conflicts with the open PRs.

## Risks / Trade-offs

- A judge that misreads a blocker's state unparks an issue too early. Mitigation: it acts only on a Q1 finding and quotes each state; the coordinator then routes it like any other `next` issue, and the owner sees the comment.
- The label write is self-reported, like `Judged-by:`; nothing checks afterwards that a judge removed only its own label (ADR 0001).

## Migration Plan

None. Adopters get the new text on the next `ateam init` or contract reconcile; with `hold_my_beer` off it is inert.

## Open Questions

None for the owner beyond approving D1–D5. The issue's three questions are answered in Context (first two) and D2 (third).
