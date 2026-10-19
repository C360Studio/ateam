# Gemini A/A pilot: startup evidence, 2026-10-19

The pilot has **no evaluable pairs or quality result**. The first launch exposed a missing workspace-trust setting. After the user approved session-only trust and the adapter was fixed, Google's API returned repeated HTTP 503 high-demand errors. The first attempt reached its 120-second limit; the runner retained the timeout and stopped before another benchmark attempt. A later canary received a response after one 503 retry, but exposed a fenced-JSON output mismatch and incomplete accounting. Native schema output and a clean, fully accounted canary are required before the full A/A pilot.

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
| User-requested capacity retry | 1 / 30 | Timeout after 120.018 seconds | Six explicit 503 retry messages in stderr; no stdout report | 0 |
| Post-Luna capacity canary | 1 / 2 | Adapter error after 16.102 seconds | Fenced JSON; three native API requests with one error; partial usage | 0 |

These are separate startup runs, not combined A/A samples. The third launch reused the same config and runner hashes below, again stopped after the first attempt, and produced no evaluable pairs. The second run's provider errors establish a service-availability failure, not a task-quality regression. Six logged retry failures are observed stderr evidence, not an authoritative billing/request ledger. No further native attempt or model fallback was launched after the runner stopped. The original three runs have stop reason `model-unverified` because model identity was unavailable; the terminal attempt status and stderr retain the underlying cause.

The local environment also reported missing ripgrep and a GrepTool fallback. No tool execution was observed; make tool availability consistent before interpreting later tool-heavy comparisons.

## Retained evidence

Raw artifacts below are deliberately gitignored; paths are relative to the repository and available on the pilot machine. Hashes identify the exact originals. No raw trace or credential is published with this report.

- Initial run: `benchmarks/runs/experiment-4112803625/run.json`; SHA-256 `091c2f4eb7ed240923bc59c69d66733e98e87dd5708e9a0cc6ea02def805d56a`.
- Scoped-trust run: `benchmarks/runs/experiment-2084156922/run.json`; SHA-256 `42cac56e87601df713b17b1b816011bb05d1253579dea8962ddc953de26bc11f`.
- Scoped-trust rescore: `benchmarks/runs/experiment-2084156922/rescore-4243026740/run.json`. Offline replay preserved the timeout/stop reason and original run bytes, with no model calls.
- Capacity retry: `benchmarks/runs/experiment-2759323729/run.json`; SHA-256 `430c300cd87e07c532dc583b2ebf20bd67ff6bf2c120ff2da6bfc10b8301d1a5`. Offline replay at `rescore-674700380/run.json` preserved the timeout and original bytes.
- Config SHA-256: `9a225cb9162fbbed35d9313687c3c9adf0fc4d2b237238e80fd100ce7496adb6`.
- Runner SHA-256: `382ae2e4fb808cb6a7066743f42f90ef5daf14fd21a79db873b2585435d2b757`.
- Evaluator SHA-256: `1fc34706a7149b708d56f51371cef95f71eb1ffa985bac5c0c324a4095398c39`.
- Offline self-test preceding retry: `benchmarks/runs/selftest-20261019T152101.256603000Z-3643892949.json`, passed.
- Offline comparison: zero pairs, one baseline timeout, no candidate attempt, missing usage. Successful timing/token comparisons are absent.

## Fix verification and remaining work

The trust regression failed before the fix. Fake-child tests confirm explicit session trust, unchanged parent environment, fresh sessions, credential redaction and retained managed-policy/tool restrictions. Removing the child trust assignment through a Go overlay was detected; baseline/after tests passed and the original source hash remained unchanged. Independent read-only review approved the fix. Full clean-tree `task verify` passed at `d6470a4537d6d971a0a2701f41ce3e2272986032`; doctor reported parity passed and 177 schema tokens. Its single session-start timing was 5,076 microseconds, above the advertised 5 ms target while below doctor's current 50 ms threshold; no latency guarantee is established.

Retry the same fixed-model configuration when provider capacity is available, retain the existing failures, and reconcile available provider usage before treating calibration as complete. Do not switch models silently, count service failures as task-quality regressions, or claim harness improvements from these startup runs.

The separate [Haiku](haiku-calibration-pilot-2026-10-19.md) and [Luna](luna-calibration-pilot-2026-10-19.md) calibrations are complete. Their results are not pooled with these Gemini startup failures.

## Post-Luna capacity canary

The user requested another Gemini retry while PR #13 was under review. The unchanged verified runner binary (SHA-256 `bde3b34628dcc603c5e417a7a016a3d2ed3d050f4b81193caba7752ffe053110`) scheduled only one safe-path pair, preserving Gemini CLI 0.43.0, gemini-3.8-flash, low reasoning, API-only auth and the original 120-second/100,000-token limits. Its offline self-test passed first.

Google returned one explicit 503 before the CLI produced a native terminal report. The report identified gemini-3.8-flash and one successful list_directory call; generated source was wrapped in Markdown JSON fences. Strict parsing rejected it rather than repairing or grading the response. Native counters reported 12,973 prompt and 564 candidate tokens (13,537 total), zero cache/thought/tool tokens and three API requests including one error. The failed request's usage is unknown, so normalized usage remained partial/unknown and the runner stopped with `unknown-usage` after the first attempt. No full pilot or model fallback followed this artifact.

- Artifact: `benchmarks/runs/gemini-capacity-canary/experiment-1711254730/run.json`; SHA-256 `aceca6f2e4446b5cb326132ab1101678e8e315971dd4b5760898e729235e67e7`.
- Offline replay: `rescore-3159759801/run.json`; retains the adapter error without a model call.
- Native schema correction is tracked separately in `openspec/changes/gemini-structured-output/`; old configuration/evidence must stay replayable.

This is evidence that at least one request returned during the retry, not proof that provider capacity is stable. Reported token counts are partial observations, not a full bill or a quality result.
