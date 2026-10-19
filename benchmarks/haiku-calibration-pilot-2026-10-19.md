# Haiku A/A pilot, 2026-10-19

The corrected API pilot completed **30/30 attempts, all passing**, yielding 15 evaluable pairs across the three Go tasks. Every attempt reported primary usage and a distinct session; native model accounting consistently identified `claude-haiku-5-5`. This establishes a working calibration path, not a harness improvement or statistical equivalence claim.

## Configuration and execution

- Claude Code `2.1.292`, API-only authentication using `ANTHROPIC_API_KEY` from the main ignored local `.env`, passed through the process environment without copying or committing the key. Bare mode prevented subscription fallback.
- Requested model `claude-haiku-5-5`, requested effort `low` in both arms. Native modelUsage reported the matching canonical model and firstParty provider. Resolved reasoning effort was not reported and remains unknown.
- Identical instruction text: “Solve the task carefully and check boundary conditions.” Three tasks (`safe-path`, `sql-user`, `gitea-webhook`) × five repetitions × two arms, counterbalanced schedule seed 42. Fresh homes/configurations/workspaces and no session persistence; all 30 session IDs differed.
- Native JSON schema required a single source property. All 30 replies used `structured_output` and ended with `stop_reason: tool_use`, as expected for the native StructuredOutput return tool. Configured filesystem tools were Read/Glob/Grep; their availability does not prove they were used. No arbitrary skills/hooks were measured.
- Isolated `claude doctor` preflight reported no remote managed settings configured for the organization and organization policy loaded from Anthropic. Local managed-policy refusal remained enabled; no policy bypass or global configuration changes were made.
- Implementation commit `4d473a424d3971f9c9b1ccb21b241041eb2c01a4` passed independent correction review and full clean-tree `task verify` before this run. The binary's offline evaluator self-test passed immediately before paid execution. No test suite ran concurrently with the live pilot.

## Outcomes and observed variation

| Measurement | Baseline (15 attempts) | Candidate (15 attempts) |
| --- | ---: | ---: |
| Passed | 15 | 15 |
| Primary usage reported | 15 | 15 |
| Mean agent execution | 4.274 s | 4.532 s |
| Agent execution range | 1.622–7.995 s | 1.723–9.042 s |
| Execution population SD | 1.837 s | 2.106 s |
| Mean primary tokens | 1,836.333 | 1,837.800 |
| Token range | 1,210–2,720 | 1,210–2,833 |
| Token population SD | 417.008 | 462.816 |

There were no improvements, regressions, failed attempts, unevaluable pairs or missing-usage pairs. The candidate label used 22 more primary tokens in total and averaged 0.257 seconds more agent execution per successful pair (about 6.0% of baseline). Identical treatments can still differ: neither difference establishes an effect. These population SDs describe this small, mixed-task sample; they are not confidence intervals, an equivalence test or a reusable significance threshold. All-pass task quality also leaves a ceiling that can hide later quality gains.

| Task (five attempts per arm) | Baseline mean agent time | Candidate mean agent time | Baseline total tokens | Candidate total tokens |
| --- | ---: | ---: | ---: | ---: |
| safe-path | 6.499 s | 6.977 s | 11,653 | 11,989 |
| sql-user | 2.553 s | 2.606 s | 7,134 | 6,896 |
| gitea-webhook | 3.771 s | 4.013 s | 8,758 | 8,682 |

Total agent execution was 132.096 seconds; independent grading took another 12.512 seconds. These exclude the preceding self-test and do not constitute a cold-start CLI/end-to-end wall-time claim. Primary input includes uncached, cache-creation and cache-read tokens once; output already includes thinking. Cache/thinking breakdowns were present for all attempts and are not added again. Fresh local sessions do not disable provider-side prompt caching; first-request/cache state and task mix can affect observations.

## Budget and earlier failed launch

