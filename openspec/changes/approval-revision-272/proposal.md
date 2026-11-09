# Change Proposal: `approval-revision-272`

Approved-by: Coby Leuschke (2026-11-09) sha256:89b8f51879f4

## Why

`Approved-by: <name> (<YYYY-MM-DD>)` in `openspec/changes/<id>/proposal.md` records who approved and when, not what. An edit to `proposal.md` or `design.md` after the owner fills the line leaves the approval reading as valid on every surface: `spec-check`, `spec-queue`, pickup, and the `ateam/proposal` commit status (HAAG A7, finding F1 of the #271 walkthrough). The review record already solves this for PR review by naming the reviewed sha; the proposal approval has no equivalent. #181's one-click Approve must write a bound form, so this change lands first (owner, 2026-11-09).

## What Changes

- The approval line carries a content digest: `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>`, the digest of `proposal.md` without its `Approved-by:` line plus `design.md` when present (design D1, D2).
- `ateam state spec-check`: an in-flight change with a filled approval whose digest no longer matches fails with exit 1, naming the edited file, the last commit that touched it (or `uncommitted`), and the current digest the owner would re-approve with. A filled approval with no digest warns (D4). Archived changes are not checked.
- `ateam state spec-queue`, pickup, the MCP board, and the web queue label each change `approved sha256:…`, `stale approval (<file> edited at <sha7>)`, or `approval unbound`, and print the current digest for every unapproved or stale change so the owner can paste it (D3, D6).
- `ateam/proposal` commit status: a stale approval at the PR head is `pending` `approval stale: <id>` (D3).
- A pure `ApproveEdits` in `internal/state` produces the bound line and the hold removal from the two files; it is what #181's Approve write calls.
- `openspec/specs/harness/spec.md`: the queue Requirement and the commit-status Requirement gain the bound form and the stale scenario.
- `.agents/protocol.md` § 2 Clarify, the AGENTS.md rule row, `.agents/rules-index.md`, the bootstrap templates, and the docs state the bound form.

## Track

`design`: the approval form is adopter-facing surface (protocol, templates, docs) and two spec Requirements change. See `design.md`.

## Non-goals

- No change to the `OPEN-Q` hold mechanics. No signing or forge-side approval; the forge login stays an attestation (ADR 0001).
- No gate-file edit: `internal/review/gate.go` `approvalNotes` is unchanged (design Open Question 4).
- No UI and no forge write; both are #181. No re-approval of archived changes.
- `tasks.md` and the change's `specs/` deltas stay outside the digest (design Open Question 3).

## Definition of Done

- Tests with attached spec references: the digest is unchanged by filling the `Approved-by:` line, removing the hold, moving the change to the archive, and CRLF line endings; it changes after an edit to `proposal.md`, an edit to `design.md`, and a `design.md` added later; `spec-check` exits 1 naming the file and commit for a committed edit and `uncommitted` for a working-tree edit; an unbound filled approval warns with exit 0; an archived unbound approval is not checked; queue labels and the status row; `ApproveEdits` round-trips through `ParseProposal`.
- `task mutate:check` reports `detection` with the digest comparison inverted and with `design.md` dropped from the digest.
- Every archived change validates unchanged; `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green; the change is archived before the PR is marked ready.

Hand-verify probe.
