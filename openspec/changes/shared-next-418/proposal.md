# Proposal: shared-next-418
Approved-by: Coby Leuschke (2026-11-26) sha256:79a60d9e4537

## Track

`design`: adds a harness Requirement, a new `ateam state next` command adopters run, a new pickup-line field, and changes to the generated pickup and handoff skills.

## Decisions you are making by approving

- The milestone description is where you set the order of work: one line lists the issues in order, and another line names the milestone it comes after. You can reorder on the forge without a pull request.
- An issue waits while another open issue it names as a dependency, in its body or in a comment, is still open.
- Every session start shows the same "next" issue to every session. An issue already claimed by an open pull request is passed over, so two sessions asking at once are not handed the same work.
- "Next" stays inside the earliest unfinished milestone. When nothing there can be started, it says so and why, and does not reach into the following milestone.
- When the order cannot be read (no order line, two milestones that both claim to come first, a loop, or comments the last sync could not read), ateam names nothing and says why, rather than guessing.
- Issues with no milestone, or in a milestone but missing from its order line, are never named next. They are listed so you can place them.
- The pickup and handoff skills take "next" from ateam. A handoff stops carrying the order in its own words.

## Why

ateam has no definition of "next", so each session built its own from its own handoff (#418). During v0.4.5, two sessions worked one milestone from different handoffs. One of them recorded a claim that wasn't real, and a stale handoff named a finished milestone as current for two days. The order already exists as hand-written milestone prose, but nothing reads it.

## What Changes

1. `ateam state sync` caches each milestone's description and every open issue's `Depends on: #n` lines (from its body and comments). It also caches the issues each open PR names with `Closes`, `Fixes`, `Resolves` or `Addresses #n`.
2. `ateam state next` prints the next issue, the reason each earlier issue was skipped, and the issues no `Order:` line places. `ateam state status` JSON carries the same result.
3. The pickup line gains `next=#n`, or `next=none (<reason>)`.
4. The pickup skill's "Next authorized step" reads `ateam state next`, and the handoff Goal names the unit in hand, not an order. This applies to the repository copy and the generated templates.

## Scope & Non-Goals

- **In-bounds:** `internal/state` (sync payload, parser, cache, summary line), `cmd/ateam/main.go` (subcommand wiring), `openspec/specs/harness/spec.md` (one ADDED Requirement), `.agents/skills/{pickup,handoff}/SKILL.md` and their `internal/bootstrap` templates, `docs/daily-use.md`, and `Order:`/`After:` lines on this repository's milestones.
- **Non-goals:** the queue checks and doctor/preflight findings (#416); the coordinator, and judge placement of new issues (#417); `Blocked by` or other dependency grammars (#416); anything in `internal/review/` (a gate file). The tea-CLI fallback reports the order as unavailable.

## Definition of Done

- Table tests over cached cards cover each skip reason, tie, cycle, missing `Order:` line, unplaced issue, PR claim, and comments-unread case. A property test checks that `next` is unchanged when unrelated cards are shuffled. `task mutate:check` detects the removal of each rule.
- The session-start hook still reads only the cache.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before ready.
