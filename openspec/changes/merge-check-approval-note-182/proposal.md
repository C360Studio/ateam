# Change Proposal: `merge-check-approval-note-182`

Approved-by: Coby Leuschke (2026-10-26)

## Why

#142 gave `proposal.md` an `Approved-by:` header and an `OPEN-Q` hold, but nothing at merge time says whether a change landed without owner approval. Step 3 of #142, split out as #182: the merge check should say so, without blocking, so we can measure how often it happens before deciding whether it should ever block.

## What Changes

- On a code pull request, `ateam review gate` (behind `scripts/merge-check.sh`) collects every OpenSpec change directory the PR touches: `openspec/changes/<id>/` or `openspec/changes/archive/<dated-id>/`, from each changed file's current name.
- For each, it reads `<dir>/proposal.md` at the PR head and, when the header is empty or absent (or the file is missing), prints `merge-check: note: OpenSpec change <id> has no Approved-by; the spec was not human-reviewed`. The exit status is unchanged.
- The proposal is read from the PR tree: it is documentation, not something the gate executes, so the #140 base-commit rule does not apply.
- Header parsing reuses #142's rule: `parseProposal` in `internal/state/openspec.go` is exported and `internal/review` imports it (`internal/state` depends only on `internal/testspec`; no cycle).
- One new forge read in `gateSource`, `fileAt(ctx, path, ref)`: GitHub `gh api` contents with the raw media type; Gitea `GET /repos/{owner}/{repo}/raw/{path}?ref=<sha>`. A 404 is "missing". Any other read error prints a note naming the path and the run continues: a documentation note never makes the gate unavailable.
- One scenario in the harness spec. No `AGENTS.md` row: the note is informational and enforces nothing.

## Non-goals

- Blocking on a missing approval; validating the `<name> (<YYYY-MM-DD>)` shape or who wrote it.
- Documents-only pull requests: they pass before the code check runs, and a proposal-only PR is expected to be unapproved.
- #201 (a `REQUEST CHANGES` record passes the gate): separate PR, same label.

## Definition of Done

- `task verify` green on a clean tree; `ateam doctor` parity green.
- Tests in `internal/review/gate_test.go` on the fake-forge fixture: unapproved change prints the note and exits 0; approved prints nothing; archived change path; missing proposal; read error prints a note and does not exit 2. Each shown able to fail by `task mutate:check`.
- `scripts/merge-check.sh` and `internal/review/` are gate files: the owner's `gate-change-approved` label before merge.
