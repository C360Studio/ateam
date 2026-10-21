# Let the Mercury bridge own its ephemeral port

Issue #104 blocks the otherwise approved archival PR #103. The runner currently reserves and releases a loopback port before the child binds it; readiness probes can reach another process that acquired the port meanwhile.

Pass `--port 0` to the already pinned LiteLLM bridge. Read its actual loopback address from the child's bounded startup output, then retain the existing authenticated model-readiness check. This removes the reservation window rather than retrying guessed free ports. No provider call, runtime replacement, dependency, generic process framework or extended stress investigation is needed.

Scope: `internal/bench/mercury.go`, a small startup-output helper, existing fake bridge fixture and focused tests; canonical benchmark specification plus this change. Preserve process cleanup, context deadlines, usage evidence and model-run authorization boundaries. Done when deterministic regression tests, mutation proof, pinned offline startup, required verification and independent review pass. Close #104 on merge, then resume #103 and the requested handoff.
