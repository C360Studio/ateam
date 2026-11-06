# Design: `review-launch-295`

## Context

Opened in this session at `main` 6b5c323 (after #293). Host facts are from the 2026-11-06 reads recorded in #295 (T3 Code main @ 6577c6b6, herdr TUI 0.9.5).

- `internal/review/policy.go:18` `CrossProvider bool`; `:26` `ReviewRecord string`, validated by the switch at `:120-123`. That switch is the pattern for one more enum field.
- `internal/review/compare.go:28` `Compare` ranks by `tier` (`:118`) then `band` (`:136`); it judges a pair and never lists candidates.
- `internal/review/gate.go:293` and `:421-441` parse `reviewed-by:` and `Reviewed at` from PR comments inside the gate run, bound to a forge fetch. `internal/review/record.go:67` `ParseRecord` parses the AIP JSON record, not the comment grammar. No reusable comment-grammar parser exists.
- `cmd/ateam/review.go:21` `runReview` dispatches `compare`, `gate`, `setup`, `route`, `check`; the usage line is `:23`.
- `internal/state/forge.go:547` `prFields` fetches no PR comments; `:553` fetches issue comments under `ghCommentBudget` (`:501`); `:316` is the Gitea per-issue comments URL, which serves PRs too. `:222` `CommentBodies` is `json:"-"`, so comment text never persists; `:655-676` builds `BoardCard`, and `Decision` (`state.go:582`) is the precedent for a comment-derived field that does persist.
- `internal/state/state.go:2124` `worktreePRStatusAndAge` is where `open PR #n` and `ahead of merged` labels come from; `:2998` `writePickupWorktrees` prints them; `:2763` `BuildPickupSummary` is shared by hook, CLI, MCP, and web. `:1329` `ResolveEnvSessionID` reads the session env list.
- `internal/hook/hook.go:142` session-start and `:158` before-prompt render the summary under the 10ms budget with no network.
- `cmd/ateam/herdr.go:33` gates herdr calls on `HERDR_ENV=1`; `.agents/herdr/herdr-plugin.toml` declares commands as argument arrays. `internal/bootstrap/mcp.go:11` already imports `go-toml/v2`. `internal/state/tasklease.go:47` fixes error text so file content never enters agent context.
- `internal/mcpserver/server.go:53`, `:77`: two tools, the budget this change does not touch.
- Prose anchors: `.agents/protocol.md:45` Close; `.agents/skills/review-pr/SKILL.md:24` § 4 Post the Record; `docs/adoption.md:119` herdr table; `docs/adr/0002-model-tiers-and-effort-bands.md:131` addendum; `openspec/specs/model-review/spec.md:193` review record mode is the enum-field requirement pattern; `harness/spec.md:870` task lease is the optional-file pattern.

## Goals / Non-Goals

Goals: the portable part of a cross-agent review (who, which model, which effort, which prompt, which sha, where to report) lives in ateam; the host-specific part is one data file; the dev session learns the record landed on its next prompt. Non-goals are in `proposal.md`.

## Decisions

**D1. A separate key, not a richer `cross_provider`.**
- A) **Recommended:** `review_launch` with `manual` (default) and `agent`, validated like `review_record` (`policy.go:120`). Cost: one field, one switch, one test table. Adopter seam: a repository that never adds the key behaves as today, except that the skills now say to stop at ready-for-review.
- B) Make `cross_provider` a string (`false`, `true`, `agent`). Forecloses nothing but breaks every adopter policy that parses it as a bool, including this repository's. Rejected.

**D2. `ateam review request <pr>` prints a spec; it executes only under `agent` with a manifest.**
- A) **Recommended:** output is one JSON object: `pr`, `head_sha`, `prompt` (`/review-pr <n>`), `implementers`, `eligible` (every other-provider model whose tier is not below each implementer's, with the highest implementer band), `host`, `return_to`, and `launch` (`manual`, `printed`, or `executed`). Implementers come from `--implementer` flags, else from the PR body's `implemented-by:` line through the gate's forge client; offline with no flags exits 2. Exec happens only when `review_launch` is `agent` and the manifest has `launch`; the command is an argument array run without a shell, placeholders substituted per argument. Cost: about 150 lines plus a candidate filter beside `Compare`. Adopter seam: a developer who never reads this gets a JSON object and a skill that tells them what to do with it; nothing runs behind their back.
- B) Always execute when a manifest exists. Forecloses `manual`. Rejected.
- C) Print only, never exec. Costs every CLI host an extra copy-paste step and makes the manifest decorative. Rejected.

