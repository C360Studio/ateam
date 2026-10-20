# Gemini instruction pilot v3 results

implemented-by: codex (reconciled adapters, frozen protocol and bounded live pilot)

## Outcome

The stable-CLI canary passed, A/A calibration completed, and A/B stopped at its sixth scheduled attempt on the configured four-turn limit. Every attempt is retained. No retry, fallback, repaired source, prompt amendment or sample expansion followed the failure.

| Phase | Retained / planned attempts | Graded passes | Quality failures | Native failures | Pairs / both passed / unevaluable | Known primary tokens |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Canary | 2 / 2 | 2 | 0 | 0 | 1 / 1 / 0 | 45,104 |
| A/A | 12 / 12 | 10 | 2 | 0 | 6 / 4 / 0 | 136,954 |
| A/B | 6 / 12 | 5 | 0 | 1 | 3 / 2 / 1 | 58,167 |

This is a completed bounded pilot with an incomplete A/B sample, not evidence that the instruction treatment improves agent performance. The comparison is minimal task guidance versus six ATeam engineering principles on the same requested model/reasoning. It is not the whole ATeam harness versus vanilla CLI. Canary, A/A, A/B and earlier v1/v2 cohorts are separate; their samples are not pooled for treatment comparisons.

## Frozen execution and checks

- Protocol/configuration/binary source: `0d4ec884e48948187a6c290ff365197c5daf6048`. Binary SHA-256: `ec92a225e9f7879d469aecf3f771bdbb5d50ebf05542828f2b8096347cb00f20`.
- Gemini CLI 0.64.0, requested `gemini-3.8-flash`, low reasoning, API-key-only authentication, fresh isolated homes/workspaces, four read-only tools, four turns, 32768 output allowance in both override scopes. Backend-resolved model/reasoning remain unknown. CLI entry SHA-256: `ab113d84c8f1692fd6c95081366718897876d8664328c7bf0ccf9437d688145f`.
- Full clean-tree `GOCACHE=/private/tmp/ateam-bench-audit-cache GOFLAGS=-p=1 task verify && task doctor` passed at implementation head `e811209b9340eebdade90fe08a5305825fc34499`, including Go race tests, offline benchmark self-test, strict specifications, attribution, lint and UI checks/build. Log: `.enjoy-logs/gemini-v3-preflight.log`. The final protocol commit changes only two README lines; formatting/spec/doctor checks and independent exact-head review passed before calls. Doctor parity passed; actual MCP connection was unchecked and observed process latency exceeded 5ms, so no sub-5ms claim is made.
- All three dry runs were retained before calls. Automatic evaluator self-tests passed before each phase. Verification workloads did not overlap inference sampling. Independent read-only reviews accepted the frozen implementation/protocol, canary gate and A/A gate.
- All original run bytes and comparisons survived offline rescore unchanged, including earlier v1/v2 artifacts under the combined implementation. No live call was needed for replay.

The [protocol](README.md) adds the same self-contained-task sentence to both arms and upgrades the CLI relative to v2. These simultaneous shared changes cannot be causally separated by cross-cohort observations. The candidate-only six-point text remains byte-identical to v2. Requested settings and local isolation do not prove backend enforcement or constitute an OS security sandbox.

## Canary and A/A acceptance

Both canary attempts passed with distinct sessions, expected reported model, normalized nonzero CLI usage and no reported API errors. Identical guidance produced 58.649s versus 5.237s: the first reported three API requests, two successful directory listings and 5698 thought tokens; the second reported one request, no tools and zero thoughts. This illustrates observed variability, not a treatment effect.

A/A retained all 12 attempts and six pairs, with five of six passes in each identical arm. Every attempt reported expected model/usage; no API errors or native/infrastructure failures were observed. Sessions were distinct across phases. Two discordant quality outcomes were inspected before A/B:

- Safe-path, candidate label, repetition 2: generated code cleaned a slash-prefixed relative path before containment checking, erasing parent traversal. The independent safety assertion failed. The paired baseline passed.
- Gitea webhook, baseline label, repetition 2: generated Go imports lacked quotes, so compilation failed with `fixture.go:4:8: missing import path`. The paired candidate passed. The decoded native source exactly matched the graded source; this was a source-quality failure, not a grader infrastructure failure.

