# Decision-opportunity benchmark tier

## Why

Issue [#111](https://github.com/C360Studio/ateam/issues/111), split from #95 under #91 and roadmap #16. The three prescribed experiment tasks (`safe-path`, `sql-user`, `gitea-webhook`) fix package, signature, behavior and in two cases the mechanism. Every qualified model saturates them (Haiku 30/30, Mercury 12/12 in both A/B arms, Luna 26/30), so an instruction treatment has nothing to improve. A discipline treatment can only show a difference on tasks where a naive solution passes the behavior tests but reuses nothing, overbuilds, or leaves a dependent value stale.

## What Changes

- Add three built-in tasks with tier `decision-opportunity`: `reuse-email` (reuse an existing package helper), `overbuild-slug` (a small function that invites options, structs or wrappers) and `propagate-limit` (raise a limit and update the error message that depends on it).
- Tasks may carry a pre-existing package file. The prompt shows it verbatim and says it already exists and must not be returned; the grader writes it beside the returned file. The adapter contract (one returned source file) is unchanged.
- A stdlib `go/ast` discipline gate grades the tier. Attempt grades and self-test task results record behavior and discipline as separate fields; `passed` for the tier requires both. New fields are omitted for existing tasks.
- `compare` reports source bytes per arm over both-passed pairs, descriptively.
- Dry-run Mercury examples `decision-aa.json` / `decision-ab.json` and an offline reference-adapter bundles variant.

## Scope and stop boundary

Implementation in `internal/bench`, `benchmarks/` and this change on `claude/bench-decision-tasks`; delivery stops at a draft PR. The per-experiment task limit stays at 3: the tier has exactly three tasks and runs as its own experiment, which needs no code change.

Non-goals: mode=auto, classifiers or hooks (#94); 3-way evaluation or any live run (#95); new adapters, runtimes or UI; changes to the existing tasks' prompts, tests or grading. Offline only: no paid or model calls are authorized. The tier is uncalibrated; A/A calibration is a separate authorized run.
