# Gemini instruction pilot v2: turn-limit stop, 2026-10-19

## Outcome

Under the larger shared output allowance, one attempt returned a complete source response that passed the independent safe-path grader. The second, identically instructed attempt failed with CLI exit 53 (`FatalTurnLimitedError`) and no usage counters. The runner retained both attempts and stopped. The canary acceptance gate failed: **one recorded pair, unevaluable; zero successful pairs**. A/A and A/B were not started. No automatic retry, fallback, response repair or additional paid call followed.

This is an operational finding, not an ATeam treatment result. Both canary arms used the same baseline instructions. The v1 response truncation and v2 turn-limit failure remain separate observations; a single successful response does not prove the larger output allowance fixed the earlier failure.

## Observations and accounting

| Measure | First / baseline | Second / candidate |
| --- | ---: | ---: |
| Status | passed | adapter-error |
| Independent quality grade | passed | unavailable |
| Agent elapsed seconds | 11.847 | 76.595 |
| Input tokens | 20,176 | unknown |
| Output tokens, including thinking | 614 | unknown |
| Cached / thinking subset tokens | 0 / 0 | unknown |
| Reported API requests / errors | 3 / 0 | unknown |
| Reported primary model | gemini-3.8-flash | unknown |

The first attempt made two successful read-only tool calls (`list_directory`, `glob`). The second stderr reports a failed `read_file` lookup and the native session-turn-limit error. It supplies no terminal usage envelope or source. The configured four-turn limit is not evidence of four billable requests. Raw stderr contains a different session ID from the first attempt, but the adapter's normalized session identity remains unavailable. Neither stream was truncated by the runner. Reported zero thinking tokens in the successful attempt does not establish backend reasoning configuration; low was requested and resolved reasoning remains unknown.

Known v2 primary usage is **20,790 tokens**, plus unknown usage from the failed attempt. Including v1, the known subtotal is **35,413 tokens**. Using [Google's Standard Gemini 3.8 Flash prices](https://ai.google.dev/gemini-api/docs/pricing), checked 2026-10-19, the v2 known-usage calculation is **$0.01743450** (20,176 input at $0.75/M + 614 output at $3.75/M). Combined with v1's $0.03552225, the known-priced subtotal is **$0.05295675**, plus unknown failed-attempt charges. These are illustrative counter-based estimates, not invoices or verified service-tier attribution. **Actual combined spend is unknown**; the owner-authorized $5 target cannot be certified from this evidence. Unknown accounting prohibits further live phases even though known usage is small.

The elapsed observations above are individual attempts, not a paired performance comparison. The failed attempt has no quality grade: it is not a correctness regression. No claim of stable capacity, equivalence, significance or harness superiority follows.

## Controls, verification and replay

The [predeclared v2 protocol](README.md) selected Gemini CLI 0.43.0, gemini-3.8-flash, low reasoning, 32,768 output tokens per call, four turns, 180 seconds per attempt, 600 seconds overall and a 100,000 known-token threshold for this canary. Treatment bytes are unchanged from v1. Native skills, hooks, routing and review remain disabled. Isolated API authentication and read-only tools were unchanged. No checks ran concurrently with native sampling.

Clean-tree `GOCACHE=/private/tmp/ateam-bench-audit-cache GOFLAGS=-p=1 task verify && task doctor` passed at `8aa9c6d91e5472e80c508933956435e44311795d`, followed by the runner build. This includes offline benchmark self-tests and UI checks. Doctor parity passed; actual MCP connection was unchecked and the observed hook process took 11,864 microseconds, so no sub-5ms process claim is made. `daaa4fea7e51b96099c9db3c3f43035664d4dc33` only removed an EOF blank line in the new specification; strict OpenSpec/diff checks and independent exact-head review passed before sampling. The live binary is the unchanged build from `8aa9c6d`. The live launch repeated and passed its automatic offline self-test.

Offline rescore reproduced the grade, failure and comparison exactly, with the original bytes unchanged and no API calls. The comparison retains one unevaluable pair and missing usage; it supplies no successful-pair time delta or token delta. Raw traces, source, request/settings, launchers and credentials remain local and ignored.

Paths are relative to this worktree:

- Original: `benchmarks/runs/gemini-instructions-v2-canary/experiment-3938985930/run.json`; SHA-256 `6de64525145dc32fbafa65fa01c705003e725e95403180bcbdb606e989401b0a`.
- Replay: `benchmarks/runs/gemini-instructions-v2-canary/experiment-3938985930/rescore-1417477481/run.json`; SHA-256 `551e1bfccad72964ab9d146a210c672c0769ac08aa3b4b83d52fd86e2a2feeab`.
- Canary configuration SHA-256: `259b3cc8290f8820b094221783bcc6a6d00b5ea9ab63b99c366c30fba5eca59f`.
- Runner SHA-256: `2126ac10803699b8c360d98f5ca50834ab53a86ff8fb76248ceca5a0b190c358`.
- Automatic self-test: `benchmarks/runs/selftest-20261019T211733.996767000Z-1083220266.json`.
- Logs/evidence: `.enjoy-logs/gemini-v2-preflight.log`, `gemini-v2-canary.log`, `gemini-v2-canary-evidence.json`, `gemini-v2-hashes.txt` and `gemini-v2-canary-dry.json`.

## Next boundary

Investigate the pinned CLI's turn-limited error/accounting behavior offline before another paid run. Review whether the source-only task needs the existing read-only discovery tools, or a clearer completion contract, rather than increasing limits reflexively. Any execution-policy change requires a new frozen cohort, independent review and successful canary; do not alter or pool these samples. Reconcile the failed attempt's billing externally if available; missing counters remain unknown. The instruction A/B question is still unanswered.
