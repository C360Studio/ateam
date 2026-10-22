# Decision-opportunity tier v1: Mercury calibration and instruction A/B

Frozen on 2026-10-22 before any inference beyond the canary. Owner authorization: paid test runs under a combined $5 target (conversation record, 2026-10-22). The target is not a hard billing cap; the runner stops between attempts on the token threshold and on any missing usage.

## Question

On tasks that offer a reuse, overbuild or change-propagation decision (`reuse-email`, `overbuild-slug`, `propagate-limit`, issue #111 / PR #114), does appending the six-point ATeam subset change correctness, discipline-gate outcomes, source size, time or tokens relative to minimal instructions, with the Mercury execution stack fixed? The same six-point subset on the three prescribed tasks cost +13.0% time and +22.7% tokens with correctness saturated (`mercury-instructions-v1`, PR #85).

## Fixed stack

- Claude Code 2.1.292 (executable SHA-256 prefix `97a01e5bc74a199e`), LiteLLM 1.99.0 from the pinned qualification venv, Inception `mercury-3`, requested reasoning `low`, structured single-source-file output, 8192 requested output tokens per request, at most four provider requests per attempt, no retries or fallback.
- Branch binary built from PR #114 head `70334aa` (after the independent review fixed the `propagate-limit` gate): runner SHA-256 prefix `171994bb2d725400`; evaluator digest prefix `0311ba302fe09f48` (canary, one task) and `2a93e1b6a9a3e02d` (A/A and A/B, three tasks). An earlier canary at head `c8e0398` (run `experiment-3784352982`, 2/2 passed, 1,904 tokens) was superseded by the grader fix and is not part of this cohort.
- Config digests from credential-free dry-runs: canary `799a3735217038fc`, A/A `63420e23fec40124`, A/B `d4d6471a3edd67fa`.
- Both arms share native settings, read-only tools, fresh workspaces and homes, task prompts (including the shown existing package source) and the protected Go graders.

## Stages and gates

| Stage | Config | Arms | Attempts | Token stop | Wall limit | Advance when |
| --- | --- | --- | ---: | ---: | ---: | --- |
| Canary | `canary.json` | identical, `reuse-email` only | 2 | 50,000 | 180 s | both attempts retain complete usage, expected model/controls, replayable artifacts, no runner failure |
| A/A | `aa.json` | identical bundle (`baseline.md`) | 6 | 50,000 | 360 s | complete usage on every attempt; discrepancies between identical arms understood before A/B |
| A/B | `ab.json` | `baseline.md` vs `baseline.md` + `ateam.md` | 12 | 100,000 | 720 s | reported as observed; no winner, significance or equivalence claim |

The treatment files are byte-identical to `mercury-instructions-v1`: `baseline.md` is 78 bytes, `ateam.md` adds the six principles. A/A uses the baseline bundle in both arms. Seed 0, counterbalanced leading arm, two repetitions per task in the A/B.

Each attempt records behavior and discipline outcomes separately; a tier attempt passes only when both pass. `compare` reports source bytes over both-passed pairs as a descriptive size signal.

## Results (2026-10-22, all three stages)

| Stage | Run | Attempts | Passed | Primary tokens | Replay |
| --- | --- | ---: | --- | ---: | --- |
| Canary | `experiment-1818981367` | 2/2 | 2/2 | 1,977 | matched ([canary/](canary/)) |
| A/A | `experiment-1446345992` | 6/6 | 3/3 and 3/3 | 7,131 | matched ([aa/](aa/)) |
| A/B | `experiment-1450088960` | 12/12 | baseline 6/6, candidate 4/6 | 15,811 | matched ([results.md](results.md)) |

Every attempt retained complete provider usage; no retries, replacements, timeouts or unknown charges. Combined 24,919 primary tokens, about one cent at the list prices checked on 2026-10-20 ($0.20/M input, $0.70/M output); the invoice was not reconciled. Interpretation and the next decision are in [interpretation.md](interpretation.md).

## Not claimed

Three tasks and six pairs cannot establish significance, equivalence or general ROI. The discipline gate reads candidate syntax only (see PR #114 residual risk). Mercury results do not transfer to other models. Raw native traces, sources and provider responses stay private under `benchmarks/runs/`.

## Grader patch after the cohort

These exports were recorded with the grader at `70334aa` (evaluator `2a93e1b6…`) and are left unchanged. Codex review at `57ab500` then found two grading defects, both fixed in `ef2f887`. First, compiler diagnostics reported only in the protected `existing.go` (for example a missing or string-typed `MaxLen`) were recorded as grading errors and dropped from quality comparison; they are now failed attempts. Second, the `propagate-limit` test accepted `var LengthError error`, because `errors.Is(nil, nil)` is true; it now requires a non-nil sentinel and a non-nil error for invalid lengths. Re-evaluating the 20 retained cohort sources with the fixed grader (private, not committed): `re-evaluated 20 sources; 0 outcomes changed`. Every behavior, discipline and pass outcome matches the recorded grade. The committed exports still name the earlier evaluator digest, so the fixed binary reports `evaluator mismatch` for them.
