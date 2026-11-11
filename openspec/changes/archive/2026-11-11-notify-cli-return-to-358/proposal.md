# Change Proposal: `notify-cli-return-to-358`

Approved-by: Coby Leuschke (2026-11-11) sha256:430d57df01ee

## Why

`ateam review notify` returns 0 with no output when the host resolves to `cli` (`cmd/ateam/reviewlaunch.go:448`), even when `--return-to` names a session. Reviewers started from Enjoy run `notify` in shells that lack `ENJOY_MCP_ENDPOINT`, so the host resolves to `cli`. They take the silent success as delivery, and the implementer never hears back. This happened on C360Studio/semsource#252 (twice) and C360Studio/semengine#586. Since #372, the same silent no-op also drops the `--reviewer-session` line a re-review needs.

## What Changes

- On `cli` with a non-empty `return_to`, `notify` sends nothing and exits 1. It prints the existing `notify: host cli has no notify …; send "<message>" to "<target>"` line on stdout, so the message can still be delivered by hand. On stderr it names `ATEAM_HOST` and the host manifests present in `.agents/hosts/` (for example `ATEAM_HOST=enjoy`).
- On `cli` with no `return_to`, behaviour is unchanged: exit 0, nothing sent.
- In the `review-pr` skill (and its `ateam init` copy in `internal/bootstrap/templates.go`), the abbreviated re-review line gains `--return-to <id>`, matching what `notify` sends (Codex nit on #373).

## Track

`quick`. It modifies one Requirement (*Review notify text*) in one capability, adds no exported surface, and touches no gate file.

## Non-goals

- No change to host detection order and no new host marker. The `.agents/hosts/*.toml` launch text (gate files) is unchanged.
- No retry and no delivery receipt from the host.

## Definition of Done

- `TestReviewNotify` covers `cli` with `--return-to`: exit 1, the send line on stdout, `ATEAM_HOST` named on stderr. It also covers `cli` without one: exit 0, no output. `task mutate:check` gives `detection` for dropping the new branch.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before the PR is marked ready.
