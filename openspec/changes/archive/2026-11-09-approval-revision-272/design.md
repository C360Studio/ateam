# Design: `approval-revision-272`

## Context

Opened in this session on `claude/approval-revision-272` at `main` e00688e. HAAG A7 (finding F1 of #271) says a human approval is bound to the input revision and is invalid once it changes.

- `internal/testspec/check.go:307-321` `ApprovedBy` returns the whole trimmed value of the first unfenced `Approved-by:` line; `:325` `EachUnfencedLine` is the fence rule. Both `internal/state` and `internal/review` read through it.
- `internal/state/openspec.go:42-45` `OpenSpecChange.ApprovedBy` (`json:"approved_by"`); `:58-69` `ParseProposal`; `:144` `ParseTasksMarkdown`; `:250` `holdMarker` classifies `OPEN-Q`; `:302-355` `ScanOpenSpecQueue` already reads `tasks.md`, `proposal.md` and `design.md` per change and skips `archive`; `:393` `FormatOpenSpecQueue` is the one renderer behind `spec-queue` (`cmd/ateam/main.go:199-204`), pickup (`state.go:2446` fills `OpenSpecQueue`), MCP (`internal/mcpserver/server.go:398`, `:531`) and the web queue (`internal/web/server.go:657-663`); `:437-520` `CheckOpenSpec`: strict validate, stray holds exit 1 (`:510-515`), line-limit warnings exit 0 (`:518-524`).
- `internal/state/commitstatus.go:77-82` `touchedChange{ID, Proposal, ApprovedBy, OpenQ}`; `:109-127` the `ateam/proposal` rows; `:239-277` `touchedChanges` reads `proposal.md` and `tasks.md` at the head with `git show`, not `design.md`.
- `internal/review/gate.go:405-420` `approvalNotes` prints a note when `Approved-by` is empty; a gate file under `scripts/gate-files.sh`. `:652-668` and `:744` `gateCommitAllowed` are the record precedent: a later content commit invalidates, merges of `main` and docs-only commits are exempt.
- `openspec/specs/harness/spec.md:80-82` queue Requirement, `:99-103` approval scenario, `:628-642` gate-note Requirement, `:1014-1021` status scenario.
- Form stated in `.agents/protocol.md:30`, `AGENTS.md:84`, `.agents/rules-index.md:35` (row parity checked by `internal/bootstrap/adopter_rules_test.go:47`), `internal/bootstrap/templates.go:223`, `:264`, `:292`, `:353`, `docs/daily-use.md:38-39`, `docs/adoption.md:13`, `:111`, `:122-123`, `docs/brownfield.md:137`, `README.md:71`, `:210`.
- 52 of 119 archived changes carry a filled unbound approval; `openspec/changes/` has none in flight on `main`. PRs squash-merge (protocol § 2 step 5), so branch commits do not survive on `main`; the archived line does. #181 writes through the forge contents API, which on GitHub may take two commits.

## Goals / Non-Goals

Goals: an approval that names what was approved, checked wherever the approval is shown, with no new command and no gate edit. Non-goals are in `proposal.md`.

## Decisions

**D1. Revision identity.**
- A) **Recommended: content digest.** `sha256` over `proposal.md` with its `Approved-by:` line removed, then `design.md` when present (each LF-normalized, length-prefixed), cut to 12 hex. Cost: ~60 lines plus tests. Unchanged by the approving edit itself, the hold removal, the archive move, a rebase, the squash, and #181's two-commit write, so the issue's exemptions come free. Forecloses naming the commit the owner read; the failure names the last commit touching the file instead.
- B) Commit sha: `at <sha7>`, the approving commit's parent; check `git log <sha>..HEAD -- proposal.md design.md` minus the approving commit (found by diffing for the one line). Breaks on rebase (sha gone, check impossible), dangles in the archive after squash, and #181 must learn the parent before writing.
- C) Both tokens: twice the parse surface for the same guarantee. Rejected.
- Seam: an adopter who never reads this sees archived approvals validate as before and in-flight filled ones warn (D4).

