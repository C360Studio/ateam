# Change Proposal: `review-launch-295`

Approved-by:

## Why

With `cross_provider: true` the merge gate needs a review record from a different agent, but no repository file says how the implementing session obtains one. The knowledge lives in a handoff constraint and session memory ("Codex reviews via Enjoy child conversations"), which means nothing in a terminal, in a Codex or Gemini session, in T3 Code, in herdr, or in any other repository. Team members and adopters run four hosts, and each starts and messages an agent differently (#295).

## What Changes

- `.agents/review-policy.json` accepts `review_launch`: `manual` (default when absent) or `agent`. `manual`: the implementing session stops at ready-for-review and hands off; a person or a scheduled workflow starts the reviewer. `agent`: the implementing session may start the reviewer through its host. `cross_provider` keeps its meaning.
- `ateam review request <pr>` prints one JSON object naming the eligible reviewer runs (every other-provider model not weaker than each implementer by tier, with the band), the prompt `/review-pr <n>`, the head sha the record must bind to, and `return_to` (host and session id). Under `agent` with a host manifest it also runs the manifest's launch command; under `manual` it never runs anything.
- An optional host manifest `.agents/hosts/<host>.json` declares `launch` and `notify` as argument arrays with placeholders. Host detection reads `ATEAM_HOST`, then `T3CODE_THREAD_ID`, `HERDR_ENV`, `ENJOY_MCP_ENDPOINT`. No manifest means print only.
- `ateam review notify` sends fixed text ("review record for PR #n landed at <sha>; run ateam state pickup") to `return_to` through the manifest's `notify` command, or prints which host tool to call. Never the findings.
- `ateam state sync` fetches the comments of open PRs that back a local worktree, parses the review record grammar once in `internal/review`, and keeps sha, verdict, agent, model and effort on the PR card. Pickup prints `review record at <sha>, <verdict>, <agent> (<model>, <effort>)` on the worktree line, with `stale` when the head moved past the sha.
- Protocol § Close, the review-pr and handoff skills, `docs/adoption.md`, and ADR 0002 describe the manual and agent paths in host-neutral terms. The handoff constraint line is dropped at the next handoff.

## Track

`design`: a new policy field, two verbs reachable from an adopting repository, a file ateam executes, and two capabilities under `openspec/specs/`. See `design.md`.

## Non-goals

- No Go interface per host and no host switch in the binary; hosts are data.
- No network call from hooks or `ateam state pickup`; only `sync` fetches.
- No new MCP tool. No per-host override of `review_launch`. No change to the record grammar or to `ateam review compare`.
- T3 Code worktree placement and the `t3/` branch prefix are #296.

## Definition of Done

- Tests with attached spec references for: `review_launch` absent, `manual`, `agent`, invalid; `review request` output for each `cross_provider` and `review_launch` pair, including the eligible-runs list when the implementer is frontier; manifest placeholder expansion with no shell, a missing manifest, and a malformed one; host detection order and `ATEAM_HOST`; `notify` text under a hostile record body; the pickup line present, absent, and stale.
- `task mutate:check` reports `detection` on the host-detection order, the placeholder expander, the eligible-runs filter, and the stale comparison.
- `ateam doctor` hook latency stays inside the 10ms target; `sync` comment fetches stay inside the existing comment budget.
- Enjoy and herdr manifests ship in `.agents/hosts/` as reference fixtures; `docs/adoption.md` gains a hosts section naming no adopter.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready.
