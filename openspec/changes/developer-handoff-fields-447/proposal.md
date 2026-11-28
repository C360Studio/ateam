# Proposal: developer-handoff-fields-447
Approved-by:

## Track

`quick`: it adds one scenario to an existing harness Requirement and no Requirement, touches one capability, adds no exported surface, and is no boot path, durability or gate file. The edits are Markdown instructions and their Go template copies: `.agents/contracts/developer.md` (`internal/bootstrap/review.go:17`) and `.agents/skills/coordinate/SKILL.md` step 4 (`internal/bootstrap/templates.go:421`).

## Decisions you are making by approving

- **An action only you can take goes in the developer's handoff Goal, named as owed by the owner.** For example: "owed by the owner: delete scratch repository X". The coordinator's one owner message lists every such action. You stop finding owed deletions deep in a handoff's Constraints.
- **A developer's handoff names the change's track, and for `quick` gives the one-line reason no design trigger applies.** The coordinator repeats it beside the proposal digest in its owner message, so you rule on the track when you approve, not after.
- **A developer never writes "no gate file touched" as a conclusion.** It quotes the line `scripts/gate-files.sh <pr>` printed and the commit it ran at. Who posts gate-file approval is #453 (PR #485) and is not changed here.
- **A developer rewriting its PR body keeps any `owner-prompts:` line it finds, unchanged.** This is contract text, not a check. A check would have to compare the line with its earlier value, which only the forge's edit history holds, and it would live in `scripts/merge-check.sh` or `internal/review/`, both gate files and the design track. If a developer launched with this text still drops the line, a follow-up issue adds the check.

## Why

#447, from the v0.4.7 coordinator dogfood: #402's owed scratch-repository deletion sat in Constraints (note 22); #414 and #415 wrote "no gate file touched" under `implemented-by:` as if checked (note 9); #414 and #412 chose `quick` with no step showing you the choice (note 7); every developer that rewrote its PR body at ready-for-review dropped the coordinator's `owner-prompts:` line (notes 19, 41). It happened again in the v0.4.8 run: PR #483's developer dropped the line and the coordinator restored it. That developer ran before any text asked it to keep the line.

## What Changes

1. Developer contract (and `developerContract`): one paragraph with the four rules above.
2. Coordinate skill step 4 (and its template): the owner message lists owed actions and repeats each proposal's track.
3. Harness spec: one scenario under *Optional coordinator role*, with a shipped-text test.

## Scope & Non-Goals

- **In-bounds:** the two Markdown files above, their copies in `internal/bootstrap/review.go` and `internal/bootstrap/templates.go`, `internal/bootstrap/coordinator_test.go`, the harness spec delta.
- **Non-goals:** a check on the `owner-prompts:` line (above); a new handoff field (Goal and Constraints already exist); the handoff skill and protocol § 2, which a developer without a coordinator follows unchanged apart from the contract; gate-file approval wording (#453); coordinate steps 3 and 5 (#446 / PR #484, #445 / PR #483).

## Definition of Done

- A test asserts the new sentences in the shipped developer contract and coordinate skill, shown able to fail with a non-destructive mutation of each template; parity keeps the repo files equal to the templates.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule.
