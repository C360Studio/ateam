# Design: `review-gate-wording-325-322`

## Context

Opened on `claude/review-gate-g1-e3` at `main` 7d21e10.

- `internal/review/gate.go:712-716` `gateIsDoc`: `.md`, or `openspec/` with `.yml`/`.yaml`/`.json`, except under `.claude/agents/` and `.agents/contracts/`. It decides whether a PR is documents only, and which commits after a record count as content (`gateCodeNames`).
- `scripts/gate-files.sh:19`: the gate-file pattern, a separate list that requires owner approval. It does not include `.agents/contracts/` either.
- `internal/state/commitstatus.go:154`: the record status description. `cleanDescription` cuts to `maxStatusDescription = 140` bytes (`:34`, `:196`).
- `internal/review/gate.go:395`: the one passing summary line. `internal/bootstrap/mergecheck_test.go` matches its prefix `reviewing agent: <agent> (record at <sha>)`, which stays unchanged.
- `internal/bootstrap/templates.go:631,633`: the `ateam init` merge-check rows. `reviewrecord.go:90-91` swaps them for the AIP rows by exact match, so the template and the checked-in rows change together.

## Decisions

**D1. Where the review-pr skill sits.**
- A) **Recommended:** code in `gateIsDoc`, the same as the contract (the issue's Done-when). A change then needs a not-weaker reviewer's record.
- B) Also add it to the gate-file pattern. The owner would approve every reviewer-instruction edit. That is more friction than the contract gets today. Not taken, and noted as a non-goal.

**D2. Where "self-reported" goes in the status.** Before the reviewer (`APPROVE self-reported by codex (…) at abc1234`), not as a suffix. A long model name could push a suffix past the 140-byte cut.

**D3. Summary wording.** Append after `(record at <sha>)` so that existing prefix matches hold and the line still names the limit.

## Risks / Trade-offs

- An adopter's PR that edits only its own `review-pr` skill now needs a review record. This is intended. It shows in `ateam review gate` as `code file: .agents/skills/review-pr/SKILL.md`.
- Anything that parses the status description sees new text. Nothing in this repository parses it, because no gate reads statuses (`docs/adoption.md` § Informational commit statuses).

## Migration Plan

None. The gate binary is built from the base commit, so this PR is judged by the old rule.

## Open Questions

1. **Owner:** D1 option A (review record) or B (owner approval too)?