**D3. Manifest format and location.**
- A) **Recommended:** `.agents/hosts/<host>.toml`, parsed with the `go-toml/v2` already in `go.mod`, in the herdr-plugin shape: `[launch] command = [...]`, `[notify] command = [...]`. Placeholders `{pr}`, `{prompt}`, `{sha}`, `{agent}`, `{model}`, `{effort}`, `{return_to}`; `{agent}`, `{model}`, `{effort}` take the first eligible run unless `--reviewer` picks one. Cost: one parser and one expander. Blast radius: the file is in the repository, so a PR can change what `agent` mode runs; it is listed with the gate files in `scripts/gate-files.sh` (Open Question 2).
- B) JSON with stdlib. Same cost, but a second manifest syntax beside herdr's. Rejected on consistency.

**D4. Host detection order.** `ATEAM_HOST` first, then `T3CODE_THREAD_ID`, `HERDR_ENV=1`, `ENJOY_MCP_ENDPOINT`, else `cli`. The override comes first so a wrong guess is one env var away; the three markers cannot co-occur in practice. `return_to` is `T3CODE_THREAD_ID` or `HERDR_PANE_ID`; Enjoy exposes no conversation id in env, so `--return-to` overrides and the Enjoy skill text tells the agent to pass its conversation id. `cli` has no `return_to` and `notify` prints nothing: `codex exec` is synchronous.

**D5. How the record reaches the cache.**
- A) **Recommended:** `sync` fetches comments for open PRs whose head branch has a local worktree, bounded by `ghCommentBudget` (`forge.go:501`); on Gitea the per-issue comments URL (`:316`) serves the PR number. A new `review.ParseRecordComment` parses the grammar once; the gate's `:421-441` block switches to it. `BoardCard` gains `Review {SHA, Verdict, Agent, Model, Effort}`; pickup's worktree line appends `review record at <sha>, <verdict>, <agent> (<model>, <effort>)` and `stale` when the worktree head is not the sha. Cost: one comments call per worktree-backed open PR per sync; the hooks keep reading the cache only. The reviewer's last step is `ateam state sync`, so the shared cache is fresh before the dev session's next prompt.
- B) Fetch for every open PR. Scales with the repository, not the machine. Rejected.
- C) No fetch; `notify` writes the record into `state.json`. Fails when the reviewer runs on another machine, which T3 Code allows. Rejected as the only path, kept as a side effect of `notify` when the checkout is shared.

**D6. `notify` payload is fixed text.** The record body is model output and the wake message enters the dev session as a prompt, so the message carries only the PR number, the sha, and `run ateam state pickup`, as `tasklease.go:47` fixes error text. Findings stay on the PR.

## Risks / Trade-offs

- A manifest `launch` under `agent` runs a repository-controlled command; the no-shell rule and the gate-file listing limit it to reviewed changes. A fork PR cannot run it, since nothing executes in CI.
- A PR with two review records keeps the newest by comment order, as the gate does; a reviewer who re-reviews an older sha confuses `stale`.
- `sync` grows by one call per worktree-backed open PR; a developer with ten open worktrees pays ten calls under the shared budget, and a budget overrun drops the record silently until the next sync.
- `eligible` lists runs by policy, not availability: a listed Gemini model nobody has installed is still listed. The manifest or the agent picks.
- Attestation remains: nothing proves who started the reviewer under `manual`.

## Migration Plan

No key, no manifest, no record: output is byte-for-byte today's, apart from the skill text. This repository sets `review_launch` per Open Question 1 and ships `.agents/hosts/enjoy.toml` and `herdr.toml`. The handoff constraint line is dropped at the first handoff after the prose lands.

## Open Questions

1. **Owner ruling:** this repository's own `review_launch`. The shipped default is `manual`; this repository has worked as `agent` since 2026-10-30. Recommendation: `agent` here, `manual` for adopters.
2. **Owner ruling:** list `.agents/hosts/` in `scripts/gate-files.sh`, so a change to what `agent` mode executes needs `gate-change-approved`. Recommendation: yes; cost is one label per manifest edit.
3. Resolved by D5: comment fetch scope is worktree-backed open PRs only.
