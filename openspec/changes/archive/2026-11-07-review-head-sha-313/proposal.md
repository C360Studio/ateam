# Change Proposal: `review-head-sha-313`

Approved-by: Coby Leuschke (2026-11-07)

## Why

During the #303 live run, `ateam review request --dry-run --implementer claude:opus:high 312` from `main` printed `/review-pr 312 at 1c2a96cc…`, main's HEAD, not #312's head `014133ef…`. With `--implementer` the command took `head_sha` from the local checkout's `HEAD`, so any request made outside the PR's checkout names the wrong commit and the reviewer reviews it (#313).

## What Changes

- `ateam review request` always reads the PR's head from the forge; `--implementer` only replaces the implementer list read from the PR body.
- With `--implementer` and the PR unreadable, the command exits 2 naming the PR's head instead of printing a local or empty sha.
- `docs/adoption.md` states that the head comes from the PR.
- The `Review request spec` Requirement in `openspec/specs/model-review/spec.md` gains one scenario.

## Track

`quick`: one scenario on an existing Requirement in one capability, no new exported surface, no gate file.

## Non-goals

- No change to the launch manifests under `.agents/hosts/` (a gate file).
- No offline fallback to the local `HEAD`: a review needs the forge to post its record anyway.

## Definition of Done

- `TestReviewRequest` covers `--implementer` from a root that is not the PR's checkout (head is the PR's) and `--implementer` with the PR unreadable (exit 2, nothing printed or run); `task mutate:check` reports `detection` with the old local-HEAD code and with the refusal removed.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