**D2. Line shape.**
- A) **Recommended:** one line, token appended: `Approved-by: Coby Leuschke (2026-11-10) sha256:3f9a1c2b4d5e`. `ApprovedBy` is untouched; a `parseApproval(value) (name, digest)` splits a trailing `sha256:<hex>` token, and surfaces show the name without it. One edit, as today.
- B) A second header `Approved-digest:`. Readers of `ApprovedBy` are untouched, but the owner edits two lines, the fence rule applies twice, and the status chooser and gate read two values.

**D3. Where it is computed and shown.** `ScanOpenSpecQueue` fills `Approval{Name, Digest, Current, State}` with `State` one of `unapproved`, `approved`, `stale`, `unbound`; it already holds all three files. `CheckOpenSpec` fails on `stale` with `openspec/changes/<id>/proposal.md: approval sha256:X is stale; <file> changed since (last commit <sha7> | uncommitted); re-approve with sha256:Y`, running `git log -1 --format=%h -- <file>` only on that path, so the green path costs no git call. `FormatOpenSpecQueue` prints the label after the name and the current digest for `unapproved` and `stale`. `touchedChanges` also reads `design.md` at the head and sets `Stale`; the chooser adds `pending` `approval stale: <id>` ahead of the approved row (Open Question 5). The gate note stays as it is (Open Question 4). Alternative: a separate `spec-approval` command. Rejected: it splits the gate that `task spec:check` already is.

**D4. A filled approval with no digest on an in-flight change.**
- A) **Recommended:** `spec-check` warns `approval unbound`, the queue says so, the status stays `success`. Nothing in flight breaks on upgrade; the owner flips it to a failure in a later quick change once every in-flight change is re-approved.
- B) Exit 1 now. Every adopter's in-flight approved change fails at upgrade until the owner re-approves it.

**D5. Digest scope.**
- A) **Recommended:** `proposal.md` and `design.md`, as #272 states. B) Also the change's `specs/**/spec.md` deltas: they carry the Requirement text, but scenarios are tightened during implementation and each tightening would need an owner re-approval. C) Also `tasks.md`: changes on every step. Rejected.

**D6. Who produces the digest.**
- A) **Recommended:** no new command. `spec-queue` prints the current digest for every unapproved or stale change; the owner pastes it, or the agent writes the line at the owner's word as in #302 task 1.1. `ApproveEdits(proposal, tasks []byte, name string, date time.Time) (proposal, tasks []byte, err error)` fills the line and removes the first task's `OPEN-Q` hold; it is pure, so #181's handler calls it on the blobs it fetched.
- B) `ateam state spec-approve <id> --name <n>` applying `ApproveEdits` to the local checkout. Convenient, but a new write whose human-use boundary (the owner's checkout against an agent's worktree) needs its own ruling; it is #181's listed follow-up.

## Risks / Trade-offs

- A typo fix in `proposal.md` after approval fails `spec:check` until the owner re-approves. Intended (A7): evidence goes in `tasks.md`, and the fix is one line.
- A `design.md` added after a quick-track approval makes the approval stale. Correct: it was never approved.
- 12 hex is 48 bits and not a signature. Anyone can recompute and paste it; the line proves what was approved, not who (ADR 0001).
- A head object missing locally already yields no `ateam/proposal` status; the stale row inherits that.
- The failure text names the last commit touching the file, which may not be the one that broke the approval when several followed.

## Migration Plan

Additive. Archived changes are untouched and unchecked. In-flight filled approvals warn (D4 A). `approved_by` keeps the raw value; a new `approval_state` field is `omitempty`. Rollback is the old binary: it reads the whole value and shows the token in the status description, harmless.

## Open Questions

Ruled 2026-11-09 by the owner: 1 → D1 A, 2 → D4 A, 3 → D5 A, 4 → A (gate unchanged). Question 5 was not ruled; the stale status stays `pending`.

1. **Owner ruling (changes the spec scenario and tasks 2.1–2.3):** D1 A digest or B commit sha.
2. **Owner ruling (task 2.2):** D4 A warn or B fail for a filled approval with no digest.
3. **Owner ruling (task 2.1):** D5 A two files, or B also the `specs/` deltas.
4. **Owner ruling:** leave `gate.go` `approvalNotes` unchanged (A) or extend it to say stale (B: a gate-file PR needing `gate-change-approved`, separate from this one).
5. Not blocking: the stale status is `pending` (matches the record row) or `failure`.

Drafted by claude (fable 5.1, medium) via the architect contract.
