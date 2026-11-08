# Change Proposal: `gate-approval-head-319`

Approved-by:

## Why

The owner's gate-file approval is the label `gate-change-approved` (`scripts/gate-files.sh:18`, `:83`). A label carries no commit. `Gate files` reruns on `synchronize` (`.github/workflows/gate-files.yml:10`) and passes again while the label stays on, so commits pushed after the owner approved inherit the approval (#319, HAAG A7, finding F2).

## What Changes

Recommended (design D1 option B): the approval names the commit it covers.

- `gate-files.sh` passes a gate-file PR only when it carries the label **and** the newest PR comment that starts `gate-change-approved: <sha>` names the PR's live head (full sha, or a prefix of at least 7 hex). Otherwise it fails with `the owner's approval covers <sha>, not head <head>` or `no gate-change-approved: <sha> comment`.
- The label stays: it triggers the rerun after the owner comments, and removing it still revokes.
- The same holds on Gitea (`ATEAM_FORGE=gitea`, issue comments API).
- `.agents/protocol.md` § Gate files, the `AGENTS.md` / `.agents/rules-index.md` gate-files rows, and `README.md:260` tell the owner to post `gate-change-approved: <head sha>` and then apply (or re-apply) the label. The `Gate-file changes wait for the owner` scenario says the approval is bound to the head.

## Track

`design`: it edits gate files (`scripts/gate-files.sh`, and possibly the workflows) and changes what an owner approval means. See `design.md`.

## Non-goals

- No write permission for the `Gate files` workflow (option A); it stays read-only.
- No proof of who wrote the comment: with a shared forge login it is an attestation, as the label is today.
- No `ateam` command to post the approval; it is a one-line `gh pr comment`.

## Definition of Done

- `TestGateFilesCheck` and `TestGateFilesCheckGitea` cover: label + comment at head (pass), at a short prefix (pass), at an older sha (fail naming both), a newer comment for an older sha after one at head (fail), no comment (fail), comment without label (fail), and a non-gate PR (pass with neither). `task mutate:check` reports `detection` with the head comparison removed.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` green; the change archived before ready; `gate-change-approved` applied to this PR by the owner at its final head.
