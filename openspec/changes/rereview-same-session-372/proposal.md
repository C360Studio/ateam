# Change Proposal: `rereview-same-session-372`

Approved-by:

## Why

After a `Verdict: REQUEST CHANGES` record, the implementer fixes the findings and requests a fresh review (`docs/daily-use.md:84`). `ateam review request` always prints the host's launch action, and every host manifest's launch starts a new session. Each round therefore pays a new reviewer to re-read the whole PR, and the new reviewer cannot check that the earlier findings were the ones fixed. The owner ruled on 2026-11-11 (#372) that a re-review goes back to the reviewing session by default when the host can still reach it, and falls back to a fresh reviewer otherwise.

## What Changes

- `ateam review notify` adds the reviewer's own session to its fixed message: `…; to re-review after fixes run ateam review request --reviewer-session <id> <n>`. The session comes from the reviewer's environment on T3 Code and herdr, or from `--from` on Enjoy.
- `ateam review request --reviewer-session <id> <pr>` sends a fixed re-review message to that session through the host manifest's existing `[notify]` action, with no new manifest table. It does this only when the cached newest record is `REQUEST CHANGES` at a commit other than the PR head, and the record's reviewer run still passes the merge gate's comparison against the current implementers. Otherwise it prints today's fresh launch. `--fresh` always launches a new reviewer.
- The re-review message tells the reviewer to check out the new head, read the full base-to-head diff and the diff since the commit it reviewed, rerun `task verify`, post a new record, and notify back.
- The `review-pr` skill and `docs/daily-use.md` step 11 describe the re-review path and the fallback.

## Track

`design`. It adds CLI surface that adopting repositories call (`--reviewer-session`, `--fresh`, `--from`) adds *Re-review to the reviewing session*, and modifies *Review notify text*, both in `model-review`.

## Non-goals

- No change to the merge gate, the record format, or any gate file. The host manifests are reused as they are.
- No ateam-side record of launched reviewers (#327, HAAG) and no handoff field for the reviewer session.
- No fix for `notify` sending nothing on the `cli` host (#358). It is a v0.4.1 candidate in its own right.

## Definition of Done

- `cmd/ateam` tests cover: notify text carries the session; request re-routes on `REQUEST CHANGES` at an older sha; request falls back to a fresh launch for a reviewer the gate now rejects, an `APPROVE` record, a record at the head, no cached record, and `--fresh`; and a failed notify command falls back to the launch spec. `task mutate:check` gives `detection` on the verdict and gate-comparison branches.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before the PR is marked ready.
