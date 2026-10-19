## Context
The existing suite evaluates fixed reference sources, not agents. We retain it as an offline evaluator check and add a small command adapter contract. Scope includes bench callers/tests/fixtures, CLI parsing, npm tool lockfile, Taskfile, and new benchmark documentation; UI and README belong to other sessions.

## Goals / Non-Goals
Deliver a reproducible offline end-to-end experiment before spending on real providers. A single configuration runs both arms on identical tasks with deterministic counterbalanced ordering. Stop after verified implementation and draft PR; do not merge or call paid agents. No leaderboard, statistical significance claim, generalized sandbox, native provider adapter, or arbitrary repository-task importer in this slice.

## Decisions
- JSON v1 uses Go stdlib strict decoding. One explicit command argv, agent/model/version labels, baseline/candidate instruction strings, built-in task IDs, repetitions, order seed, and per-attempt timeout. Resolve executables/arguments explicitly; no shell interpolation. Dry run performs no adapter execution.
- The command receives a versioned JSON request on stdin and emits a single JSON response with candidate Go source and optional reported token usage. Raw stdout/stderr and the exact request are retained. Adapter usage is self-reported, never inferred from output length. Activation of external tools is unverified; no causal attribution is made from configuration alone.
- Fresh per-attempt temporary working directory and HOME, explicit minimal environment, no automatic parent credentials/config. This is configuration isolation, not a security sandbox: trusted local command adapters and generated code execute with OS permissions; use external containers for untrusted workloads. Launches are bounded and process descendants terminated where supported.
- Independent grader runs task-owned Go tests against the submitted source, outside the adapter's workspace. Keep missing usage null and classify passed/failed/timeout/adapter-error/grading-error distinctly. Grading does not use mutation reach as a quality oracle. Retain source, raw output, grade diagnostics, timing and hashes.
- Each run has an exclusive unique directory, resolved config, task/evaluator digest, adapter executable digest, Go/platform provenance, original experiment digest and trial schedule. Artifacts are immutable by convention with exclusive writes; digest checks detect corruption, not malicious forgery.
- Rescore reads retained candidates, verifies metadata and hashes, reruns the current matching evaluator, and writes a new result artifact without modifying the run. Reject mismatched evaluator versions instead of quietly mixing grades.
- Compare reports all attempted pairs and quality changes; timing comparisons use pairs where both passed and disclose that denominator. Missing usage is excluded with its count. No cost estimate without price data, no significance or equivalence claim. Initial reports explicitly describe a pilot, not publication evidence.
- Local OpenSpec is pinned with npm lockfile and Taskfile prefers local bin. Do not regenerate the repository's shared skill adapters.

## Risks / Trade-offs
Task set is small and Go-only. Adapter labels and usage need provider-native validation later. No filesystem isolation can stop malicious code without an external sandbox. Wall-clock comparisons remain sensitive to machine load and provider caching. Narrow examples are not representative agent-productivity evidence. No all-in-one score hides quality regressions behind token savings.

## Verification
Failing regressions precede evaluator fixes. Offline fake command covers request/response, failures, deadlines, missing usage, two arms, artifacts, and rescore. Strict config/artifact tests cover invalid JSON, duplicate/unknown tasks, invalid limits, digest mismatch and incomplete trial histories. Generated JSON decode checks exercise panic/validation boundaries; deterministic schedule tests cover pairing/order. Mutation checks target acceptance/refusal and comparison logic. Run Taskfile verify on a clean commit, doctor, and bench selftest; preserve exact evidence in verification.md.