Both failures were retained and reproduced. No unexplained measurement defect was found, so the predeclared A/A gate allowed A/B. Four of six pairs had both members pass; their mean candidate-minus-baseline execution time was -10.657s. All six pairs had known usage, with candidate-minus-baseline total +10,136 tokens. These differences between identical treatments do not establish equivalence or a stable noise floor.

| Task | A/A baseline passes / attempts | A/A candidate-label passes / attempts | A/B baseline passes / attempts | A/B ATeam passes / attempts |
| --- | ---: | ---: | ---: | ---: |
| safe-path | 2 / 2 | 1 / 2 | 1 / 1 | 1 / 1 |
| sql-user | 2 / 2 | 2 / 2 | 1 / 1 | 1 / 1 |
| gitea-webhook | 1 / 2 | 2 / 2 | 1 / 1 | 0 / 1 (native failure, ungraded) |

## A/B stop boundary

The first five attempts passed. The sixth, ATeam gitea-webhook repetition 1, exited 53 after 25.165s with a stderr-only `FatalTurnLimitedError`. The adapter retained optional fatal evidence and correctly reported the configured four-turn limit; stdout, normalized usage/model identity and a source/grade were unavailable. The diagnostic error session is separate from normalized identity. The runner stopped before all six repetition-2 attempts.

Stderr records a denied `read_file` attempt for the generated workspace’s `.env` sentinel. The diagnostic uses `/private/var/...` for the requested path and mixed `/private/var/...` and `/var/...` allowed-directory forms. This is an observation for offline investigation, not proof of the cause or a complete tool/turn trace. It was not the owner’s main `.env`; the denial does not establish every action attempted earlier. Four configured turns do not prove four billable requests. No 503 appears in this retained fatal diagnostic, but failed-attempt API counts remain unknown.

The completed prefix has three baseline passes and two candidate passes plus one candidate native failure. Three pairs are retained; only two are both-passed/evaluable for successful paired time, whose candidate-minus-baseline mean is -3.585s. Only two pairs have both usages known; their total-token delta is +6,991. Arm-wide execution means are 10.970s baseline (3/3 retained) and 12.241s candidate (3/3, including the failed call). These selected-prefix summaries are descriptive; neither the timing subset nor the missing-token arm total supports a winner claim.

The remaining six attempts were not run. Any further paid cohort needs a new predeclared protocol after offline investigation of tool choice, path handling and completion under the fixed turn limit. Do not continue this stopped cohort or retroactively raise its cap. Complete fatal-exit billing remains unknown under the explicit best-effort contract.

## Accounting and budget

Primary tokens are input plus output; thoughts are included within output and cached input is a subset, never added again. Native role counters duplicate the parent model totals and are not summed twice. The 19 v3 attempts with normalized usage reported zero cached tokens and zero API errors; the fatal attempt has unknown usage/API counts. Positive CLI aggregates are best effort and cannot prove every upstream request supplied metadata.

| Cohort | Known primary tokens | Illustrative known-priced subtotal | Missing billing |
| --- | ---: | ---: | --- |
| Earlier v1 + v2 | 35,413 | $0.05295675 | v2 fatal attempt |
| v3 canary | 45,104 | $0.05294700 | Provider reconciliation unavailable |
| v3 A/A | 136,954 | $0.13650150 | Provider reconciliation unavailable |
| v3 partial A/B | 58,167 | $0.05272125 | Fatal attempt |
| Combined | 275,638 | $0.29512650 | Two fatal attempts plus unverified provider reconciliation |

V3 alone retained 20 attempts, with primary usage reported for 19/20 and 240,225 known tokens; illustrative subtotal $0.24216975. The combined known subtotal is well below the owner’s $5 planning target, but actual spend is unknown, so this is not a verified under-$5 bill. No threshold was enlarged: 100k/400k/400k and 600/1800/1800 seconds remained frozen, with 180 seconds per attempt. Phase-boundary known-token reserves were 919,483 before A/A and 782,529 before A/B, both sufficient for the predeclared 400k threshold. Unknown charges and in-flight overshoot were not converted to zero.

