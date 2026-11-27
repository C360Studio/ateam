# Proposal: gate-read-retry-441
Approved-by:

## Track

`design`: it edits gate files (`scripts/gate-files.sh`, `internal/review/`, both `merge-check.yml` workflows) and adds a spec Requirement. The PR then also needs the owner's `gate-change-approved`.

## Decisions you are making by approving

- When the forge fails to answer one of a gate's reads, `Gate files`, `Merge check` and the `Merge check probe` try that read again a few more times, waiting a little longer each time, before reporting "unavailable". A short GitHub outage no longer turns a required check red on a PR it has nothing against.
- They still fail closed: if the read keeps failing, the check exits 2 exactly as today, about a minute later. A longer outage (PR #437's lasted about seven minutes) still needs a rerun.
- `Gate files` retries any failed read. `Merge check` and the probe retry everything except an answer the forge gave on purpose (a "not found" or "forbidden"), because the gate reads some of those as answers, for example "this token may not read the branch protection" on Gitea.
- An answer that arrives but has the wrong shape (a short file list, a count that does not match) is not retried; it stays "unavailable" at once.
- When the probe fails, its log now shows the gate's error line, so the next failure names the read that failed instead of exiting with no output.

## Why

On 2026-11-27, with CI running on five PRs at once, GitHub answered the PR files API with HTTP 500 ("this diff is temporarily unavailable due to heavy server load"). `Gate files`, `Merge check` and the probe each exited 2 on the first failed read (#441; PR #436, PR #437, PR #440), and every PR needed a rerun plus an owner waiver to merge. The probe on PR #434 exited 2 with no output at all, because the step's `out=$(…)` under `bash -e` ends before printing.

## What Changes

1. `scripts/gate-files.sh`: each of its four reads runs through one bounded retry helper.
2. `internal/review/gate_github.go` and `gate_gitea.go`: the single read helper of each forge retries a failed read on the same schedule, skipping 4xx answers.
3. `.github/workflows/merge-check.yml` and `.gitea/workflows/merge-check.yml`: the probe step prints the gate's output before exiting with its status.
4. Harness spec: one new Requirement, "Gate reads retry transient forge failures".

## Scope & Non-Goals

- **In-bounds:** the five files above, their tests (`internal/review/gate_test.go`, `internal/bootstrap/gatefiles_test.go`), the harness spec.
- **Non-goals:** retrying shape mismatches (the #434 race hypothesis); `readForgePage` callers outside the merge gate (`ateam review check`, review evidence); longer job timeouts; any change to exit codes, check names or what passes.

## Definition of Done

- Tests show a read failing twice then answering passes, one failing on every attempt exits 2 naming the read, and a 404 or 403 answer is read once (Go). Mutation checks detect removal of the retry and of the 4xx exclusion.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green; the change is archived before ready; `gate-change-approved` is the owner's.
