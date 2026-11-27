# Proposal: third-rejection-stop-411
Approved-by:

## Track

`design`: adds a model-review Requirement and changes what `ateam review request`, a command adopting repositories run, does after review rounds. It touches no gate file: the count lives in the sync cache (`internal/state`) and the refusal in `cmd/ateam`, not in `internal/review/`.

## Decisions you are making by approving

- **D1. After a pull request's third rejecting review, asking for another review is refused until you decide.** The command that starts or re-sends a review stops with a short round summary instead, both for a fresh reviewer and for sending the round back to the same reviewer. It reads the review history the last sync saw, so it needs no new forge call and no gate-file change.
- **D2. You unlock the next round by posting one line on the pull request,** naming how many rejecting reviews you have seen (for example after the third, that number is 3). Each further rejection needs a new line, so a fourth rejection stops again. In hold-my-beer mode the judge writes that line in its ruling; nobody else may.
- **D3. The summary carries facts, not severity.** Per round: the reviewed commit, the reviewer run and a link to its record, plus the first and current commit. Whether each finding blocks stays with you or the judge, reading the linked records; the record grammar does not change. (Owner question: the issue asked for severity per round.)
- **D4. No label is applied.** The refusal and the coordinator's existing stop are the signal; `status:needs-decision` stays an issue label, and the coordinator still never changes labels. (Owner question: the issue proposed labelling the pull request.)

## Why

On semsource#365 four REQUEST CHANGES rounds ran before the owner stepped in by feel (#410). The coordinator contract already stops a unit at the third rejection (`.agents/contracts/coordinator.md:7`) and hold-my-beer sends it to a judge (`:9`), but only by the coordinator counting in prose; a developer running reviews by hand has no stop at all.

## What Changes

1. Sync caches, per open PR with a local worktree, the rejecting records it read (sha, reviewer run) and the newest owner continue line.
2. `ateam review request` refuses, exit 2, before any launch or same-session send, when the rejecting count is 3 or more and no continue line names that count; stderr names the line to post and lists the rounds.
3. Coordinator contract, coordinate skill and judge contract say who writes the continue line; templates follow.
4. ADDED Requirement under `model-review`.

## Scope & Non-Goals

- **In-bounds:** `internal/state` (cache), `cmd/ateam/reviewlaunch.go`, `.agents/contracts/{coordinator,judge}.md`, `.agents/skills/coordinate/SKILL.md`, `internal/bootstrap/templates.go`, the change's spec delta, tests.
- **Non-goals:** the merge gate and every gate file; a severity field in the record grammar; posting comments or labels from `ateam`; a workflow; protocol § 2 step 5 (PR #437 rewrites it).

## Definition of Done

- Tests: refusal at 3, pass at 2, pass at 3 with `continue 3`, refusal at 4 with only `continue 3`, refusal on the `--reviewer-session` path, fenced continue lines ignored; each proven by `task mutate:check`.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule live at merge.
