# Luna A/A pilot, 2026-10-19

The API pilot completed **30/30 attempts: 26 passed and four failed the quality grader**, yielding 15 evaluable pairs with complete primary usage and 30 distinct sessions. Both identical arms passed 13/15. The four failures were generated Go files with an unused `strings` import; they are real retained quality outcomes, not adapter failures. This establishes a working calibration path, not a harness improvement or a model ranking.

## Configuration and execution

Codex CLI `0.160.1` ran through its native app-server stdio protocol using `OPENAI_API_KEY` from main's ignored local `.env`. Only that value was loaded into the isolated child environment; it was not copied into an auth file or published. The fixed custom provider used `https://api.openai.com/v1`, API authentication and zero transport retries. Subscription authentication and automatic model fallback were disabled.

Both arms requested `gpt-6-luna` with `low` reasoning and identical instructions: “Solve the task carefully and check boundary conditions.” The full schedule contains three tasks (`safe-path`, `sql-user`, `gitea-webhook`) × five repetitions × two arms, counterbalanced with seed 42. Each attempt uses a fresh ephemeral thread, home, configuration and workspace. Native schema output requires one `source` property. Host instruction discovery and the four bundled skills are suppressed; shell/external capabilities are disabled in configuration, and observed unexpected tools are refused. This does not prove the complete native tool-definition inventory is empty.

Native `thread/start` reported the requested model, provider and effort, approval policy `never`, read-only sandbox without network access, and no instruction-file sources. These are **CLI-resolved settings**, not backend model attestation. Provider-reported model identity remains unknown. No routing, cross-model comparison, reasoning-level comparison or arbitrary skills/hooks were measured.

Implementation commit `64f3e3e0a6e78fab135f60cf93604cb69550ebe1` passed independent change review and clean-tree `task verify` before the successful canary and full pilot. The runner repeated offline evaluator self-tests before each paid launch. No test suite ran concurrently with live sampling.

## Outcomes and observed variation

| Measurement | Baseline (15 attempts) | Candidate (15 attempts) |
| --- | ---: | ---: |
| Passed | 13 | 13 |
| Primary usage reported | 15 | 15 |
| Mean agent execution, all attempts | 5.021 s | 4.270 s |
| Agent execution range | 1.892–12.112 s | 1.726–10.924 s |
| Execution population SD | 3.495 s | 2.801 s |
| Mean primary tokens | 7,730.067 | 7,675.133 |
| Token range | 7,474–8,230 | 7,474–8,286 |
| Token population SD | 298.042 | 245.359 |

All 15 pairs had usage and evaluable quality: 12 both passed, one both failed, one baseline-only pass and one candidate-only pass. The comparison labels the latter two as one regression and one improvement; identical instructions mean these are observed paired differences, not treatment effects. All four failures occurred on `gitea-webhook` (candidate repetition 2, both repetition 3, baseline repetition 5). No adapter, timeout or grading-infrastructure errors occurred.

Across the **12 pairs where both passed**, the candidate label averaged 0.961 seconds less agent execution. Across all 15 usage pairs, it used 824 fewer primary tokens. The all-attempt means in the table use a different denominator; fast failures are not counted as successful-pair speedups.

| Task (five attempts per arm) | Baseline mean agent time | Candidate mean agent time | Baseline total tokens | Candidate total tokens |
| --- | ---: | ---: | ---: | ---: |
| safe-path | 9.850 s | 7.737 s | 40,718 | 39,891 |
| sql-user | 2.378 s | 2.175 s | 37,373 | 37,374 |
| gitea-webhook | 2.836 s | 2.899 s | 37,860 | 37,862 |

Total agent execution was 139.369 seconds; grading took another 10.133 seconds. Native totals were 221,290 input and 9,788 output tokens, including 221,200 cache-write input tokens, zero cached-read tokens and 3,808 reasoning-output tokens. Every attempt reported these breakdowns. The only observed item types were user messages, reasoning and agent messages.

Identical treatments can produce different timing, token counts and grades. These observations calibrate this small, mixed-task sample; they are not an equivalence test, significance threshold or evidence that a harness change improved performance. Agent execution includes CLI startup and model interaction; evaluator grading and preceding self-tests are recorded separately. Fresh local sessions do not disable provider prompt caching.

## Canary corrections and spending policy

The first canary stopped on a legitimate thread-scoped warning that the parser incorrectly treated as a turn event. Its turn request had already been submitted; usage and cost are unknown. The second canary reached a completed schema message and a raw snapshot of 7,374 input plus 809 output tokens, then stopped when an account-scoped rate-limit diagnostic lacked a thread ID. It retained **8,183 raw tokens**, but normalized usage remained unknown without terminal completion. Neither failed run supplies an evaluable pair or quality result.

Both parser defects were reproduced with failing regressions and corrected narrowly. Warning and rate-limit diagnostics stay in raw output; they cannot establish model identity, authentication, token usage or billing. Mutations detected removal of their dispatch and validation guards. Each correction passed independent review and full verification before retry. Original failed artifacts remained byte-for-byte unchanged and replayable.

