# Design: `merge-check-workflow-255`

## Context

Opened in this session against `origin/main` e7db8f4, with PR #258 (`7adf753`, Gitea WIP deferral, auto-merge armed) as the expected base.

- `.github/workflows/ci.yml:10-11`: `pull_request` types `[opened, synchronize, reopened, ready_for_review]`; `:17-19` concurrency per ref, cancel except `ready_for_review`.
- `.github/workflows/ci.yml:62-94`: `merge-check` job: #250 draft guard and rename, base-commit checkout (`ref: ${{ github.event.pull_request.base.sha || github.sha }}`, #140), `go build` of `ateam` from base, `scripts/merge-check.sh "$PR_NUMBER"` with `GH_TOKEN`/`GH_REPO`.
- `.github/workflows/ci.yml:103-118`: `required` needs `[verify, merge-check]`, `if: always() && (...)`, fails on any non-success.
- `.github/workflows/gate-files.yml:8-27`: the precedent for a separate required workflow (`pull_request_target`, base checkout, concurrency per PR number, cancel-in-progress).
- PR #258 `.gitea/workflows/ci.yml`: adds `edited` to the Gitea types (leaving `WIP:` is an edit) and the draft guard to the Gitea twin.
- `internal/review/gate_github.go:159-176`: the rule check looks for one context, `const check = "Required"`. `internal/review/gate.go:59` `giteaRequired = "CI / Required (pull_request)"`; `gate_gitea.go:166-187` requires it exactly (owner ruling 1 on #143: no glob).
- Live ruleset on `main` (read 2026-10-29): `required_status_checks` = `Required`, `Gate files`; strict on. The `Merge check` context already exists (today's `ci.yml` job), so the owner can select it in the ruleset before this change merges.
- `docs/adoption.md:77`: adopters call `ateam review gate` from their own CI. Any change to the rule check reaches them.
- `.agents/skills/review-pr/SKILL.md:37-39` (template `internal/bootstrap/templates.go:514`): step 5 tells the reviewer to `gh run rerun --failed`. `docs/daily-use.md:99` documents the one red check.
- Tests: `internal/bootstrap/mergecheck_test.go:915-926` (`TestCIWorkflowPinned`, sensitivity), `gatefiles_test.go:134` (`TestGateFilesWorkflowRunsFromBase` reads the `merge-check` job out of `ci.yml`), `gatefiles_test.go:179` (`TestGiteaWorkflowsTwinGitHub` swap table for `ci.yml`).
- Spec: `openspec/specs/harness/spec.md:434` Requirement "Merge gate and CI workflow enforcement", scenarios at `:438-489`.

## Goals / Non-Goals

Goals: `Merge check` re-runs when its inputs change at the PR head; a flake issue no longer reds `Required`; no manual rerun in the common review path. Non-goals: see `proposal.md`.

## Decisions

### D1. Where the job runs

- **A (recommended). Own workflow `merge-check.yml`**, `Required` keeps only `Verify`. Cost: one new file per forge, one pin test, one ruleset context. Forecloses nothing.
- **B. Add `edited` to `ci.yml`.** No new file, but every description edit cancels and reruns the 15-minute `Verify`. Rejected on cost.
- **C. Status quo, document the rerun.** Zero code; the red check and flake coupling stay.

Adopter seam: none; adopters do not copy ateam's workflows (`docs/adoption.md:77`).

### D2. Event: `pull_request` or `pull_request_target`

- **A (recommended). `pull_request`, base-commit checkout as today.** The introducing PR runs its own new workflow, so the `Merge check` context exists at its head and no merge goes ungated. A PR could edit `merge-check.yml`, but that is a gate file and fails `Gate files` without the owner label. Gitea context: `Merge check / Merge check (pull_request)`.
- **B. `pull_request_target`, like `gate-files.yml`.** The workflow definition also comes from base, closing the edit-the-workflow path without relying on the label. Cost: the introducing PR emits no `Merge check` (base has no file), so it merges ungated or by owner bypass; the token defaults to write and must be pinned read.

### D3. What the rule check requires (adopter-visible)

- **A. `Required` only (status quo).** No adopter change. If the ruleset drops `Merge check`, nothing notices: an unreviewed PR merges with `Required` green.
- **B. `Required` and `Merge check` always.** This repo is self-checking, but every adopter that runs `ateam review gate` inside its own `Required` aggregate starts failing on upgrade.
- **C (recommended). A list from the workflow env, default `Required`.** `merge-check.yml` sets `ATEAM_REQUIRED_CHECKS: Required,Merge check` (Gitea: the full contexts); the gate requires each one. Adopters who set nothing see no change; this repo detects a ruleset that drops `Merge check`. Cost: one env var in the spec and `ateam review gate --help`.

### D4. Triggers and concurrency

`pull_request: [opened, synchronize, reopened, ready_for_review, edited]`. Concurrency `merge-check-${{ github.event.pull_request.number }}`, cancel-in-progress (about 30s per run; the latest event wins). Recommended to omit `labeled`/`unlabeled` (#255 listed them): no PR label is a gate input. Drop the `push: main` run: on `main` it only re-reads the ruleset, and `task merge:check` does that locally.

### D5. A re-review whose description does not change

A second review at a new head reuses the `reviewed-by:` line, so no `edited` event fires and `Merge check` stays red from the push.

- **A (recommended).** Step 5 posts the record first, then edits the description (the edit is the trigger). If the description is already right, it reruns the latest `merge-check.yml` run (one job, no `--failed` filter needed).
- **B.** The reviewer always rewrites the line (e.g. `Reviewed at <sha>` in the body). That is a record-format change, which is a non-goal.

## Risks / Trade-offs

- A cancelled earlier run and a later one share the check name at one head; GitHub reads the latest check run per name, so the cancel is cosmetic.
- PRs opened before the merge get the new workflows from the merge ref on their next event.
- Gitea fires `edited` on any body edit (accepted on #257); the merge check is cheap.

## Migration Plan

1. Rebase on `main` once PR #258 lands (it rewrites `.gitea/workflows/ci.yml`).
2. Land the workflows, the gate change, the docs, and the tests. On the PR, confirm that `Merge check` comes from `merge-check.yml`.
3. The owner adds `Merge check` to the `main` ruleset (it can be selected now). On the Gitea test instance, a live probe records the new context on a throwaway PR, then the test repo is restored.

## Open Questions

Resolved by the owner on 2026-10-29: D2 A (`pull_request`), D3 C (env list, default `Required`), D4 omit `labeled`/`unlabeled`.
