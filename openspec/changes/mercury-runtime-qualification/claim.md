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

This claim publication contains tracking and task-formatting changes only. It does not install or execute downloaded code, implement the adapter, or authorize paid calls. The existing installation/implementation and spending holds remain explicit in [tasks.md](tasks.md); the owner can authorize the next execution slice separately.

The discussed two-attempt canary followed by twelve A/A attempts, 100,000 reported tokens, 120 seconds per attempt, 30 minutes overall and $1 spending target are provisional planning values, not a frozen or authorized run protocol. Verify pricing, supported output/turn controls and overshoot limits before proposing live authorization. The earlier Gemini budget is not reused.
