# Design: gate-read-retry-441

## Context

- `scripts/gate-files.sh:52-55` (GitHub) and `:34-50` (Gitea, `curl -fsS --max-time 30`) define four reads: pull, files, labels, comments. Each caller (`:58`, `:62`, `:83`, `:97`) exits 2 on the first failed read. No read there has a failure that means an answer.
- `internal/review/gate_github.go:24-33`: `ghSource.gh` runs one `gh` call and returns its stderr in the error. Every GitHub read goes through it (`ghRead :67`, `ghList :83`, `commit :375`, `headPassed :550`). `fileAt :532-534` reads `(HTTP 404)` in that error as "no such file" (the proposal.md read of `approvalNotes`, `gate.go:414-421`).
- `internal/review/gate_gitea.go:32-38`: `giteaSource.get` wraps `readForgePage` (`forge.go:216-260`, 10 s per request, typed `forgeStatusError` for non-200). Every Gitea read goes through it. `upToDate :145` reads 403/404 on branch protection as "hidden from this token"; `fileAt :426` reads 404 as "no such file". A decode error or an oversized body also comes back from `readForgePage` as an error.
- `gate.go:144-167`: the first error that is not a `*gateExit` ends the run as `unavailable`, exit 2. So one transient read fails the whole run.
- `.github/workflows/merge-check.yml:61-64`, `.gitea/workflows/merge-check.yml:58-61`: `out=$("$ATEAM" review gate --awaiting …)` under the default `bash -e` ends the step before `printf`. The `Merge check` job's `merge-check.sh` uses `exec`, so its output already reaches the log.
- All three jobs have `timeout-minutes: 5` (`merge-check.yml:33,77`; `gate-files.yml:32`).
- Evidence (#441, 2026-11-27): PR #440 `Gate files` attempt 1 at 15:47Z, HTTP 500 "diff temporarily unavailable"; PR #436 at 15:57Z, same; PR #437 run 46030398924/8 created 16:04:08Z, files API returning 500 until 16:11:36Z. Each rerun passed. PR #434 probe exit 2 with no output; cause unknown.
- Tests: `internal/bootstrap/gatefiles_test.go:14-25` fakes `gh` with a `<key>.fail` file (always fails); `internal/review/gate_test.go:131-141` fakes `gh` with an `.err` file (always fails). `mergecheck_test.go:905` requires the probe step to contain `"$ATEAM" review gate --awaiting "$PR_NUMBER"` and `>> "$GITHUB_OUTPUT"`.

## Goals / Non-Goals

**Goals:** a forge read that fails briefly is tried again before the gate reports unavailable; a read that keeps failing still exits 2; a failed probe prints its cause.

**Non-Goals:** retrying wrong-shape answers; `readForgePage`'s callers outside the merge gate; longer job timeouts; changing exit codes, check names or verdicts.

## Decisions

### D1. Which failures are retried

- **A: every failed read, everywhere.** Least code. But the Go gate reads some 4xx answers as answers (`gate_github.go:532`, `gate_gitea.go:145,426`); retrying those adds a minute to every run that hits one, which a Gitea token without branch-protection access does on every run.
- **B (recommended): the shell script retries every failed read; the Go gate retries unless the forge answered 4xx.** For `gh`, "answered 4xx" is `(HTTP 4dd)` in its stderr, the same text `fileAt` already reads. For Gitea, it is a `forgeStatusError` below 500; a request that got no response (a `*url.Error`, timeouts included) or a 5xx is retried. Decode and size errors are wrong-shape answers and are not retried. The shell script has no read whose failure is an answer, so classifying there buys nothing; a bad token costs it one extra minute before the same exit 2.
- **C: only 5xx, both places.** Misses connection resets and timeouts, the other common transient. In the shell it needs stderr parsing per forge.
- *Adopter seam:* none to act on. Adopters run these files from their own copy; a team that never copies the new versions keeps today's single-try behaviour.

### D2. The schedule

- **A (recommended): 4 attempts, waiting 5 s, 15 s, then 40 s (60 s worst case per read).** The first read that exhausts its attempts ends the run, so a whole job waits at most about a minute plus request time, inside the 5-minute timeout. Covers a blip like #440's; does not cover #437's seven-minute outage, which then needs a rerun as today.
- **B: about 3 minutes (e.g. 10 s, 30 s, 60 s, 90 s).** Would have ridden out more of #437's outage, but sits near the job timeout once request time (gh has none; `curl --max-time 30`, Go 10 s) is added. A timed-out job is still red, but with no "unavailable" line.
- **C: raise `timeout-minutes` and wait longer.** Holds runners through an outage on every open PR at once; the event that caused this was load.
- Waits are not jittered: each job's reads start at its own time.
- *Adopter seam:* only the wait before a red check changes.

### D3. Where the Go retry lives

- **A (recommended): inside `ghSource.gh` and `giteaSource.get`.** Two functions cover every gate read on each forge, the `Merge check` and the probe alike. The schedule is one package variable that tests set to zero waits. Waits honour `ctx`.
- **B: inside `readForgePage`.** Also changes `ReadPullRequest`/`ReadComment` (review evidence for `ateam review check`), and does not cover GitHub, which reads through `gh`.
- **C: re-run the whole gate in the workflow step.** Repeats every read, including ones that succeeded, and needs the same 4xx distinction from exit codes the gate does not expose.

### D4. The probe's output

- **A (recommended):** `out=$(…) || { rc=$?; printf '%s\n' "$out"; exit "$rc"; }` in both workflows. Exit status and the "no answer means the full check runs" rule (`merge-check.yml:73`) are unchanged; `mergecheck_test.go:905` still matches.
- **B: `set +e` around the step.** Same effect, easier to get wrong when the step grows.

## Risks / Trade-offs

- A red `Gate files` or `Merge check` now takes up to a minute longer to appear during an outage.
- The shell script reads its waits from `ATEAM_GATE_RETRY_WAITS` (default `5 15 40`) so its tests run without sleeping. The workflows do not set it, and a PR that edits the script is judged by its own copy anyway (`gate-files.sh:1-5`), so the variable gives a PR nothing it does not already have. The attempt count is the number of waits plus one.
- The probe's race hypothesis on #434 (check runs being created as it reads) is a shape mismatch and stays unretried; D4 makes the next occurrence name its read.

## Migration Plan

None. `Gate files` runs the PR's own script, so this PR exercises its retry on its own CI. `Merge check` and the probe build from the base commit, so the Go retry takes effect on PRs opened or rerun after this one merges.

## Open Questions

- D1–D4 need owner rulings: approving the proposal accepts the recommended options; naming another option changes the spec delta and tasks before implementation.
