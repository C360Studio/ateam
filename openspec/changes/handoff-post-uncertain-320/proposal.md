# Change Proposal: `handoff-post-uncertain-320`

Approved-by:

## Why

`ateam state handoff` posts its PR comment under a 5 s timeout (`internal/state/forge.go:885`, `:921`). A timeout after the request was sent returns an error. The CLI prints `forge PR comment skipped: …` and exits 0 (`cmd/ateam/main.go:390`, `:399`), and MCP says `PR comment: skipped (…)` (`internal/mcpserver/server.go:342`), even though the comment may exist. Every run posts a new comment, so retrying a timed-out handoff can leave two copies on the PR (#320, HAAG R4 and R2, finding F3).

## What Changes

- A post that times out is reported as `uncertain`, not `skipped`: CLI `forge PR comment uncertain: <err>; rerun ateam state handoff to post it once`, MCP `PR comment: uncertain (<err>)`. Other errors keep `skipped`. The exit code is unchanged (0); the local checkpoint is saved either way.
- Before posting, `PostPRHandoffComment` reads the PR's comments. If the newest handoff comment (`isHandoffComment`) has the same body, it posts nothing and reports `posted (already on PR)`. If that read fails, it posts as it does today.
- `docs/hook-flow.md` names the uncertain outcome and the retry rule. The `Durable shared state and five-field handoff` Requirement gains one scenario.

## Track

`design`: it changes the forge write's ack behavior (what a post reports, and whether a retry writes). See `design.md`.

## Non-goals

- No hidden marker or id in the comment body; the five-field body itself is the identity.
- No read-back after a timeout to settle the outcome; the next run's pre-post read settles it.
- No change to commit statuses (#302), which are posted only after a definite post.
- No exit-code change for skipped or uncertain posts.

## Definition of Done

- Tests on both forges: a post that times out reports `uncertain` (CLI and MCP text); a retry whose newest handoff comment has the same body posts nothing; a different body, or an older equal comment followed by a different one, posts; a failed comment read still posts. `task mutate:check` reports `detection` with the dedupe removed and with timeouts reported as skipped.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
