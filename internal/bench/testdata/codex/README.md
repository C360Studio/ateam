# Recorded Codex app-server fixture

`luna-app-server-success.jsonl` is a sanitized capture from the successful
2026-10-19 Luna API canary, using Codex CLI 0.160.1, `gpt-6-luna`, reasoning
`low`, and the benchmark's `benchmark_openai` provider. The private source is
`benchmarks/runs/luna-canary-final/experiment-958790448/attempt-0000.json`
(gitignored). It is a recorded native response, not a hand-authored protocol
example; it is not the original immutable run artifact.

The fixture retains the native order and fields of all 19 non-delta events,
including both thread warnings, paired user/reasoning/agent items, usage,
the account rate-limit diagnostic, and terminal completion. It omits all
284 `item/agentMessage/delta` events. Thread, session, turn, item and installation
IDs, host name, local paths and user-agent host details are replaced with fixture
values. Absolute timestamps become relative offsets; durations remain unchanged.
The repository task prompt is replaced with a placeholder, and the generated
source is replaced consistently with `package fixture\n`. The native reasoning
summary/content arrays were empty and remain empty. No credentials or private
raw trace are included.

Native token counts are unchanged: 7,374 input, 0 cached input, 7,371 cache-write
input, 738 output, 442 reasoning output, and 8,112 total, with identical `total`
and `last` snapshots. These describe the original request and response, not the
shortened fixture text. Cache-write and reasoning counters are subsets, not
additional normalized input/output tokens.

The model/provider/effort fields prove the CLI's resolved configuration only.
This capture contains no backend model attestation, billing identity or API
invoice. The account rate-limit snapshot is diagnostic metadata only. The
fixture tests protocol replay and accounting, not source quality or performance.
