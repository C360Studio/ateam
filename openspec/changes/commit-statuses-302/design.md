# Design: `commit-statuses-302`

## Context

Opened in this session on `claude/commit-statuses-302` at `main` f75ff30. Owner constraints are given: informational only, never a required check, never from a hook, no change to `task merge:check` or the ruleset, no UI, no T3-specific code.

- `internal/state/forge.go:829-897` `PostPRHandoffComment` dispatches on `cfg.Provider`. GitHub shells out to `gh pr comment` through `ghCommand(ctx, WriteToken(), ...)` (`:842`) and discards the output, so the comment URL is lost; Gitea lists open pulls (`:851`), matches `head.ref`, and POSTs (`:876`), discarding the response body. It returns only `(bool, error)`. GitHub is reached only through `gh`, never raw HTTP.
- Two callers post: `cmd/ateam/main.go:386-390` (`--no-post` at `:339`, skipped on `ForgeLocal`) and `internal/mcpserver/server.go:334-348` (`post` omitted or true).
- `internal/state/token.go:10-12` already lists statuses as a write; `WriteToken` `:24`; `ghCommand` sets `GH_TOKEN` from the role token `:44-52`; `giteaToken` `:36`.
- `internal/state/state.go:469-482` `Handoff` carries `CheckpointSHA`, `Dirty`, `Stale`, `StaleReason`, `UpdatedAt`. `Stale` is judged against the *local* head of the slot's branch (`:991-1005`, `:1019-1027`), not the PR head. The posted comment carries the five fields only, no SHA (`:519-533`).
- `state.go:1370`: the cache is `<root>/.ateam/state.json`, and `cmd/ateam/main.go:670` makes `root` the worktree, so a worktree's handoff lives in that worktree's cache, not the main checkout's.
- `cmd/ateam/main.go:422-452` `runStateSync`: 10s context, `SyncForgeBoard`, then `Update`. `SyncForgeBoard` (`forge.go:186`) is also reached from pickup, prune, board and MCP through `RefreshWithBestEffortSync` (`main.go:252`, `:288`, `:466`; `mcpserver/server.go:294`, `:471`) and from the UI sync (`internal/web/server.go:723`). No hook calls it (`internal/hook/hook.go` only calls `SyncBranchHead`).
- `forge.go:251-261` `worktreeBranches`; `forge.go:663-694` (GitHub, one `gh pr view --json comments` per worktree PR under `ghCommentBudget` `:566-572`, at most 8 at once) and `forge.go:353-368` (Gitea, `giteaCommentFetchLimit` 8) keep only bodies and set `Review` from `latestReview` (`:263`). `gh`'s comment objects carry `url` (probed on PR 301).
- `forge.go:712-772` `mapForgeItemToCard` fills `BoardCard.HeadSHA`, `Branch`, `URL`, `Review` (`state.go:570-591`). `review.RecordComment` (`internal/review/recordcomment.go:15-26`): `Verdict` may be `other` or `""`.
- `internal/state/openspec.go:58-70` `ParseProposal` and `:144` `ParseTasksMarkdown` take bytes, so a proposal can be read at any SHA with `git show`; `:247-256` classifies the `OPEN-Q` hold; `:42-45` `ApprovedBy`.
- `internal/review/gate.go:712-717` `gateIsDoc` and `:426-449` `gateChangeDirs` are unexported, under the gate path in `scripts/gate-files.sh:18`. No exported classifier exists. The gate never reads commit statuses (no `statuses` or `check-runs` in `gate_github.go`, `gate_gitea.go`, `merge-check.sh`).
- `cmd/ateam/review.go:478-482` `forgeTokenWarnings` is the only token check in `doctor` (`main.go:595`) and makes no network call; CI runs `doctor --check` (`.github/workflows/ci.yml:57`). `BoardSyncError` (`state.go:629`) is the precedent for a persisted forge error. A classic token answers `X-Oauth-Scopes` (probed); a fine-grained one sends none.
- Fixtures: `installForgeCLI` (`internal/state/forge_settings_test.go:17`) fakes `gh`; `httptest.NewServer` fakes Gitea (`forge_test.go:63`, `reviewrecord_test.go:109`).

