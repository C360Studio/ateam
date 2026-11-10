# Design: `spec-approve-181`

## Context

Opened on 2026-11-10 at `ba0f2ee`:

- `internal/state/approval.go:137` defines the hold text `Hold: OPEN-Q proposal awaits owner approval`. `ApproveEdits(proposal, design, tasks, name, date)` at `:144-200` returns the bound `Approved-by: <name> (<date>) sha256:<12 hex>` proposal and the tasks file without the hold. It errors when either line is missing, and the digest covers `proposal.md` and `design.md` (`ApprovalDigest`).
- `internal/web/server.go:155-177` `validateMutationRequest`: POST only, rejects `Sec-Fetch-Site: cross-site` and a foreign `Origin`, requires JSON, then `authorizeAPIRequest` (`:122-142`). With no `ATEAM_API_TOKEN`, any loopback client passes; with one set, every client must send it.
- `internal/web/server.go:97` registers `/api/openspec`; `:650-670` serves it from `ScanOpenSpecQueue(root)`.
- `internal/state/commitstatus.go:338-360` `PublishCommitStatuses` dispatches on `cfg.Provider`: GitHub through `gh api` with `ghCommand(ctx, WriteToken(), ...)` (`:449-463`), Gitea through `net/http` (`:470`). It is the only forge write besides PR comments and labels; no code writes file contents.
- `internal/state/token.go:24` `WriteToken()` resolves `ATEAM_TOKEN_WRITE`, then `ATEAM_TOKEN_READ` (spec *Forge token roles*, `openspec/specs/harness/spec.md:155`).
- `internal/state/forge.go:36-41` `ForgeConfig{Provider, BaseURL, Owner, Repo}`; `:295` lists worktrees with `parseWorktrees`.
- `internal/review/gate.go:717-722` `gateIsDoc`: any `.md` outside the agent and reviewer contracts is a document, so an approval commit is docs-only and does not void a review record.

## Goals / Non-Goals

Goals: one click writes the owner's approval edit on the change's branch, through the forge, with optimistic concurrency, and never touches a worktree. Non-goals: the proposal's list (edit/create/delete), any new auth scheme, and local writes.

## Decisions

### D1. Where the write goes

- **A. Forge contents API (recommended).** GitHub `PUT /repos/{o}/{r}/contents/{path}` per file. Gitea `POST /repos/{o}/{r}/contents` takes several files in one commit. Each call sends the blob `sha` it read, and a stale one is rejected. Cost: one helper of roughly 150 lines with two providers. Blast radius: one branch on the forge. Forecloses nothing; edit/create reuse it.
- **B. GitHub git data API (blobs, tree, commit, ref update).** This gives one commit on GitHub too, but it takes five calls and needs its own ref-update concurrency check. It costs about twice A on GitHub for no reviewer-visible gain.
- **C. Local `git commit` in the owning worktree.** Rejected: it breaks *never touch a worktree you do not own*, and the agent's uncommitted work would mix with the owner's edit.

Adopter seam: an adopter needs `ATEAM_TOKEN_WRITE` (or the fallback chain) with contents-write scope. Without one, the endpoint returns the forge's 401/403 as 502 with the message, and nothing is written.

### D2. Commit order on GitHub

Under A, GitHub takes two commits. Write **`proposal.md` first**. If the second write fails, the branch has an approved header with the hold still present. That state is safe: `spec:queue` still shows `OPEN-Q`, and retrying re-applies `ApproveEdits`, which rewrites the header and removes the hold. The other order can leave the hold removed with an empty `Approved-by:`, which reads as unblocked. The 409/502 response names the half that landed.

### D3. Reading a consistent revision

Resolve the branch head commit once, then read all three files at that commit SHA, not at the branch name, so the digest covers the bytes that were shown. The blob `sha` values come from the same read, so a push between the read and the write gives 409, not a silently wrong digest. The branch comes from the request. The handler checks that it is a branch carrying `openspec/changes/<change>/` at that head, and refuses `main` (400).

### D4. Identity and trust (owner ruling, 2026-11-10)

- **A. Keep loopback trust (ruled).** Any process on the owner's machine that can reach the server, including an agent, can call the endpoint, and the commit carries the write token's login. Cost: none. Risk: an agent can approve its own proposal. Mitigation inside A: the commit message is `docs(openspec): approve <change> via ateam serve`, which keeps the click distinguishable in history, and the response and the server log name the caller's address.
- **B. Require `ATEAM_API_TOKEN` for this endpoint even on loopback.** Not taken. It is the upgrade path if the risk materialises, and it is a one-line guard.
- **C. Copy the edit for the owner to commit.** Not taken.

The handler carries `// shortcut: loopback clients can approve (owner ruling 2026-11-10), require ATEAM_API_TOKEN here if an agent is seen calling it`. The name is the forge login from `GET /user` (`gh api user`, Gitea `/api/v1/user`). There is no name field in the request; see OQ1.

## Risks / Trade-offs

- Loopback trust, per D4. The owner accepted it; the shortcut marks the upgrade.
- An agent's local branch can be ahead of the remote. The approval binds the pushed bytes. If the agent later pushes a different `proposal.md`, `spec:check` fails on the stale digest, which is the existing #272 behaviour and the intended one.
- Gitea's response status for a stale `sha`: I have not checked whether it is 409 or 422. The recorded-response test settles it at implementation, and both map to 409.

## Migration Plan

Additive. There is no stored state. Without a write token, the button is still shown and fails with the forge's message.

## Open Questions

- **OQ1 (owner).** Should the request accept a `name` override, as #181's text says (`{change, name}`, defaulting to the login), or always use the login? This design takes login-only: fewer inputs at a loopback-trusted endpoint, and the header matches the commit author.
- **OQ2 (owner).** On GitHub, two commits (D1 A) or one commit through the git data API (D1 B)? This design takes two commits, which #181 accepts.