Pricing basis: [Google Standard Gemini 3.8 Flash pricing](https://ai.google.dev/gemini-api/docs/pricing), checked October 20, 2026: $0.75/M uncached input, $0.075/M cached input and $3.75/M output including thoughts through December 31, 2026. Estimates apply to observed normalized fields only; actual service tier, billing and completeness are unverified.

## Immutable evidence

Raw traces, credentials, generated sources and session identifiers remain local/ignored. Published evidence uses sanitized outcomes and SHA-256 identities. Paths below are relative to the `codex/gemini-instruction-pilot` worktree.

### CANARY

- Original: `benchmarks/runs/gemini-instructions-v3-canary/experiment-3644050045/run.json`; SHA-256 `53194ecd01a72cc18ccef067062e046910904c3622236984fee7877506a20f8c`.
- Replay: `benchmarks/runs/gemini-instructions-v3-canary/experiment-3644050045/rescore-2214396764/run.json`; SHA-256 `226f776ebfbf3f1171b47b76c2b4b0a2796a0b3fe95f16aa0ffae9dff23143db`.
- Comparison and inspection: `.enjoy-logs/gemini-v3-canary-summary.json`; original bytes unchanged and comparison equal.

### AA

- Original: `benchmarks/runs/gemini-instructions-v3-aa/experiment-1083840159/run.json`; SHA-256 `9367e1bdcc876e96e1db94b5c53acef38bae5debb2a694ed96dc2f24cb9352cd`.
- Replay: `benchmarks/runs/gemini-instructions-v3-aa/experiment-1083840159/rescore-2376062002/run.json`; SHA-256 `7067ac3f2960ca7f446ee7bb23109e3b0de85a7da08b56b3e4485a15bbca6044`.
- Comparison and inspection: `.enjoy-logs/gemini-v3-aa-summary.json`; original bytes unchanged and comparison equal.

### AB

- Original: `benchmarks/runs/gemini-instructions-v3-ab/experiment-733613824/run.json`; SHA-256 `c1ace0d8bac09310f5ef70ab47ec08ec2eff91cf527e2841dfb1b5ef5998dfdd`.
- Replay: `benchmarks/runs/gemini-instructions-v3-ab/experiment-733613824/rescore-1456601303/run.json`; SHA-256 `18ded35057db61d65e1265cbc21c931dbe206133154291cbb507cfc78de6d0b3`.
- Comparison and inspection: `.enjoy-logs/gemini-v3-ab-summary.json`; original bytes unchanged and comparison equal.

Frozen protocol/configuration file hashes:

| File | SHA-256 |
| --- | --- |
| `README.md` | `0b5a0452327331439bec4ace1c2a9fbc673dc73ae3a318fc134a8c3fd9372088` |
| `aa.json` | `0aa0f91cc5d86403b006c9a76f4b1ea36b2cfe84d4b2985debca5953d15c6d91` |
| `ab.json` | `33d8a89d5233e749bcb208f468edb3be7c0baf9f9928f211107578e93ffb0590` |
| `ateam.md` | `449a4123f33a250132a1a2dac874d226d79c7c54c9ca32d53d46dd66494c4aba` |
| `baseline.json` | `0c64d5bb2fef49f2eea91b585069a3dd574872d5c0fb41d469fa539e50e1289b` |
| `baseline.md` | `b969561cdb8373611aa0f3da17d812da37bb8632f93a3b787c0989da65d4d0d7` |
| `canary.json` | `29c60fbc8a4fb6f7ccde29ce2f03ea4a09dece27e6340a40943895b2921f9021` |
| `candidate.json` | `102746167e0e6aef50a5982aa749d5c685458abeb85d9da2705684cd164c21af` |

The binary/configuration freeze and dry-run hashes are retained in `.enjoy-logs/gemini-v3-freeze.json` and the three `gemini-v3-*-dry-run.json` files. Advancement records are `gemini-v3-canary-gate.json` and `gemini-v3-aa-gate.json`. Final report review and delivery checks are recorded in PR #32. During delivery, main advanced to `e37563b` with CI and a new cross-agent merge-review gate. Both PRs were reconciled again; two inherited strict OpenSpec length failures were repaired by moving unchanged rule details into scenarios. The frozen live binary/protocol and results remain bound to the earlier recorded commits. Independent Codex review is complete, but the new gate additionally requires Claude or Gemini review for Codex-authored code. That cross-agent review remains outstanding; no review identity or merge protection was bypassed.

Skipped: the six remaining A/B attempts, retries, model/routing/native-skill comparisons and invoice reconciliation. Residual risk: small selected task sample, stochastic execution, incomplete A/B and unknown fatal-call billing prevent harness-superiority claims.
