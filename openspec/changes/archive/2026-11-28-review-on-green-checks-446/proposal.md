# Proposal: review-on-green-checks-446
Approved-by: Coby Leuschke (2026-11-28) sha256:e1ff4924a86a

## Track

`quick`: it adds one scenario to an existing harness Requirement and no Requirement, touches one capability, adds no exported surface, and is no boot path, durability or gate file. The edits are Markdown instructions and their Go template copies: `.agents/skills/coordinate/SKILL.md` step 3 (`internal/bootstrap/templates.go:417`) and `.agents/contracts/developer.md` (`internal/bootstrap/review.go:17`).

## Decisions you are making by approving

- **A developer hands off as `ready` only once every required check at its PR head has passed, apart from `Merge check`.** `Merge check` stays absent until the reviewer writes its `reviewed-by:` line, so it never counts as red or running here. While a check is still running the developer waits for it; when one fails it fixes the failure and pushes, or hands off as `blocked` naming the check. You stop paying for a reviewer run on a head CI has already failed.
- **The coordinator reads the required checks at the PR head before `ateam review request`, and messages the developer run instead of requesting review when one other than `Merge check` failed or is still running.** It reads them with `gh pr checks <pr> --required` (on Gitea, the head commit's statuses for the contexts branch protection requires). No reviewer run starts on that handoff; the next developer handoff triggers the check again.
- **Nothing enforces this in code.** `ateam review request` still starts a review on a red head when someone runs it by hand; making it refuse would edit `internal/review/`, a gate file, and is left out.

## Why

#446, from the v0.4.7 coordinator dogfood (note 18): the #415 developer handed off as `ready` with the required `Verify` job red, and coordinate step 3 requests review on any `ready` handoff, so a full reviewer run was spent on a failure CI had already caught. From #412 on the coordinator waited for green by its own judgment; nothing written asks for it.

## What Changes

1. Developer contract (and `developerContract`): one paragraph defining when a `ready` handoff may be given.
2. Coordinate skill step 3 (and its template): read the required checks at the head before `ateam review request`; message the developer run instead when one is red or running.
3. Harness spec: one scenario under *Optional coordinator role*, with a shipped-text test.

## Scope & Non-Goals

- **In-bounds:** the two Markdown files above, their copies in `internal/bootstrap/templates.go` and `internal/bootstrap/review.go`, `internal/bootstrap/coordinator_test.go`, the harness spec delta.
- **Non-goals:** a check in `ateam review request` (gate directory); the protocol's § 2 step 5 and the handoff skill, which a developer without a coordinator follows unchanged; coordinate step 5, which #445 / PR #483 edits; `Gate files` red on a gate-file PR, which the coordinator contract already treats as an owner gate that ends the run.

## Definition of Done

- A test asserts the new sentences in the shipped developer contract and coordinate skill, shown able to fail with a non-destructive mutation of each template; parity keeps the repo files equal to the templates.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule.
