# Mercury integration in the existing Go benchmark runner

The owner redirected PR #77 from standalone cancellation research to the smallest useful `internal/bench` integration. The Python guardian work was never committed and has been removed from the deliverable. The earlier cancellation plan is superseded by this scope; archived evidence remains unchanged.

## Scope and decision

Use an optional `claude.mercury` config with fixed `mercury-3`/low effort and a pinned LiteLLM executable. Keep the Claude runtime, existing single-file tasks, JSON schema output, scheduler, graders, token/time budgets, artifact replay and `--dry-run`. Go owns bridge startup/readiness and both process groups through the existing process helpers. Go also owns a fixed-route loopback HTTP proxy that captures provider usage before translation. LiteLLM remains an explicitly installed external dependency; no custom Python runs in this path.

Provider credentials remain only in the Go runner. Each attempt uses a fresh loopback bridge/token/configuration; Claude and LiteLLM receive only an ephemeral local token. LiteLLM config declares `allowed_openai_params: [max_retries]` and `max_retries: 0`; the proxy independently rejects nonzero SDK retry counts and latches failed exchanges before any later dispatch. The fixed provider endpoint is Inception. The bridge executable identity and version are retained and revalidated; direct Claude/Gemini/Codex behavior remains unchanged. A single finish path kills/waits for the bridge and closes/waits for the proxy before accepting accounting or removing the workspace. Provider HTTP requests carry the attempt context, so cancellation closes their sockets directly.

Go reconstructs source plus authoritative accounting from a retained envelope containing native stdout and the provider journal. Missing/partial/invalid usage, failed capture, mismatched model/controls, retries or execution errors fail the attempt and stop later calls. CLI estimates do not become provider billing. Raw usage includes cached/reasoning subsets without adding them twice. Capture buffers bounded responses; this path measures task outcomes and end-to-end runtime, not native streaming latency.

## Acceptance

- Go unit/helper-process tests cover resolution/dry-run, credential isolation, successful source grading/replay, missing/oversized journals, provider-accounting refusals and provider socket closure on cancel/timeout. Use existing mutation tooling for meaningful guards.
- Exercise the actual pinned Claude/LiteLLM `StructuredOutput` path against a local Go fixture with fake credentials and denied external networking. Generic multi-turn tool/permission matrices are deferred with a code shortcut; the schema tool used by these tasks is in scope.
- Run relevant direct-Claude regression checks, full clean-tree verification, doctor and independent technical review. Stop at review readiness; owner-arranged cross-service review precedes merge.
- Propose the bounded $1 canary/A/A in `benchmarks/mercury-canary-plan.md`; no paid call is authorized by implementation or offline qualification.

This replaces the sequence of separate research slices. Remaining unknowns should be reported against the actual benchmark path, not expanded into another runtime project.