A separate successful canary completed **2/2 passing attempts**, one pair, with 16,281 primary tokens. Its offline replay preserved grades and comparison exactly. Canary results are excluded from the full-pilot sample.

The user selected a **$2 target with token/time limits**, not a hard monetary cap. Observed native context used about 8,000 tokens per attempt, so the local full-pilot configuration revised the example's 100,000-token threshold to **275,536**: a 300,000-token planning reserve less the 8,183 known failed-canary snapshot and 16,281 successful-canary tokens. The first interrupted request remains unknown. Limits stayed at 120 seconds per attempt, one hour overall and at most 30 attempts. Token checks happen between attempts and can overshoot on the final request; no verified native output-token or model-turn cap exists. The monitor interrupts on retained adapter/accounting failures, with a possible following request already in flight.

The full pilot reported **231,078 primary tokens**. Across all Luna launches, the successful canary plus full pilot account for 247,359 complete primary tokens; adding the second failed canary's raw snapshot gives **255,542 known observed tokens**, plus unknown first-canary usage and any unreported continuation after cancellation. None of these sums is a reconciled bill.

Applying the listed Standard short-context rates to the full pilot's observed counters gives an **illustrative $0.032553**: 90 uncached non-write input tokens, 221,200 cache-write tokens and 9,788 output tokens. This is a price calculation, not a native cost report or verified invoice; service-tier/billing attribution remains unknown.

For context, the [official Luna model page](https://developers.openai.com/api/docs/models/gpt-6-luna), checked 2026-10-19, lists Standard short-context rates per million tokens of $0.10 uncached input, $0.01 cached input, $0.125 cache writes and $0.50 output. Longer context and service tiers change rates. Native traces do not provide reconciled charges, so these rates do not establish actual billing or a hard spending ceiling. Cache and reasoning counters are subsets of input/output, not extra primary tokens.

## Immutable evidence and replay

Raw traces, exact local pilot configuration and monitor logs remain ignored. No keys, local host identifiers or raw private context are published. Paths below identify local retained originals.

- Full pilot: `benchmarks/runs/luna-pilot/experiment-990414760/run.json`; SHA-256 `ce5fcbe681d2279fd66c999ea38e0e8a64f7759e1454976f72dc00626c428873`.
- Offline replay: `rescore-4028442914/run.json` beneath that directory. All 26 passes and four quality failures remained; comparison JSON was identical and the original hash unchanged. No adapter/model calls occurred.
- Config SHA-256: `084ff94b2d7d0f1d506d98ce119d3a1d4a44dfb5e8d4f915401207b602697ee3`. The ignored local copy `.enjoy-logs/luna-pilot.json` differs from tracked `benchmarks/examples/codex-aa.json` only in `max_total_tokens` (275,536); exact configuration is also inside the artifact.
- Runner SHA-256: `bde3b34628dcc603c5e417a7a016a3d2ed3d050f4b81193caba7752ffe053110`.
- Codex executable SHA-256: `09fa44fdc37a5fc70dc1ace31235f90468a2e193d0e85f7552eab068ea2582be`.
- Evaluator SHA-256: `1fc34706a7149b708d56f51371cef95f71eb1ffa985bac5c0c324a4095398c39`.
- A sanitized successful native trace and fixed-counter test are tracked under `internal/bench/testdata/codex/` and `TestCodexRecordedAppServerSuccess`. Removed streaming deltas and replaced sensitive/content fields are explicitly documented; it is a protocol fixture, not a quality or latency sample.

- First failed canary: `benchmarks/runs/luna-canary/experiment-4004755910/run.json`; SHA-256 `ceaf72fabf767e41c882b64b90da7c81741f458ac974eaf2160a468b14a48044`. Corrected-parser replay: `rescore-3929938127/run.json`; comparison identical, original unchanged.
- Second failed canary: `benchmarks/runs/luna-canary-retry/experiment-4000768275/run.json`; SHA-256 `e6f9b702bfddbd2c4de975e9e7ea156e08d718043dd1c422f6f1dc774f0dd97e`. Corrected-parser replay: `rescore-1312252571/run.json`; comparison identical, original unchanged.
- Successful canary: `benchmarks/runs/luna-canary-final/experiment-958790448/run.json`; SHA-256 `350a291f1936bcef96d3f6b7a32917dfacb5213498e3e27742e409813fd0e9b0`. Replay: `rescore-850335198/run.json`; comparison identical, original unchanged.
- Local verification and monitoring: `.enjoy-logs/luna-rate-verify.log`, `luna-canary-final.log`, `luna-pilot.log`, `luna-pilot-policy.json`.

## Limits and next step

Luna and Haiku samples retain separate native provenance; they do not support a controlled cross-model ranking. Gemini's capacity failures remain separate. The next useful experiment changes one harness instruction or setting while holding model, reasoning and task schedule fixed. Reasoning, model selection and role routing combinations remain post-MVP work in issue #16.

The native doctor check retains its existing threshold; it is now executed without race instrumentation because the first implementation verification reproduced an instrumentation-induced latency failure. Production hook behavior and thresholds were unchanged. No general hook-latency guarantee, authoritative billing, backend model attestation, broad quality improvement or cross-platform native-policy support is claimed.
