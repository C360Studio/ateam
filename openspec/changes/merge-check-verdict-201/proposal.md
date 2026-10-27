# Change Proposal: `merge-check-verdict-201`

Approved-by:

## Why

`ateam review gate` (behind `task merge:check` and the `Merge check` job) accepts a review record by its reviewer, strength, and designated commit, but never reads its `Verdict:` line. A `Verdict: REQUEST CHANGES` record with valid attribution makes the check green and can release an armed auto-merge; Codex reproduced it on PR #194 with the fake-GitHub fixture (exit 0, `merge-check: ok`). #194 only worked around it in the review-pr skill. A second gap, from the owner's note on #201: `gateIsRecord` and `gateDesignated` scan fenced code blocks, so a record quoting the reviewer contract's template or an old `Reviewed at` line inside a fence can false-match or fail the single-commit rule. `aiprecord.go` already strips fences (`unfencedLines`).

## What Changes

- `gateIsRecord` and `gateDesignated` read only lines outside ``` and ~~~ fences, reusing `unfencedLines` from `internal/review/aiprecord.go`.
- The latest record (today's rule in `record`: the last comment that is a record and designates one commit) must carry exactly one unfenced line `Verdict: APPROVE` (case-insensitive, surrounding spaces allowed). Otherwise the gate adds a blocking finding naming the verdict it found (`REQUEST CHANGES`, missing, or more than one) and the fix "post a new review record with Verdict: APPROVE". A later APPROVE record still supersedes an earlier REQUEST CHANGES one.
- A rejecting record is a finding, not "no record", so the message says why the PR is blocked instead of "no review record".
- Docs follow: `.agents/skills/review-pr/SKILL.md` § 5 (and its byte-identical mirror in `internal/bootstrap/templates.go`) drops "the gate does not read the verdict"; `.agents/contracts/reviewer.md` § Record says the gate requires `Verdict: APPROVE` and ignores fenced text.
- One requirement with scenarios in the harness spec.

## Non-goals

- Reading findings or severities inside the record; the verdict line is the reviewer's summary of them.
- Changing which comment is the latest record, the strength rule, or the commit-freshness rule.
- External AIP records (`aiprecord.go`, #130): they already check their own `Verdict` field.

## Open question for the owner

A record with no `Verdict:` line: this proposal blocks it (every record on PRs #184–#203 carries one, and the contract's template requires it). The alternative is to block only an explicit non-APPROVE verdict.

## Definition of Done

- `task verify` green on a clean tree; `ateam doctor` parity green.
- Tests in `internal/review/gate_test.go` on the fake-forge fixture, both forges: REQUEST CHANGES blocks; APPROVE passes; REQUEST CHANGES then a later APPROVE passes; APPROVE then a later REQUEST CHANGES blocks; missing and duplicate verdicts block; a fenced `reviewed-by:`, `Reviewed at`, or `Verdict:` line is ignored. Each shown able to fail by `task mutate:check`.
- `internal/review/` is a gate file: the owner's `gate-change-approved` label before merge.
