# Proposal: gate-approval-delegation-453
Approved-by: Coby Leuschke (2026-11-28) sha256:ff7c9a41ec68

## Track

`design`: it widens what a developer run may write on the forge (the `gate-change-approved` comment and label), in the contracts `ateam init` ships to adopters. No code, CLI flag, gate file or spec Requirement changes; `scripts/gate-files.sh` already checks what the new route produces.

## Decisions you are making by approving

- **You approve a gate-file change by saying yes in the developer's conversation, naming the PR and its head commit; that developer run then posts the approval for you.** It posts the approval comment, then applies the label, so the check reruns. You no longer run the commands yourself. This records your ruling on #453 (Q1 = A).
- **Only your own yes counts.** A yes that names no commit, a commit that is no longer the PR's head, or a message passed on by the coordinator or a judge is not an approval; the developer asks again instead of posting. The coordinator still never posts it, and its message to you links the developer's conversation.
- **If the head moves after your yes, the developer posts nothing and asks again for the new head.** A merge from `main` moves the head too, so the approval still names one exact commit (Q2 = A).
- **You are asked for the approval last.** The developer asks only after the approving review, the archive commit and an update from `main`, when the gate-file PR is the next to merge; the merge plan in #451 puts it last.
- **No spec, script or workflow changes.** The check already reads a comment and a label at the live head; it cannot tell who posted them under a shared login, as today.

## Why

#453, from the v0.4.7 coordinator dogfood (notes 25–27, 42). Protocol § 4 *Gate files* (`.agents/protocol.md:94`, `:101`) says only the owner posts the approval, but in practice the owner said yes for a named head and an agent posted it, and three times the owner asked the coordinator, the one role that may not. The owner ruled Q1 = A and Q2 = A on 2026-11-28 ([comment](https://github.com/C360Studio/ateam/issues/453#issuecomment-7240879118)).

## What Changes

1. `.agents/protocol.md` § 4 *Gate files* and *Unattended and fan-out runs*: the delegated post, what counts as the owner's yes, comment before label, and asking last.
2. `.agents/contracts/developer.md`: the steps the claiming run takes, and never posting on a relayed or stale yes.
3. `.agents/contracts/coordinator.md` and `.agents/skills/coordinate/SKILL.md` step 4: the owner message names the head and links the developer's session; the coordinator still never posts.
4. `AGENTS.md`, `.agents/rules-index.md` (the `gate-files+review` row), `docs/adoption.md` (*Who decides*), `docs/adr/0003-roles-and-routes.md` (the developer's *May write*).
5. The same text in `internal/bootstrap/review.go` and `templates.go`, with literals pinned in the init tests.

## Scope & Non-Goals

- **In-bounds:** the files above, `internal/bootstrap/coordinator_test.go`, `internal/bootstrap/review_test.go`.
- **Non-goals:** any gate file (`scripts/gate-files.sh`, the workflows, `.agents/hosts/`, `.agents/review-policy.json`, `internal/review/`); accepting a later head after a clean `main` merge (Q2 = B); `openspec/specs/`; a coordinator or judge write.
- **Out of scope, not ruled:** the whole-review finding that `scripts/gate-files.sh:126` never checks who wrote the approval comment. These rulings do not cover it; it stays on #453's thread for a separate decision and a gate-file change.

## Definition of Done

- The repository files and what `ateam init` writes stay byte-identical (`TestInitShipsCoordinatorAsCanonical`, `TestCanonicalReviewContracts`), and each new literal fails its test when removed (`task mutate:check` on the template file).
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green.
