# Recorded Claude fixture

`claude-haiku-schema-success.json` was captured from Claude Code 2.1.292 using
`claude-haiku-5-5` with native JSON Schema output. It is attempt 0 of the local
2026-10-19 pilot `haiku-schema-pilot/experiment-3799230470`.

The fixture retains only parser-relevant result, source, accounting and outcome
fields. Its session ID is replaced with `fixture-session`; UUIDs, timing,
permissions and other envelope metadata are omitted. The source is the model's
answer to the repository's benchmark task. No credentials or private user
content are included, and the full raw trace remains local and ignored.

Native token counts are preserved: 4 uncached input, 950 cache creation,
0 cache read and 1,766 output, including 1,162 thinking tokens. The result ends
with `stop_reason: "tool_use"` and supplies `structured_output`. The CLI's
`total_cost_usd` is a client estimate; its model entry reports
`costBasis: "unknown"`, so this fixture does not establish authoritative billing.