## Goals / Non-Goals

Goals: the three contexts on the PR head from data ateam already holds, at most one read per PR, writes only on change. Non-goals are in `proposal.md`.

## Decisions

**D1. Where the write lives and how sync finds its inputs.**
- A) **Recommended:** new `internal/state/commitstatus.go`: a pure `chooseStatuses(inputs) []CommitStatus` and `PublishCommitStatuses(ctx, root, cfg, pr, []CommitStatus) (StatusResult, error)`. Called from `runStateSync` after `Update` with the cards just returned, and from both handoff callers after the comment. Sync maps each open, unmerged PR card with a `HeadSHA` to the worktree whose `Branch` matches (`parseWorktrees`, `state.go:2607`); it reads the handoff from `Load(<worktree>)`, the newest slot on that branch by `UpdatedAt`. Cost: ~200 lines plus tests. Blast radius: two commands and one MCP action.
- B) Post inside `SyncForgeBoard`. Cost: pickup, prune, board, MCP and the UI button all start writing, inside the 3s pickup budget. Rejected.
- C) A separate `ateam state statuses` command. Forecloses nothing, but statuses appear only for adopters who learn to run it.
- Seam: statuses follow `sync` and posting handoffs only; doing nothing means they refresh on those two paths and lag a bare `git push` until the next one.

**D2. "Code PR" for the proposal context without editing the gate.** The PR's change directories come from `git diff --name-only <merge-base of DefaultRemoteRef and head>..<head>` in the worktree (`state.go:563`), read at the head with `git show`.
- A) Export `gateIsDoc` from `internal/review`. One source of truth, but the PR becomes a gate change needing `gate-change-approved`.
- B) Copy the four-line rule into `internal/state` with a comment naming `gate.go:712` and a table test of the gate's examples. No gate edit; drift is possible and only mislabels an informational status.
- C) **Recommended:** no classifier. Post `ateam/proposal` only when the PR touches an OpenSpec change; post nothing otherwise. Forecloses the "missing change" signal. Reason: quick-track and trivial PRs legitimately carry no change, adopters without `openspec/` would get a red `error` on every code PR, and T3 renders `error` as a failed check. Open Question 1.

**D3. State chooser (first match wins).** Descriptions are cut to 140 bytes on a rune boundary and stripped of control characters (record fields come from PR comments).

| Context | Condition | State, description |
| --- | --- | --- |
| proposal | no touched change (D2 C) | nothing |
| proposal | any touched change has empty `Approved-by` or an open `OPEN-Q` hold | `pending` `awaiting owner approval (OPEN-Q): <id>` |
| proposal | all approved | `success` `approved by <Approved-by>: <id>` |
| handoff | no slot on the branch | nothing |
| handoff | checkpoint is not the head (`SameCommitSHA`, `state.go:487`) | `pending` `handoff stale: at <sha7>, head <sha7>` |
| handoff | `Dirty` | `pending` `handoff at <sha7> has uncommitted changes` |
| handoff | otherwise | `success` `handoff at <sha7>, <UpdatedAt UTC to the minute>` |
| record | no record | nothing |
| record | record SHA is not the head | `pending` `record at <sha7> is behind head <sha7>` |
| record | `APPROVE` / `REQUEST CHANGES` | `success` / `failure` `<verdict> by <agent> (<model>, <effort>) at <sha7>` |
| record | `other` or `""` | `pending` `record at <sha7> has no single verdict` |

The issue's `<age>` is replaced by the handoff time: an age changes every run and would defeat D4. The stale row compares with the PR head, not `Handoff.Stale`, which judges the local head. The record row says "behind head", never what the gate decides: merge-check accepts docs-only commits after a record (`gate.go:744-747`).

