# Gemini A/A pilot: startup evidence, 2026-10-19

The pilot has **no evaluable pairs or quality result**. The first launch exposed a missing workspace-trust setting. After the user approved session-only trust and the adapter was fixed, Google's API returned repeated HTTP 503 high-demand errors. The first attempt reached its 120-second limit; the runner retained the timeout and stopped before another benchmark attempt. Provider availability must recover before A/A calibration can be completed.

## Configuration and authorization

- API-key authentication from the user's ignored local `.env`, loaded into the process environment without copying or committing the credential.
- User-approved $2 target with token/time bounds, not a hard dollar cap; maximum 30 scheduled attempts, 100,000 reported-token threshold, 120 seconds per attempt, one hour per run. Raw evidence remains local and ignored.
- Gemini CLI 0.43.0, requested model `gemini-3.8-flash`, requested reasoning `low`, identical A/A instructions, seed 42, three tasks × five repetitions × two arms.
- The retry used implementation commit `d6470a4537d6d971a0a2701f41ce3e2272986032`. Trust is literal `GEMINI_CLI_TRUST_WORKSPACE=true` in the generated workspace's isolated child environment only; no global trust records or parent environment were changed. Managed-policy refusal and restricted read-only tools remain enforced.
- Resolved model/reasoning and normalized usage are unknown: no complete native JSON report arrived. Cost is unknown, not zero; provider billing was not independently reconciled.

## Observations

| Launch | Retained attempts / scheduled | Terminal outcome | Native evidence | Evaluable pairs |
| --- | --- | --- | --- | --- |
| Before trust fix | 1 / 30 | Adapter error after 0.866 seconds | Exit 55: untrusted workspace | 0 |
| With approved scoped trust | 1 / 30 | Timeout after 120.007 seconds | Six explicit 503 retry messages in stderr; no stdout report | 0 |

These are separate startup runs, not combined A/A samples. The second run's provider errors establish a service-availability failure, not a task-quality regression. Six logged retry failures are observed stderr evidence, not an authoritative billing/request ledger. No further native attempt or model fallback was launched after the runner stopped. Both runs have stop reason `model-unverified` because model identity was unavailable; the terminal attempt status and stderr retain the underlying cause.

The local environment also reported missing ripgrep and a GrepTool fallback. No tool execution was observed; make tool availability consistent before interpreting later tool-heavy comparisons.

## Retained evidence

Raw artifacts below are deliberately gitignored; paths are relative to the repository and available on the pilot machine. Hashes identify the exact originals. No raw trace or credential is published with this report.

- Initial run: `benchmarks/runs/experiment-4112803625/run.json`; SHA-256 `091c2f4eb7ed240923bc59c69d66733e98e87dd5708e9a0cc6ea02def805d56a`.
- Scoped-trust run: `benchmarks/runs/experiment-2084156922/run.json`; SHA-256 `42cac56e87601df713b17b1b816011bb05d1253579dea8962ddc953de26bc11f`.
- Scoped-trust rescore: `benchmarks/runs/experiment-2084156922/rescore-4243026740/run.json`. Offline replay preserved the timeout/stop reason and original run bytes, with no model calls.
- Config SHA-256: `9a225cb9162fbbed35d9313687c3c9adf0fc4d2b237238e80fd100ce7496adb6`.
- Runner SHA-256: `382ae2e4fb808cb6a7066743f42f90ef5daf14fd21a79db873b2585435d2b757`.
- Evaluator SHA-256: `1fc34706a7149b708d56f51371cef95f71eb1ffa985bac5c0c324a4095398c39`.
- Offline self-test preceding retry: `benchmarks/runs/selftest-20261019T152101.256603000Z-3643892949.json`, passed.
- Offline comparison: zero pairs, one baseline timeout, no candidate attempt, missing usage. Successful timing/token comparisons are absent.

## Fix verification and remaining work

The trust regression failed before the fix. Fake-child tests confirm explicit session trust, unchanged parent environment, fresh sessions, credential redaction and retained managed-policy/tool restrictions. Removing the child trust assignment through a Go overlay was detected; baseline/after tests passed and the original source hash remained unchanged. Independent read-only review approved the fix. Full clean-tree `task verify` passed at `d6470a4537d6d971a0a2701f41ce3e2272986032`; doctor reported parity passed and 177 schema tokens. Its single session-start timing was 5,076 microseconds, above the advertised 5 ms target while below doctor's current 50 ms threshold; no latency guarantee is established.

Retry the same fixed-model configuration when provider capacity is available, retain the existing failures, and reconcile available provider usage before treating calibration as complete. Do not switch models silently, count service failures as task-quality regressions, or claim harness improvements from these startup runs.
