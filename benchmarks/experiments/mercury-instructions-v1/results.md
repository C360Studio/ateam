# Mercury instruction A/B v1 — worked example

## Question and finding

Does appending six Ponytail/ATeam engineering principles improve the existing three Go fixtures relative to minimal instructions, with the Mercury execution stack fixed?

Both arms passed all six attempts. The candidate used 22.7% more primary tokens and averaged 13.0% longer execution in this cohort. We observed no correctness improvement for that added time/token use. This is saturated correctness on a small task set, not proof of zero general ROI: maintainability, human review time, later rework and full-harness behavior were not measured.

This report follows the [frozen protocol](https://github.com/C360Studio/ateam/blob/0fb4a6066ff3f6adf76043261cd42961cd5c13a0/benchmarks/experiments/mercury-instructions-v1/README.md), declared before inference. The owner explicitly authorized one bounded A/B after reviewing the six-point subset. It ran on October 21, 2026. No retry, replacement, prompt tuning or sample expansion followed the observations. That one-run authorization is consumed.

## Test and controls

The [baseline](baseline.md) contains 78 bytes of minimal task guidance. The candidate adds [ateam.md](ateam.md), producing 921 bytes. These are a fixed instruction subset, not the full Ponytail plugin; native hooks/skills and repository workflow were not treatments. The baseline matches prior Mercury calibration exactly, and the appended subset is unchanged from the earlier Gemini experiment.

- Native Claude Code 2.1.292, LiteLLM 1.99.0, Inception `mercury-3`, low requested reasoning, structured single-source-file response.
- Safe-path, SQL-user and Gitea-webhook tasks; two repetitions per task/arm; seed 0; 12 attempts / six matched pairs with counterbalanced leading arms.
- Identical native settings, read-only tools, fresh workspaces/homes, task prompts and protected Go graders in both arms.
- Limits: 30 seconds/attempt, 720 seconds overall, 100,000 input + output tokens checked between attempts, 8192 requested output tokens/request, at most four provider requests/attempt. Proposed $1 target was not a hard billing cap.

The task prompts name exact interfaces and expected boundary behavior. SQL-user even names the parameterized query; webhook names HMAC-SHA256 and `hmac.Equal`. These fixtures offer limited scope/reuse decisions. They measure their protected checks, not arbitrary application correctness or security.

## Observations

| Metric | Baseline | Six-point subset |
| --- | ---: | ---: |
| Scheduled / attempted / graded | 6 / 6 / 6 | 6 / 6 / 6 |
| Passed / quality failed / unevaluable | 6 / 0 / 0 | 6 / 0 / 0 |
| Mean execution time | 6.641 s | 7.502 s |
| Execution range | 3.373–11.618 s | 3.428–16.128 s |
| Total grading time | 4.317 s | 3.546 s |
| Input tokens | 4,318 | 5,233 |
| Output tokens | 9,577 | 11,819 |
| Total primary tokens | 13,895 | 17,052 |
| Cached input subset | 2,062 | 2,571 |
| Reasoning output subset | 9,307 | 11,655 |

Execution includes native/bridge overhead and excludes grading. This is not raw provider streaming throughput. Subsets are already included in primary totals. There were no unstarted attempts, unpaired attempts, execution failures, grading infrastructure errors, unknown-usage attempts or retries. All 12 provider requests completed with HTTP 200 and raw usage. Recorded request controls matched the frozen model, low effort and 8192 output allowance; provider responses identified `mercury-3`. Requested effort is not independent proof of backend reasoning configuration.

All six pairs passed in both arms, so all six contribute to paired time and token comparisons. Candidate-minus-baseline mean execution difference was +0.861 s; total token difference was +3,157. Percentages compare the per-arm means and totals, respectively.

| Task / repetition | Baseline time | Candidate time | Time delta | Baseline tokens | Candidate tokens | Token delta |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| safe-path / 1 | 11.554 s | 16.128 s | +4.574 s | 4,505 | 7,322 | +2,817 |
| sql-user / 1 | 5.875 s | 3.428 s | −2.447 s | 1,056 | 1,220 | +164 |
| gitea-webhook / 1 | 3.373 s | 5.660 s | +2.287 s | 1,318 | 1,882 | +564 |
| safe-path / 2 | 11.618 s | 12.354 s | +0.736 s | 4,610 | 4,014 | −596 |
| sql-user / 2 | 3.450 s | 3.623 s | +0.173 s | 1,004 | 1,209 | +205 |
| gitea-webhook / 2 | 3.975 s | 3.819 s | −0.156 s | 1,402 | 1,405 | +3 |

The A/B consumed 30,947 primary tokens. Earlier canary and A/A used 8,248 and 13,962 respectively, making the separate phase subtotal 53,157. Calibration samples are not pooled into the A/B treatment comparison. Actual monetary cost remains unverified; tokens are not an invoice.

## Evidence and reproduction

[results.json](results.json) holds the curated machine-readable comparison, every attempt and every pair. [evidence.json](evidence.json) links it to original run and offline-rescore hashes and states where evidence is available. This worked-example schema is illustrative; the report exporter has not yet been implemented.

The credential-free prelaunch dry-run exactly matched the frozen configuration and bundle/executable/grader hashes. Automatic offline self-test passed before inference. The run exited 0. Offline `compare` validated the retained history; `rescore` regraded all 12 sources with matching outcomes, response/usage/source bytes and unchanged original file hash. All 20 sessions across canary/A/A/A/B were distinct. No model call was needed for replay.

For a reviewer with the private artifacts and matching pinned runner/grader:

```sh
.ateam/mercury-canary-preparation/ateam bench compare benchmarks/runs/mercury-instructions-v1-ab/experiment-1271886322/run.json --json
.ateam/mercury-canary-preparation/ateam bench rescore benchmarks/runs/mercury-instructions-v1-ab/experiment-1271886322/run.json
```

Raw native/provider traces, source outputs and local executable snapshots remain ignored/private. These three summary files are not a substitute replay artifact. Hashes detect byte changes; they do not authenticate evidence or make the private run available to a repository-only reader. No absolute local paths, native session IDs, keys or account screenshots are included.

## Interpretation and next decision

This cohort found an observed time/token overhead without a measurable correctness gain. It cannot establish statistical significance, equivalence, a general winner, or overall productivity ROI. Instruction length/content, provider caching, stochastic reasoning and a six-pair sample limit attribution. The single largest token increase was safe-path repetition 1; the second safe-path pair used fewer candidate tokens, illustrating variability.

Preserve this result as a worked example. Next, automate trustworthy report closeout from existing artifacts and separately design a balanced task set with meaningful reuse, scope and change-propagation decisions. Do not choose only tasks likely to favor the treatment or reinterpret these fixtures after seeing the result. Further live runs need a new predeclared protocol and explicit authorization.
