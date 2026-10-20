# Design

The adapter owns process exit status and bounded raw streams. On an actual exit53 with empty stdout, inspect stderr for complete native JSON error objects at the pinned CLI's line boundaries (bare `{` or `[ERROR] {`). Accept only the strict session_id/error schema, nonempty session/message, type FatalTurnLimitedError and code53. Identical duplicate emissions are allowed; malformed or conflicting candidate envelopes refuse normalization. No source, tokens or backend identity can come from stderr.

Store optional `gemini_failure` in native evidence with the reported session and error details. Keep the existing missing/incomplete stdout-derived accounting unchanged; the additional record is reported fatal evidence, not a usage ledger. New executions include a diagnostic naming the configured four-turn limit and missing usage. Other failures retain their existing diagnostics.

Replay re-derives additional failure evidence from retained stderr only when that optional field is present, checking the failed outcome and diagnostic. Historical artifacts without the field keep their original parsed response, bytes and comparison. Truncation, timeout/cancellation, nonempty stdout and other exit codes must not acquire turn-limit evidence.

Scope: gemini execution/evidence, run validation, focused tests, benchmark documentation and this OpenSpec change. Main risks are accepting noisy/conflicting text as authoritative and changing old artifact semantics. Finite boundary cases plus seeded/generated noise/refusal checks cover the framing/parser interaction; mutations must prove the classification and stop assertions activate. No extra dependency or telemetry channel.
