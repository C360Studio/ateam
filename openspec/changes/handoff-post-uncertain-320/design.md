# Design: `handoff-post-uncertain-320`

## Context

Opened in this session on `claude/handoff-post-uncertain-320` at `main` 787b900.

- `internal/state/forge.go:874` `PostPRHandoffComment` is the only post path. Callers: `cmd/ateam/main.go:388` (CLI) and `internal/mcpserver/server.go:340` (MCP). Tests: `state_test.go:265`, `commitstatus_test.go:585`, `token_test.go:93`, `:127`.
- GitHub (`forge.go:885-895`): `gh pr comment <branch>` runs under one 5 s `cctx`. Any error returns `gh pr comment …: <err>`; a killed `gh` after its HTTP write cannot be told apart from one killed before it. The same `cctx` then bounds `gh pr view` for the PR head (`:902`).
- Gitea (`forge.go:917-965`): `http.Client{Timeout: 5s}` lists open pulls, then POSTs to `issues/<n>/comments`. A `client.Do` error (`:951`) is returned bare; a status ≥ 300 is a definite rejection (`:956`).
- `forge.go:277` `isHandoffComment` already recognizes a five-field body. Sync reads PR comments with `gh pr view <n> --json comments` (`:714`) and `fetchGiteaIssueComments` (`:538`, 1.5 s, returns nil on any failure).
- `commitstatus.go:394` `ghRunError` already classifies `context.DeadlineExceeded` as a timeout: the precedent for D1.
- `cmd/ateam/main.go:390` prints `skipped` for every error; `:399` exits 0. `server.go:342` prints `skipped (<err>)`. `docs/hook-flow.md:82` says a post failure is reported on stderr and does not undo the checkpoint.
- `FormatFiveFieldPrompt` (`state.go:518`) is deterministic: the same five fields give the same bytes, so a retry with unchanged fields posts an identical body.

## Goals / Non-Goals

Goal: a handoff never says `skipped` when the comment may exist, and a retry with the same fields does not add a second copy. Non-goals are in `proposal.md`.

## Decisions

**D1. What counts as uncertain.**
- A) **Recommended:** only a timeout during the send step: `errors.Is(err, context.DeadlineExceeded)` or a `net.Error` with `Timeout()`. `PostPRHandoffComment` wraps it in an exported `ErrPostUncertain` (`errors.Is`-able). Cost: about 10 lines. Everything else (gh missing, no PR, HTTP ≥ 300, pull listing failure) stays `skipped`.
- B) Any error after the request may have been written (also connection resets, gh non-zero exits). This is closer to the truth, but gh's exit 1 also covers "no PR for branch", so most real skips would read as uncertain. Rejected.
- Seam: `ErrPostUncertain` is new exported surface in `internal/state` (not importable by adopters, `internal/`). An adopter sees only the new word in CLI/MCP text. Doing nothing leaves the false `skipped`.

**D2. Retry identity.**
- A) **Recommended:** before posting, read the PR's comments; if the newest comment for which `isHandoffComment` holds has a body equal (trimmed, CRLF-normalized) to the one about to be posted, return `Posted: true` with that comment's URL and `Existing: true`. Comparing only the newest means a deliberate A → B → A sequence still posts the second A. No format change.
- B) Embed `<!-- ateam-handoff <slot> <checkpoint> <updated_at> -->` and match on it. Exact, but `UpdatedAt` changes on every run, so a retry would never match unless the marker drops it, which leaves (slot, checkpoint), which equal-body already covers. Adds a format every reader must ignore. Rejected.
- Cost of A: one extra forge read per post (GitHub `gh pr view <branch> --json comments`; Gitea `issues/<n>/comments`, after the pull listing that already finds `<n>`). A failed read posts anyway (fail toward today's behavior: at worst a duplicate).
- Seam: an adopter sees `PR comment: posted (already on PR)`; commit statuses still run, because the PR head is read either way.

**D3. Budget.** The comment read gets its own 2 s timeout, separate from the post's 5 s, so a slow read cannot turn a sendable post into a timeout. The CLI's worst case grows from 5 s to 7 s.

## Risks / Trade-offs

- A post that times out and then never lands is reported uncertain; the rerun posts it. A post that times out and lands *later* than the rerun's read can still duplicate. This is the remaining window, and it is stated in docs.
- `gh pr view --json comments` may cap long threads (gh's GraphQL page); the newest handoff past the cap would not be seen, and the retry posts. Not verified in this session (Open Question 2).
- Body equality ignores who posted: another session's identical handoff counts as already posted. This is accepted, because the content is the same.

## Migration Plan

None. No stored state changes; old comments are compared by body like new ones.

## Open Questions

1. **Owner:** keep exit 0 on `uncertain` (recommended: the checkpoint is saved, and scripts that chain handoff keep working), or exit non-zero so a wrapper retries?
2. Does `gh pr view --json comments` return every comment on a long PR? To be checked against a PR with > 100 comments during implementation; if capped, use `gh api …/issues/<n>/comments --paginate`.