**D4. Idempotence read.**
- A) **Recommended:** one combined-status read per PR (`GET repos/{o}/{r}/commits/{sha}/status`; Gitea `/api/v1/repos/{o}/{r}/commits/{sha}/status`), then POST only the contexts whose state or description differ. GitHub through `gh api`, read token for the GET, write token for POSTs. Each call 1.5s; sync runs at most 8 PRs at once inside its remaining 10s.
- B) Cache the last post in `.ateam/state.json`. Zero reads, but the cache is per checkout (`state.go:1370`): a handoff in the worktree and a sync in the main checkout, or a second machine, re-post every time. Rejected for duplicates.

**D5. Switch.**
- A) **Recommended:** on wherever a handoff posts or sync reaches a remote forge; `ATEAM_COMMIT_STATUSES=0` turns both off; `--no-post` and `post: false` skip statuses with the comment. `sync`'s JSON gains `statuses: {posted, unchanged, errors}`. Adopter seam: after upgrading, up to three `ateam/*` checks appear on PRs with a worktree here; none is required; docs say how to turn them off.
- B) Off by default; `ATEAM_COMMIT_STATUSES=1` turns them on. Cost: the human surface stays empty until someone reads the docs. Open Question 2.

**D6. `target_url`, with no extra call.** HTML repo URL is the PR URL minus `/pull/<n>` (GitHub) or `/pulls/<n>` (Gitea).
- A) **Recommended:** proposal: `<repo>/blob/<sha>/<path>` (Gitea `<repo>/src/commit/<sha>/<path>`). Handoff: on the handoff path, the URL `gh pr comment` prints or the Gitea response's `html_url`; on sync, the newest handoff-block comment from comments already fetched. Record: the record comment's URL from the same fetch. Fallback for all: the PR URL.
- B) The PR URL for all three. Zero cost; no deep link.

**D7. Doctor's warning.**
- A) **Recommended:** report the outcome. A failed status write records `StatusWriteError` (`<context> on <sha7>: HTTP <code>` or `gh exit <n>`, never a body or token) beside `BoardSyncError`; a success clears it; `doctor` warns from the cache. No network, so CI's `doctor --check` is unchanged. Weakness: silent until the first attempt, and per checkout.
- B) Probe: `X-Oauth-Scopes` contains `repo` or `repo:status`; otherwise `permissions.push` from `GET repos/{o}/{r}`. A fine-grained GitHub token and a Gitea scoped token cannot be judged without a write, and doctor gains a network call in CI. Open Question 3.

## Risks / Trade-offs

- Statuses lag a push until the next sync or posting handoff; a handoff taken before its push posts `pending` on the old head.
- The change list depends on the local `DefaultRemoteRef`; a stale base can list a change already merged.
- `failure` and `error` render red in GitHub, Gitea and T3 though nothing is required. Not verified: that a red non-required status leaves GitHub auto-merge armed.
- Combined status returns the first 100 contexts; a sha with more could hide an `ateam/*` one and cause a duplicate post.

## Migration Plan

Additive. No cache migration; `StatusWriteError` is `omitempty`. Rollback is `ATEAM_COMMIT_STATUSES=0` or the old binary; statuses already posted stay on their shas, harmless.

## Open Questions

1. **Owner ruling (changes spec and task 2.2):** D2 C (no status without a touched change) or B (copy the rule; `error` on code PRs without one). A needs `gate-change-approved`.
2. **Owner ruling:** D5 default on (A) or off (B).
3. **Owner ruling (changes the spec scenario and task 2.7):** D7 outcome warning (A) or a scope probe (B).
4. Not blocking: D3 adds a `Dirty` row the proposal does not name; drop it if a dirty handoff at the head should read `success`.

Drafted by claude (opus, low) via the architect contract.
