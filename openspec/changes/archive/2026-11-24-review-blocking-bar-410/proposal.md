# Proposal: review-blocking-bar-410
Approved-by: Coby Leuschke (2026-11-24) sha256:44eac0eb82eb

## Track

`design`: adds a `model-review` Requirement and changes the reviewer contract, the `review-pr` skill and the re-review message that `ateam init` ships to adopting repositories.

## Decisions you are making by approving

- A reviewer blocks a pull request only for a regression against the base branch, silent data or identifier corruption reachable from a supported input, a spec or contract violation it shows with a reproduction or traced path, a breach of a never-cut rule (security, trust-boundary validation, data-loss handling, tests on non-trivial logic), or missing required verification. Anything else still approves, and the record lists it as non-blocking for the author to file as a follow-up issue.
- A finding that an existing later check already catches is non-blocking, unless the spec makes that earlier refusal a promise and the reviewer says why it matters.
- From the second round on, a reviewer first confirms each earlier finding is closed, then reads only the commits since its last record. It reads the whole change again only when those commits touch files the earlier diff did not, or when someone asks.
- When two reviewers disagree, the latest record the merge check reads decides; the other verdict is quoted as dissent and the deciding reviewer answers each of its findings.
- A repository can still raise its bar through its own project rules; there is no flag for it in this change.

## Why

semsource#365 ran four REQUEST CHANGES rounds, each finding less likely than the last, ending in a contrived case an existing check already refused; the owner stopped it by feel (#410). The contract says what to find but not what blocks, and every round reread the whole growing diff.

## What Changes

1. Reviewer contract (repository copy and `ateam init` template): blocking bar, backstop credit, non-blocking list, dissent rule, delta scope for later rounds.
2. `review-pr` skill § 4 (both copies): the re-review paragraph confirms prior findings and reads the delta instead of the full diff.
3. `ateam review notify`'s re-review text sent by `ateam review request --reviewer-session` says the same.
4. One ADDED `model-review` Requirement.

## Scope & Non-Goals

- **In-bounds:** `.agents/contracts/reviewer.md`, `internal/bootstrap/review.go`, `.agents/skills/review-pr/SKILL.md`, `internal/bootstrap/templates.go`, `cmd/ateam/reviewlaunch.go` and its test, `openspec/specs/model-review/spec.md` via the delta.
- **Non-goals:** `--bar` / `--since` flags (owner ruling 2026-11-24); third-round escalation (#411); archive timing (#412); any change to `task merge:check` or other gate files; a machine-readable severity field.

## Definition of Done

- Bootstrap parity tests assert the new contract and skill sentences in both copies; the re-review text test asserts the new wording; `task mutate:check` detects each.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived before ready.