The user approved a **$2 target with token/time bounds**, not a hard monetary cap. The corrected run scheduled at most 30 attempts with a reduced 75,000 reported-token threshold, 120 seconds per attempt and one hour overall. The reduction reserved the 23,075 known tokens already used by the first launch. A local monitor would interrupt on adapter/accounting failure or $1.40 of additional CLI-estimated cost; it did not trigger. That monitor polls completed artifacts, so a following request may already be in flight. The adapter requested four agentic turns, 8,192 output tokens per request, zero transport retries and `MAX_STRUCTURED_OUTPUT_RETRIES=1` (one structured-output attempt).

The corrected run reported **55,112 primary tokens and $0.5755378 CLI-estimated cost**. Its native `costBasis` is `unknown`: the estimate is not reconciled billing, and CLI model metadata reported a 32,000-token maximum rather than proving the requested per-request cap. No authoritative dollar ceiling or actual invoice amount is claimed.

The first launch used prose-only JSON instructions. Eleven completed native successes returned Markdown-fenced JSON and were correctly rejected by the text adapter; a twelfth attempt was interrupted when the defect was diagnosed. That launch yielded zero evaluable pairs, 23,075 known tokens and $0.288833 CLI estimate; interrupted usage remains unknown. We fixed the transport contract with native schema output, rather than stripping fences or reclassifying old failures. The successful retry is a separate sample.

Across both launches the known totals are **78,187 primary tokens and $0.8643708 CLI estimate**, plus unknown interrupted usage/cost. These observations are below the target on the CLI estimate only; provider billing has not been reconciled. Gemini capacity failures remain separately recorded and are not pooled with Haiku.

## Immutable evidence and replay

Raw traces and local policy/monitor files remain ignored and are not published. Paths are relative to this worktree on the pilot machine; hashes identify the originals.

- Successful artifact: `benchmarks/runs/haiku-schema-pilot/experiment-3799230470/run.json`; SHA-256 `9c68c17cfcff2c1673266f0d7546b26bdf37cc68f4e22d593eabaf549803f0c8`.
- Offline replay: `benchmarks/runs/haiku-schema-pilot/experiment-3799230470/rescore-2276432059/run.json`. All 30 grades remained passing, comparison JSON was identical, and original bytes/hash were unchanged. Replay made no model calls.
- Config SHA-256: `f2ed79d6eac2e3df9a0d7558093c52e4531a5edc4ba8aa915fbe55e12c745683`. Config is the tracked `benchmarks/examples/claude-aa.json` with only `max_total_tokens` reduced from 100,000 to 75,000; retained at `.enjoy-logs/haiku-schema-aa.json` and inside the artifact.
- Runner SHA-256: `ef5dfa68e6ff87123de3335f80cc4a742113095f96cd7ae607038c58a4ce211c`.
- Claude executable SHA-256: `97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f`.
- Evaluator SHA-256: `1fc34706a7149b708d56f51371cef95f71eb1ffa985bac5c0c324a4095398c39`.
- Immediately preceding self-test: `benchmarks/runs/selftest-20261019T162248.664278000Z-1065219932.json`.
- First failed launch: `benchmarks/runs/experiment-3832828402/run.json`; SHA-256 `6fd3438dd69e682a8c75fd03088b6442a3f0fa0fa85e68f6b9275e237e4f8465`. Its offline replay retains all adapter errors and the interruption; the new format is required for new execution, while legacy artifacts remain replay-only.
- Local verification/policy evidence: `.enjoy-logs/haiku-schema-verify.log`, `haiku-policy-preflight-3.json`, `haiku-schema-live-policy.json`, `haiku-schema-live-aa.log`, `haiku-schema-comparison.json`, `haiku-schema-replay-comparison.json`.

## Limits and next step

Meaningful RED regressions and detected mutations cover missing schema arguments, text fallback, terminal handling, exact property names, usage accounting and model refusal. Doctor parity passed with 177 schema tokens; its single hook observation was 6,008 microseconds, exceeding the advertised 5 ms target while below the current 50 ms doctor threshold. No hook-latency guarantee is established.

Haiku now supplies the first complete real-agent A/A pilot. Next, apply the same isolated API-only workflow to Codex Luna, retaining separate provenance and calibration. Later harness A/B runs should hold model/effort fixed and require their own approved spending limits. Reasoning, cross-model and routing combinations remain post-MVP dimensions in issue #16; this pilot does not measure them.
