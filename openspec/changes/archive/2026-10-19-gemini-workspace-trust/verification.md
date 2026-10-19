# Scoped trust verification

- Decision: finite child-environment and two-arm isolation examples suffice for one declarative environment assignment; no new parser or generated input grammar was added.
- Rule: only the generated workspace child receives explicit session trust; the parent environment, managed-policy refusal and configured tool restrictions remain intact.
- Inputs: two fresh arms, a nonempty parent trust sentinel, fake-child validation, managed settings overrides and fixed execution policy assertions.
- Run: `go test ./internal/bench -run '^TestGemini' -count=1` passed; independent read-only review and isolation/policy tests passed. Full clean-tree `GOCACHE="$PWD/.enjoy-logs/go-cache" task verify` passed at `d6470a4537d6d971a0a2701f41ce3e2272986032`. Actual retry got past folder trust and reached provider HTTP 503 responses; no calibration quality result.
- Mutation: direct Go overlay removal of the trust environment assignment produced `generated workspace lacks session-only trust`; baseline and after-run passed. Source SHA-256 stayed `bdad4e6a75262fc12560b0b35d2dfd30f94ea275efc6098a8dd129ec6c6ca734`. Evidence is retained locally in `.enjoy-logs/gemini-trust-mutation.log`; no repository source was temporarily mutated.
- Not covered: successful live A/A, provider billing reconciliation, OS sandboxing or a hook latency guarantee. The provider-capacity failure remains a blocker to calibration, not to this scoped startup fix.

See [the pilot report](../../../../benchmarks/calibration-pilot-2026-10-19.md) for artifact hashes, configuration and the preserved timeout/rescore evidence.
