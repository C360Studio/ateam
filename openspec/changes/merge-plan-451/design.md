# Design: merge-plan-451

Drafted by claude (opus, high) in the developer run, under the design skill. Read-only on the deliverable until approval; the owner rules. D4 records the owner's ruling on #453 (Q2 = A); the rest are open, each with a recommendation.

## Context

- `.agents/skills/coordinate/SKILL.md:18` (`internal/bootstrap/templates.go:413`): step 2 starts the next unit "once the draft PR claims the issue"; nothing orders units that edit the same files. The launch prompt says nothing about auto-merge.
- `.agents/skills/coordinate/SKILL.md:22` (`templates.go:417`): on `Verdict: APPROVE`, "start or message the developer run to close per protocol § 2 step 5". So every approved PR arms as soon as it is approved.
- `.agents/skills/coordinate/SKILL.md:26` (`templates.go:421`): step 4's owner message lists gates only; nothing reports a change of `ateam state next` or the milestone's `Order:` line.
- `.agents/skills/coordinate/SKILL.md:34` (`templates.go:429`): step 5's handoff lists "each unit in flight with its PR and the gate it waits on"; no order.
- `.agents/contracts/coordinator.md:5`: "Archiving, marking ready, arming auto-merge and removing the worktree belong to the developer run that claimed the work". The coordinator may message a developer run; it never arms.
- `.agents/contracts/developer.md:7`, `:9` (`internal/bootstrap/review.go:23`, `:25`): the coordinated-question and ready-for-review rules; nothing on when to arm.
- `.agents/protocol.md:45` (§ 2 step 5): "Once the latest review record approves, move the change to `openspec/changes/archive/…` … and only then arm auto-merge"; "the merge fires when the required check is green on an up-to-date branch".
- `internal/review/gate.go:801-815` (`gateCommitAllowed`): a commit after the record is allowed only when it is a merge whose subject matches `^merge\b.*\bmain\b` and touches none of the PR's changed files, or a docs-only non-merge commit. So a `main` merge that brings in a sibling's edit to a shared file voids the record; a rebase voids it always. `.agents/contracts/reviewer.md:28`: on Gitea every later commit needs a new record.
- `openspec/specs/harness/spec.md:1436-1454` (*Optional coordinator role*): three scenarios; PRs #486 (#447), #487 (#448) and #488 (#449) each carry a MODIFIED delta of this block.
- In flight: PR #487 (#448) adds the `holding` wake-up reason ("when the coordinator tells you to wait, for example before arming auto-merge"); PR #485 (#453) has the developer ask for gate-file approval last, "when the pull request is the next to merge", relying on this merge plan. Both edit the lines above.
- Field evidence, v0.4.8 run: PRs #483, #485, #486, #487 and #488 are in flight at once, and all but #483 edit `SKILL.md`, the developer contract and `templates.go`; their tasks already say "arm auto-merge when the coordinator says so".

## Goals / Non-Goals

**Goals:** a unit sharing a file with another is reviewed once, not once per sibling merge; only one approved PR is armed at a time; a gate-file PR's approved head survives to its merge; a scope change mid-run reaches the owner.
**Non-Goals:** any gate file (`internal/review/` keeps its clean-merge rule); a tool that computes the plan; changing what `ateam state next` returns; same-session re-review after an `APPROVE` (#452).

## Decisions

### D1. When overlapping units start

- **A (recommended): start in parallel as today; serialise only review and merge.** Proposals and implementation overlap, so the owner can approve several digests in one message. Cost: each later unit resolves its conflicts when it updates from `main`, before its review.
- **B: start a unit that will edit the same files only after the earlier one merges.** No conflicts, but the owner sees one proposal at a time and the run takes as long as the sum of its units.
- *Adopter seam:* nothing to configure. A coordinator that ignores the plan behaves as today.

### D2. The order

- **A (recommended): first ready, first merged, a gate-file unit last.** The order is the order units hand off `ready`; the coordinator records it in its handoff's Addresses as `Merge order: #a -> #b`. A unit waiting on the owner never blocks one behind it.
- **B: the milestone's `Order:` line.** Predictable, but a unit stuck at an owner gate blocks every unit after it.

### D3. When a unit sharing a file is reviewed

Two units share a file when `git diff --name-only main...<branch>` lists it for both. Read it at each `ready` handoff; the issue's *Where* section is only a guess before a branch has commits.
- **A (recommended): request review of a unit only once every earlier unit in the order that shares a file with it has merged and it has updated from `main`.** Until then the coordinator tells it to wait (`holding`, #448). Each unit is reviewed once, at a head that already holds every earlier unit it shares a file with.
- **B: review every unit in parallel, re-review after each update.** Faster wall-clock, but every sibling merge that touches a shared file voids the record (`gate.go:801`): up to n² review rounds, the cost #451 reports. Cheaper once #452 lands; revisit then.
- *Adopter seam:* a unit sharing no file with any other is reviewed at once, as today.

### D4. Who arms, and when

- **A (recommended): the developer arms only on the coordinator's message, and the coordinator releases one approved PR at a time.** On `Verdict: APPROVE` the developer archives, hands off `holding`, and waits. The coordinator releases the first approved unit in the order; after its merge (the developer's `closed`, #448) it tells the next to update from `main` with a merge, never a rebase, and arm once its required checks pass. A merge that touches none of its files keeps its record (`gate.go:801`); one that does needs a new record, and the developer hands off `ready` for it.
- **B: the developer arms on APPROVE, as today.** Auto-merge lands each PR whenever its checks pass, in no order, and each merge leaves every other armed PR out of date (note 24, 37).
- *Gate-file unit (ruled on #453, Q2 = A):* released only when no other approved unit waits, and while it waits on the owner's approval no other unit is released, so the head the owner approves is the head that merges.
- *Adopter seam:* a developer started without a coordinator arms per protocol § 2 step 5, unchanged.

### D5. Scope changes mid-run

- **A (recommended): before each owner message the coordinator compares `ateam state next` and the milestone's `Order:` line with its last handoff, and names any change in that message.** The run continues; the owner decides.
- **B: stop the run on any change.** Safe, but a filed follow-up issue would end every run.

### D6. Spec shape

- **A (recommended): one ADDED Requirement, *Coordinator merge plan*.** No overlap with the three MODIFIED deltas of *Optional coordinator role* in flight, so whichever archives first cannot drop this change's scenarios.
- **B: scenarios under *Optional coordinator role*.** One fewer heading, but a fourth MODIFIED delta of the same block, rebased after each sibling archives.

## Risks / Trade-offs

- On Gitea every later commit needs a new record (`reviewer.md:28`), so an update after approval always needs one; D3 keeps the reviewed head up to date only for units that share files.
- The plan is instruction text. A developer started from an older launch prompt arms on APPROVE; the coordinator cannot disarm it (`coordinator.md:5`) and asks the owner.
- D3 uses #448's `holding`; if #448 lands without it, the coordinator's wait message uses `blocked` and this change says so.

## Migration Plan

None. Adopters get the text on the next `ateam init` or contract reconcile. Units in flight when this lands keep their prompts; their tasks already say "arm auto-merge when the coordinator says so".

## Open Questions

None for the owner beyond approving D1–D6 (approving accepts the recommended options unless you name another).
