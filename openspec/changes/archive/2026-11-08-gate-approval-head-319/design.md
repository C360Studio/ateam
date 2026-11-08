# Design: `gate-approval-head-319`

## Context

Opened in this session on `claude/gate-approval-head-319` at `main` 3fca537.

- `scripts/gate-files.sh:18` `approval=gate-change-approved`; `:77-88` reads labels and passes on an exact match. Labels come from `gh pr view --json labels` (`:49`) or the Gitea pull object (`:45`). `:47` `read_pull` already reads the live PR, so `head.sha` is in hand.
- `.github/workflows/gate-files.yml:9-14`: `pull_request_target` on `opened, synchronize, reopened, ready_for_review, labeled, unlabeled`, `concurrency … cancel-in-progress: true`, `permissions: contents: read, pull-requests: read`. The job is skipped on a draft except on `unlabeled` (`:25-26`). Gitea twin: `.gitea/workflows/gate-files.yml:11-19`, and it adds `edited`.
- Merge-check already binds a review record to one commit (`Reviewed at <sha>`, `internal/review/gate.go`); this is the precedent for a sha-bound approval.
- Tests: `internal/bootstrap/gatefiles_test.go:58` (GitHub fake `gh`), `:266` (Gitea `httptest`), `:135` (runs from base), `:178` (Gitea twin of the GitHub workflow).
- In this session, PR #330's `ready_for_review` run was cancelled by a run for a body edit made seconds earlier (`gh run list`: `cancelled` then `skipped`). Event-driven state in this workflow can be lost to `cancel-in-progress`.

## Goals / Non-Goals

Goal: an owner approval of a gate-file change covers one head; any later push fails `Gate files` until the owner approves again. Non-goals are in `proposal.md`.

## Decisions

**D1. How the approval is bound to a revision.**
- A) Revoke on push (issue option 1): on `synchronize`, the workflow removes the label and fails. Owner UX is unchanged (re-apply the label). Cost: `pull-requests: write` on a `pull_request_target` job, whose read-only token is the premise of #140. Holes: the job is skipped on a draft, so a push to a draft never revokes, and `ready_for_review` then passes with the stale label; and a `synchronize` run cancelled by a later event (`cancel-in-progress`, seen on #330) never revokes. Each hole needs its own fix, and a missed event fails open.
- B) **Recommended:** bind to a sha the script compares on every run (issue option 2). The owner posts `gate-change-approved: <sha>` and applies the label; the script passes only when the newest such comment names the live head. It is stateless: every run compares live head with the newest comment, so a lost or skipped event cannot leave a stale pass, and the workflow stays read-only. Cost: about 20 lines of shell per forge, plus one comment read (`gh api --paginate …/issues/<n>/comments`; Gitea `issues/<n>/comments`). Friction: after each push, the owner posts a new comment and re-applies the label (or reruns the job).
- C) Compare the `labeled` event time with the head commit's date. Commit dates are set by the pusher and forgeable, and neither forge records push time on the PR. Rejected.
- D) Native code-owner review with "dismiss stale approvals". GitHub forbids approving one's own PR, so it fails under the shared forge login the label exists for, and Gitea parity differs. Rejected.
- Seam: `ateam init` does not generate `scripts/gate-files.sh` (no non-test reference in `internal/bootstrap`), so only repositories that copied it by hand change; an owner who keeps applying only the label sees `no gate-change-approved: <sha> comment` and the exact command to post. Doing nothing leaves #319 open.

**D2. Comment shape.** A line `gate-change-approved: <sha>` at the start of a comment body (after trimming), sha `[0-9a-f]{7,40}`, matched as a prefix of the live `head.sha`. The newest matching comment wins, so a later approval for an older sha revokes. Any author counts (shared login; attestation as today).

**D3. Rerun trigger.** No new event: the owner's label (re-)application triggers `labeled`. Adding `issue_comment` would run on the default branch, not the PR head, so its check would not land on the PR. Rejected.

## Risks / Trade-offs

- Owner friction per push on gate-file PRs (one comment + label toggle). Gate-file PRs are rare (#319, #325 in v0.4.0).
- A comment from an agent sharing the owner's login can forge approval; unchanged from the label today.
- An adopter who copied `gate-files.sh` by hand keeps the label-only rule until they copy it again.

## Migration Plan

Open gate-file PRs at merge time need the comment. This PR itself is judged by the base-branch script (label only), so its own approval is the label.

## Open Questions

1. **Owner (ruled 2026-11-08: option B):** option B with its per-push friction, or option A with write permission and the draft and cancellation holes fixed separately?
2. Gitea: does the job token read issue comments on a private repository? Resolved: the Gitea `Merge check` job already reads `issues/<n>/comments` with `github.token` for review records (`internal/review/gate_gitea.go:279`, `.gitea/workflows/merge-check.yml:56`).
