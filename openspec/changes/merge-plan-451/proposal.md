# Proposal: merge-plan-451
Approved-by:

## Track

`design`: it adds a harness Requirement and changes when a developer run may arm auto-merge, in the contract and skill `ateam init` ships to adopters. No code, CLI flag or gate file changes. Detail and options: `design.md`.

## Decisions you are making by approving

Approving accepts the recommended option of each decision in `design.md` unless you name another.

- **Units still start in parallel; only their review and merge take turns (D1).** You can approve several proposals in one message, as in this run. A later unit fixes its conflicts when it catches up with `main`, before its review.
- **The merge order is the order units become ready for review, with a gate-file change last (D2).** The coordinator writes the order in its handoff. A unit waiting on you does not hold up the units behind it.
- **A unit that edits a file an earlier unit also edits is reviewed only after that unit has merged and it has caught up with `main` (D3).** Until then it waits, and the coordinator tells it so. Each such unit is reviewed once instead of once per sibling merge. Units sharing no file are reviewed at once, as today.
- **A developer arms auto-merge only when the coordinator tells it to, and the coordinator releases one approved pull request at a time (D4).** After each merge it tells the next one to catch up with `main` by a merge, never a rebase, and arm. If catching up touched one of its files, it goes back for a fresh review first. A developer started without a coordinator arms as today.
- **A gate-file change goes last and holds the line (D4, your ruling on #453).** It is released only when nothing else approved is waiting, and while it waits for your approval nothing else merges, so the commit you approve is the one that lands.
- **When `ateam state next` or the milestone's order changes mid-run, the next message to you says so (D5).** The run continues; you decide what to do about it.
- **The rules go in a new spec Requirement, not under the coordinator one three other open pull requests edit (D6).**

## Why

#451, from the v0.4.7 coordinator dogfood (notes 24, 28, 32, 36–38): five PRs added Requirements to the same spec file, each merge invalidated the next PR's review, auto-merge landed PRs in no order, the gate-file PR lost every race for its approved head, and a mid-run scope change was noticed by accident. The v0.4.8 run repeats it: PRs #483 and #485–#488 are in flight together and four edit the same skill, contract and template.

## What Changes

1. Coordinate skill steps 2–5 (and `templates.go`): the launch prompt's arming sentence; the order, the shared-file check and the release; the scope-change line; `Merge order:` in the handoff.
2. Developer contract (and `developerContract` in `review.go`): arm only on the coordinator's message, catch up by a merge, and hand off `ready` again if the merge needs a new record.
3. Protocol § 2 step 5: one sentence pointing a coordinated run to the coordinator's message.
4. Harness spec: ADDED *Coordinator merge plan*, with a shipped-text test.

## Scope & Non-Goals

- **In-bounds:** the files above, `internal/bootstrap/coordinator_test.go`, the harness spec delta.
- **Non-goals:** any gate file, including the clean-merge rule in `internal/review/gate.go`; a command that computes the plan; the coordinator contract, which already leaves arming to the developer; same-session re-review after an approval (#452); gate-file approval wording (#453, PR #485); Gitea's every-commit re-review rule.

## Definition of Done

- A test asserts the new sentences in the shipped coordinate skill and developer contract, shown able to fail with a non-destructive mutation of each template; parity keeps the repository files equal to the templates.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule.
