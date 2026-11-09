# Change Proposal: `awaiting-probe-359`

Approved-by: Coby Leuschke (2026-11-09) sha256:4a46b57f0691

## Why

The awaiting probe (#262, `internal/review/gate.go:315-357`) skips `Merge check` only while a code pull request has no `reviewed-by:` line **and no review record comment**. In the normal flow the record lands before the agent writes the line, and every push or description edit after an earlier record also lands in that state, so each reviewed pull request posts two or three red `Merge check` runs by design. Each one emails the owner. On 2026-11-09 five of twelve non-green runs were this window (#359).

The probe is narrow on purpose: a skipped job is renamed, so it never overwrites an earlier passing `Merge check` at the same head. "No line means skip" would let a line removed after an approval keep its pass.

## What Changes

- `ateam review gate --awaiting` answers `awaiting=true` for a code pull request with no `reviewed-by:` line **unless a successful `Merge check` already stands at the head**. A record no longer decides; a standing pass at the head does, because that pass is the only thing a skip could leave wrong (design D1).
- Each forge source reads the head's `Merge check` result: GitHub check runs by name, Gitea commit statuses by context (D2). A failed read leaves no answer, so `Merge check` runs in full, as today.
- The GitHub `awaiting` job gains `checks: read`; `.gitea/workflows/merge-check.yml` needs no new scope if the token already reads statuses (D2).
- `openspec/specs/harness/spec.md` Requirement *Merge check runs on pull-request state events* (`:820`): the awaiting and probe scenarios (`:844-854`) and the line-removal scenario (`:854-857`) state the new condition.

## Track

`design`: it touches gate files (`internal/review/`, `.github/workflows/`, `.gitea/workflows/`) and the probe's condition differs from the one #359 proposes. See `design.md`.

## Non-goals

- The `Gate files` `startup_failure` (#360), the strict up-to-date rule, merge queues, and the owner's Actions email setting.
- No change to the full merge check, the record format, or the `reviewed-by:` line rule.

## Definition of Done

- `TestGateAwaiting` cases on both forges: record at the head with no line and no standing pass is `awaiting=true`; record at an older head with no line is `awaiting=true`; a passing `Merge check` at the head with the line removed is `awaiting=false`; a failed or pending `Merge check` at the head with no line is `awaiting=true`; a failed check read exits non-zero with no `awaiting=` line.
- `task mutate:check` reports `detection` with the standing-pass test inverted and with the check name matched by prefix.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the owner's `gate-change-approved` label and comment at the final head; the change archived before ready.
