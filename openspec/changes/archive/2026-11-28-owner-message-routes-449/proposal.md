# Proposal: owner-message-routes-449
Approved-by: Coby Leuschke (2026-11-28) sha256:44168063a022

## Track

`quick`: it changes one scenario and adds one under an existing harness Requirement, touches one capability, adds no exported surface, and is no boot path, durability or gate file. The edits are Markdown instructions and their Go template copies: `.agents/skills/coordinate/SKILL.md` step 4 (`internal/bootstrap/templates.go:421`), `.agents/contracts/coordinator.md` (`templates.go:364`) and `.agents/contracts/developer.md` (`internal/bootstrap/review.go:17`).

## Decisions you are making by approving

- **The owner message keeps the route you have used for every approval this run, and adds the two others.** For each proposal it lists, in order: the developer's conversation link (the one the coordinator's launch tool returned), the PR, the digest, and three ways to approve: (1) answer "approve" to the coordinator, which passes your words unchanged to that developer run; (2) say it in the developer's conversation; (3) commit the `Approved-by:` edit yourself. This departs from the issue's suggested fix ("never offer to carry an approval"): in the v0.4.8 run you answered every approval with "Approve; have the developer make the edit for me", and the developer wrote it. If you want the issue's version instead, say so and route (1) is dropped.
- **A route that failed is not offered again for that unit.** When a developer run hands off as `blocked` because its host refused the edit (what happened in v0.4.7, notes 12 and 15), the coordinator's next message for that proposal offers only routes (2) and (3).
- **The developer writes `Approved-by:` only on your own words.** It fills the line with your name, the date and the digest `task spec:queue` prints, removes the hold in the same commit, and quotes your words and where they came from in the commit message. A coordinator's or judge's summary of your answer, or a "yes" to a different digest, is not an approval; it asks again. The coordinator still never writes the line. Protocol § 2 and `AGENTS.md` still say the owner fills it; an edit written on your own words counts as yours, as the board's **Approve** already does, so their wording is not changed.
- **Nothing is re-asked once answered.** Before each owner message the coordinator runs `ateam state sync` and, for every gate it would list, reads the branch (`Approved-by:` filled, hold gone) and the PR's comments since it last asked; a gate already answered is dropped and its developer run messaged to resume.
- **`owner-prompts:` counts a gate once.** A gate already asked and still unanswered, repeated in a later message, does not add 1; a message adds 1 only to the PRs it newly asks about.
- **Gate-file approval is not changed here.** Who posts `gate-change-approved` is #453 (PR #485), which counts no relayed yes; this change covers only `Approved-by:`.

## Why

#449, from the v0.4.7 coordinator dogfood: the offered relay was blocked by the host's permission layer and #402's run reverted its edit (notes 12, 15); the message named a PR where the approval had to happen in the developer's conversation (notes 33, 43); approvals already given were asked again because an `Approved-by:` commit signals nothing (notes 6, 8, 10); and committing from a phone is hard (note 5). In the v0.4.8 run the relay worked for every approval, so the fix keeps it and adds the fallbacks.

## What Changes

1. Coordinate step 4 (and its template): the per-gate order and three routes, the read-before-asking rule, and the counting rule.
2. Coordinator contract (and its template): the counting rule in its `owner-prompts:` sentence.
3. Developer contract (and `developerContract`): writing `Approved-by:` on the owner's own words, and `blocked` when the host refuses the edit.
4. Harness spec: the counting scenario changed and one new scenario, with a shipped-text test.

## Scope & Non-Goals

- **In-bounds:** the three Markdown files above, their copies in `internal/bootstrap/templates.go` and `internal/bootstrap/review.go`, `internal/bootstrap/coordinator_test.go`, the harness spec delta.
- **Non-goals:** a new wake-up reason for an approval (#448 / PR #487 owns the reason list; reading before asking covers note 10); protocol § 2, `AGENTS.md` and the rules index; gate-file approval (#453); any code that reads `owner-prompts:`; the board's **Approve**, which needs `ateam serve` reachable and is not offered from a phone.

## Definition of Done

- A test asserts the new sentences in the shipped coordinate skill and both contracts, shown able to fail with a non-destructive mutation of each template; parity keeps the repo files equal to the templates.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule.
