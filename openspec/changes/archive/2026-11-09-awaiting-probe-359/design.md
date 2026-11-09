# Design: `awaiting-probe-359`

## Context

Opened in this session on `claude/awaiting-probe-359` at `main` 02a8537.

- `internal/review/gate.go:315-357` `awaiting()`: documents-only answers `false` (`:333-336`); a `reviewed-by:` line answers `false` (`:337-342`); any record comment by any agent, at any head, approving or not, answers `false` (`:347-354`); else `true`. The doc comment (`:315-320`) names the hole it guards.
- `gate.go:612-664` `record()` resolves the latest record's designated SHA against the full commit list (`gateDesignated`, `:903`) and its verdict (`gateVerdict`, `:958`); `awaiting()` reads neither.
- `gate.go:89-97` `gatePull.Head` is already read by `pullRequest()`; neither source reads check runs or commit statuses today (`gate_github.go`, `gate_gitea.go`: no `check-runs` or `statuses` path).
- `.github/workflows/merge-check.yml:29-62` the `awaiting` job (`Merge check probe`), scopes `contents`, `issues`, `pull-requests: read` (`:34-37`); `:72-74` the renamed skip; one queued run per pull request, `cancel-in-progress: false` (`:15`), so when the probe runs no earlier `Merge check` for the PR is in flight. `.gitea/workflows/merge-check.yml:34-37` the same; `:103` Gitea's context for the job is `Merge check / Merge check (pull_request)`.
- `openspec/specs/harness/spec.md:820` Requirement; `:844-847` awaiting scenario; `:849-854` probe scenario; `:854-857` line removed after an approval; `:864-867` rejection after an approval at the same head, delivered by a rerun.
- `internal/review/gate_test.go:396-430` `TestGateAwaiting` runs every row on both fake forges.

**The observation that decides this.** "Approving record at the head, no line yet" (rows 1-3 of #359) and "line removed after an approval at the same head" (the hole) are the same pull-request state: same head, same comments, same body. Nothing in the PR tells them apart. What differs is whether a passing `Merge check` already stands at that SHA. #359's proposed rule ("no approving record designates the head") cannot hold both of its own cases: by that rule the state in rows 1-3 runs in full and fails, so it removes only rows 4-5.

## Goals / Non-Goals

Goal: no red `Merge check` from a pull request that is merely waiting for its line, without letting a removed line keep a pass. Non-goals are in `proposal.md`.

## Decisions

**D1. What makes a no-line PR judgeable.**
- A) **Recommended: a standing pass at the head.** No line and no successful `Merge check` at `pull.Head` → `awaiting=true`; a standing pass → full run, which fails on the missing line and replaces the pass. Removes all five runs in #359's table. The hole stays closed: the only way a skip leaves a wrong answer is a pass to leave standing, and that case runs in full. The rejection-after-approval scenario (`spec.md:864`) still runs in full. Cost: one forge read per probe on each forge, ~40 lines plus tests, one workflow scope. Forecloses nothing in the full check.
- B) #359 as written: an approving record designating the head → full. Reuses `gateDesignated` and `gateVerdict`, no new read or scope. Removes rows 4-5 only (older-head record); rows 1-3 stay red, so every PR still emails once or twice.
- C) Read the reviewer from the approving record when the description has no line, making the line optional. Removes the window entirely but changes the merge-check rule, the protocol, the `AGENTS.md` row, and the record format's role. Out of proportion to the inbox problem; rejected unless the owner wants that policy change.
- Seam: an adopter sees fewer red runs and no new instruction. A PR whose line is never written stays blocked by the absent required context, exactly as an unreviewed PR is today.

**D2. Reading the head's result.**
- A) **Recommended:** GitHub `GET commits/<sha>/check-runs?check_name=Merge check` (needs `checks: read` on the `awaiting` job); take the latest run, pass only on `conclusion == success`. Gitea `GET commits/<sha>/statuses`, newest entry whose context equals the `:103` context, pass only on `success`. Exact names, so `Merge check probe` and the renamed skips never count.
- B) GitHub combined commit status: Actions jobs are check runs, not statuses, so it never sees `Merge check`. Rejected.
- Failure: any read or shape error exits non-zero with no `awaiting=` line; the workflow already treats a missing answer as "run in full" (`spec.md:849-854`). Fail-closed toward a red run, never toward a skip.

## Risks / Trade-offs

- A race: a description edit and a push in quick succession. Runs are queued per PR, so the probe reads after the earlier run finished; a pass at the old head does not count at the new head.
- A re-run of an old workflow run evaluates today's state at today's head, same as now.
- A green `Merge check` from a different workflow with the same job name would count. Only `merge-check.yml` defines it here; the Gitea context includes the workflow name.

## Migration Plan

Lands as one PR. The probe is built from the base commit (`merge-check.yml:83-85`), so this PR's own runs use the old probe; the live acceptance in task 2.3 needs a follow-up PR or a rerun after merge.

## Open Questions

1. **Owner ruling:** D1 A (standing pass) or B (#359 as written)? A changes tasks 2.1-2.3; B drops 2.1 and the workflow scope.
2. **Owner ruling:** Gitea job tokens and `GET /commits/<sha>/statuses`. If `contents: read` does not cover it, add the Gitea scope or let the Gitea probe stay on today's rule?
3. Is the live acceptance in task 2.3 required before this PR merges, or recorded on the next code PR after it? (Default: next PR, because of the base-commit build.)
