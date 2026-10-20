# Gemini instruction pilot: canary stopped, 2026-10-19

## Outcome

Gemini returned a native response with **one reported API request and zero reported API errors**. No HTTP 503 was observed in this launch. The response was incomplete JSON, so the existing strict parser stopped the run with `adapter-error` / `gemini final message: unexpected EOF`. This is availability evidence for one request, not stable capacity or a completed A/A calibration.

Only the first baseline attempt of the two-attempt safe-path canary ran. The candidate attempt, full A/A and A/B were not started. There are **zero evaluable pairs and no task-quality result**. The predeclared advancement gate failed; no automatic live retry, model fallback or output repair followed.

## Observed counters and likely boundary

| Measure | Observation |
| --- | ---: |
| Agent elapsed time | 64.829 seconds |
| Input tokens | 6,438 |
| Candidate-output tokens | 323 |
| Thought tokens | 7,862 |
| Normalized output, including thoughts | 8,185 |
| Total primary tokens | 14,623 |
| Cached input | 0 |
| Native tool calls | 0 |
| Reported API requests / errors | 1 / 0 |

The adapter requested 8,192 maximum output tokens and low reasoning. Reported output was 8,185 tokens, including 7,862 thoughts, and the JSON stopped mid-source. **Output-budget exhaustion is a strong inference**, not a verified finish reason: the retained CLI envelope exposes neither the backend finish reason nor resolved thinking configuration. Low reasoning was requested; it was not independently observed. A native `completed` envelope did not establish a complete final source response.

The raw native envelope was 2,307 bytes and its response string 988 bytes, far below the runner's 1 MiB stream ceiling; no runner-output truncation was recorded. The parser preserved the invalid response rather than stripping fences, repairing JSON or grading partial source. The diagnostic also reported ripgrep fallback; no tool was called in this attempt.

Using [Google's Standard Gemini 3.8 Flash rates](https://ai.google.dev/gemini-api/docs/pricing), checked 2026-10-19, gives an illustrative **$0.03552225**: 6,438 input tokens at $0.75/M plus 8,185 output tokens at $3.75/M. Thinking is already included in output and is not charged twice in this calculation. This is a rate calculation from CLI counters, not an invoice, verified service-tier attribution or complete historical Gemini bill. The $2 target was not a hard cap.

## Verification and replay

The frozen protocol and execution code were at `5a2b99962e72f60b256a7620e9d2c94ec35cd37a`, based on main `ce3a351`. Independent read-only protocol review found no blockers after stop rules were aligned with the unchanged runner. All three configurations resolved in dry runs. Full clean-tree `GOCACHE=/private/tmp/ateam-bench-audit-cache GOFLAGS=-p=1 task verify && task doctor` passed before sampling, followed by the runner build. No verification workload ran concurrently with inference. Doctor reported parity passed, no actual MCP connection check and one 8,484-us process-hook observation; no sub-5ms process claim is made.

An initial local launcher invocation failed the offline evaluator gate (2/6 tasks) before any agent attempt or API call. The wrapper had omitted ordinary HOME/TMPDIR environment, and mutation fixtures reported that their test was not selected. Restoring the ordinary host environment to the parent runner made the credential-free self-test pass; managed-setting override variables were also retained so they could trigger refusal. The native adapter still constructs its own isolated child environment. This operator-wrapper correction changed no tracked runner code. The subsequent paid launch repeated and passed its automatic offline evaluator gate before inference. The failed offline report remains retained.

Offline rescore preserved the adapter error, original bytes and comparison exactly. It performed no model calls and did not invent a grade for incomplete source. Raw traces, exact request/configuration, self-test reports and logs remain private and ignored.

Paths below are relative to the `codex/gemini-instruction-pilot` worktree:

- Original: `benchmarks/runs/gemini-instructions-v1-canary/experiment-1089168808/run.json`; SHA-256 `71ec37f2b5a07e2f90a8dbfac8880b7cf6c682bfec26e1e4517a17d2d46746e5`.
- Verified replay: `benchmarks/runs/gemini-instructions-v1-canary/experiment-1089168808/rescore-3204560437/run.json`; SHA-256 `4196b647529c5d16ccb6f6b84095fab5fa2f9fddbf879bb3a1846a7c3175e794`.
- Canary configuration SHA-256: `e564cd7b3aeb3171a80d5f16339f9bd7df9fad96039af06137fc90c8179c4e19`.
- Runner SHA-256: `625f30b8296556a2abdbaff0b2f29e4ea1f6d99732f1e2bd1403c4075ba73029`.
- Passing automatic self-test: `benchmarks/runs/selftest-20261019T205716.995378000Z-2293432930.json`.
- Failed offline launch gate: `benchmarks/runs/selftest-20261019T205550.221992000Z-3427702138.json`.
- Preflight, corrected-wrapper self-test, native launch and replay summary: `.enjoy-logs/gemini-pilot-preflight-2.log`, `gemini-launcher-selftest.log`, `gemini-instructions-canary-2.log`, `gemini-canary-evidence.json`.

## Next step

Inspect the pinned CLI's output/thinking-budget contract, then propose an explicit shared output budget that leaves room for reasoning plus a complete schema response. Keep the model and reasoning the same across arms. A revised limit needs a fresh frozen protocol, independent review, budget projection and successful canary before A/A or A/B. Do not call this a task-quality regression or claim ATeam helps/hurts from this failed canary attempt. Historical 503 failures remain separate observations; this one response does not prove the paid tier caused recovery.
