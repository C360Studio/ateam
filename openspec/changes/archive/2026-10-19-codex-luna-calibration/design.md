# Design

Use the installed Codex CLI (0.160.1), fixed gpt-6-luna and explicit shared low effort. The configured key source is OPENAI_API_KEY, scoped to the native child using the CLI's documented API authentication mechanism. Fresh HOME/CODEX_HOME and explicit configuration prevent subscription or ambient instruction/session inheritance. Retain host policies rather than bypassing them.

Prefer the existing native adapter helpers over a new abstraction. Request native JSON Schema output and parse the CLI event lifecycle, requiring a complete terminal response. Normalize primary input/output once; cached input and reasoning output are subsets. Missing usage remains unknown. Record CLI/binary/config/evaluator hashes, requested settings and actual native observations separately.

Before implementation approval, establish the installed CLI's model-identity telemetry and enforceable controls. If ordinary exec JSONL lacks model identity, do not copy the configured value into reported models or weaken other adapters' refusal rules. The selected evidence mechanism and its limits must be independently reviewed.

Verification covers malformed/duplicate/trailing events, incomplete terminals, missing/contradictory counters, changed/unverified models, API-only environment isolation, output bounds, cancellation and immutable offline replay. Critical refusals/accounting require non-destructive mutation evidence; finite/generated cases must state their oracle and coverage limits.

A live pilot follows reviewed code and clean-tree task verify, using the same three tasks × five repetitions × two arms. Use the established $2 target with token/time controls; it is not a hard dollar guarantee. Preserve provider-specific runs and all failed attempts separately.


## Reviewed protocol decision

Codex exec JSONL does not expose model/effort identity. Use the installed native app-server over stdio instead. Validate the actual thread/start response's model, modelProvider and reasoningEffort before sending turn/start; omit model/effort overrides on that request. Label this CLI-resolved configuration, leave provider-reported models unset and backend identity unknown, and reject model/rerouted. Both arms request identical settings; no provider switch or inference proxy is introduced.

Use a fixed custom provider named benchmark_openai with the official https://api.openai.com/v1 endpoint, env_key=OPENAI_API_KEY and requires_openai_auth=false. Built-in provider IDs cannot be overridden to set retry bounds. HTTP and SSE retries are both zero; no subscription/auth-file fallback. No supported model-turn or output-token cap was found, so only elapsed-time bounds and between-attempt reported-token thresholds are claimed.

Native cumulative total usage is counted once; last-request snapshots are retained but not added. Fields remain optional in parsed evidence so absent values remain unknown. Require exact thread/turn identity, one complete lifecycle and schema-valid final source; reject errors, reroutes, auxiliary agents and unexpected execution tools. Independent review approved this explicit assurance level and limits before implementation.
