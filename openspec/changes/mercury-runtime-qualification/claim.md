# Offline execution-contract claim

Issue #58 remains the tracker. PR #61 delivered the static audit and plan at main `411510438f8739c5a56a497301e51fba4465000e`; its merge did not finish runtime qualification. The next claim uses `codex/mercury-offline-contract` and the existing OpenSpec change rather than opening a duplicate issue or design.

## Deliverable and scope

The next slice follows the [reviewed offline sequence](design.md#offline-qualification-sequence-for-a-later-authorized-slice): produce a reproducible execution-contract report and offline fixtures for the actual pinned Claude Code/LiteLLM path, ending with a qualified contract or concrete blocker. Audit and freeze distributions/dependencies before isolated installation; use fake credentials and an external-network-denied mock provider for execution.

Prove both sides of parameter translation, instruction delivery, permissions, tool histories, cancellation, error/retry behavior and source-level usage completeness. Keep unknowns and all failed histories. Use meaningful independent test oracles, generated boundary/history checks where required, nondestructive mutation evidence and replay. Preserve the existing native adapters and frozen Gemini evidence.

Work belongs in this OpenSpec change and the smallest necessary offline test/fixture support. Choose implementation files after inspecting the existing test helpers. Do not add a production Mercury bench adapter until the execution contract is demonstrated and independently reviewed. Native skills remain #28; a second runtime and cross-runtime comparison remain #59.

## Acceptance and stop boundary

- Freeze the complete reviewed stack and effective controls; distinguish requested settings from captured behavior.
- Demonstrate isolated offline boundary, failure, accounting and replay checks, or report the exact unsupported boundary.
- Record independent review and the repository checks relevant to the eventual implementation.
- Stop after the offline contract report. Production adapter delivery and live calibration are subsequent gates.

The initial publication was claim-only. The owner subsequently requested driving this PR to review readiness, authorizing the scoped isolated offline setup and implementation. Installation/execution remains subject to distribution inspection and enforced isolation. Production adapter delivery and paid calls remain outside this slice; the spending hold remains in [tasks.md](tasks.md).

The discussed two-attempt canary followed by twelve A/A attempts, 100,000 reported tokens, 120 seconds per attempt, 30 minutes overall and $1 spending target are provisional planning values, not a frozen or authorized run protocol. Verify pricing, supported output/turn controls and overshoot limits before proposing live authorization. The earlier Gemini budget is not reused.

## Delivered investigation result

The [offline report](offline-report.md) records actual pinned CLI/bridge execution and a provider-usage provenance blocker. The scoped report and fixtures are delivered; the runtime is not qualified. Production adapter delivery and paid calibration remain deferred pending a separately reviewed resolution and the remaining qualification gates.
