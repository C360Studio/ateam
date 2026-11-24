# Design: whole-review-marker-399

## Context

- `internal/hook/hook.go:121,152` — the session-start hook only `state.Load`s the cache and calls `BuildPickupSummary`; it runs no git command, which keeps it under its 10 ms budget.
- `internal/state/state.go:2543-2600` — `Refresh` (run by `pickup`, `status`, `sync`, the MCP server) calls `SnapshotGit` (`:2789`), which already runs `rev-list --left-right --count` for the default branch (`:2810-2830`) and stores the result in `GitStatus` (`:555-570`).
- `internal/state/state.go:3074` — the `[ateam]` summary line prints `shortcuts=` and `holds=`; nothing reports repository-wide review age.
- `cmd/ateam/review.go:21-43` — `runReview` dispatches `compare`, `gate`, `request`/`notify`, `setup`, `route`, `check`; `cmd/ateam/` is not a gate file, `internal/review/` is (`scripts/gate-files.sh`).
- `internal/state/prune.go:249` — the existing precedent for ateam pushing to `origin` itself (branch deletion with `--force-with-lease`).
- `internal/state/settings.go:22-33` — `merge_method` may be `squash`, `merge` or `rebase`; `internal/bootstrap/reviewrecord.go:29` — under `review_record: aip` branches land `--ff-only`.
- `.agents/protocol.md:45` and `internal/bootstrap/templates.go:653` (`protocolCloseStep`), `reviewrecord.go:29` (`protocolCloseStepAIP`) — the Close step, in repository copy and both generated variants.
- No workflow triggers on tags (`.github/workflows/`: `ci.yml`, `gate-files.yml`, `merge-check.yml`); release tags are `v*` (`git tag`).
- Measured on this repository: `git diff --shortstat v0.3.1 HEAD` (481 files, +36,758/-1,823) took 0.04 s; `git rev-list --count --first-parent` 0.01 s.

## Goals / Non-Goals

**Goals:** a shared, durable marker for "the whole repository was read as one system at commit X"; a cheap, always-visible count of what landed since.
**Non-Goals:** the review's method or output; any gate; forcing a cadence.

## Decisions

### D1. Where the record lives

- **A (recommended): an annotated git tag `review/whole/<YYYY-MM-DD>-<short sha>`** at the reviewed commit, message carrying the reviewer and optional `--note` link. Shared by `git push`, auto-followed by `git fetch`, identical on GitHub and Gitea, no API or cache cap. Forecloses: adopters whose tag rules forbid non-release tags.
- **B: a closed forge issue labelled `review:whole`** with the commit in its body. Discussable, but needs a forge sync, two API dialects, and falls out of the cached 50 closed items.
- **C: a committed file** (`.agents/reviews/whole.json`). Needs a PR to record a review; the PR itself moves the count.
- *Adopter seam:* a release workflow triggered on every tag (`tags: ['*']`) would fire; `v*` patterns do not match. Doing nothing: the field shows `never`.

### D2. What is counted

- **A (recommended): first-parent commits and `git diff --shortstat` from the newest marker tag to the default branch's remote tip**, shown as `<n> commits, +<a>/-<d> lines`. Under squash merge (the default) one commit is one PR; under `merge` or `--ff-only` the label stays true.
- **B: merged PRs from the forge cache.** Matches #399's wording, but depends on sync freshness and the closed-item cap.
- *Adopter seam:* counts follow `origin/<default>`, so a stale fetch shows a stale count, like `behind=`.

### D3. Where it is computed and shown

- **A (recommended): in `SnapshotGit` during `Refresh`, cached in `GitStatus`, printed by `BuildPickupSummary` and in `ateam state status` JSON.** The hook keeps reading the cache; cost lands on `pickup`/`sync` (~0.05 s measured).
- **B: compute in the hook.** Freshest, but adds two git calls to a 10 ms path.
- "Newest" means the tag whose commit is latest on first-parent history; tags not reachable from the remote tip are ignored.

### D4. How a review is recorded

- **A (recommended): `ateam review whole --at <sha> [--note <url>] [--no-push]`**, implemented in `internal/state`, wired in `cmd/ateam/review.go`; it refuses a commit not on the default branch and pushes the tag to `origin`.
- **B: document `git tag -a … && git push` only.** No code, but an easy name or prefix mismatch makes the counter silently read `never`.
- *Adopter seam:* recording needs push rights to tags; `--no-push` lets a maintainer push later.

### D5. Display when no record exists

- **A (recommended): `since-review=never`**, no counts (a diff from the root commit would cost the most exactly where it says least).
- **B: omit the field** until the first record; the gap stays invisible, which is #399's failure.

## Risks / Trade-offs

- A counter is not a review; the protocol sentence is advice and nothing enforces it.
- Tag names are a contract: renaming the prefix later orphans old records.

## Migration Plan

Additive. Existing repositories show `since-review=never` until their first `ateam review whole`.

## Open Questions

1. **Owner ruling:** D1 storage — git tag (A) or forge issue (B)?
2. **Owner ruling:** D2 unit — commits (A) or forge PR count (B)?
3. **Owner ruling:** D4 — does the command push the tag by default?
