# Design

The executable plan and interpretation rules live in [the Mercury v1 protocol](../../../../benchmarks/experiments/mercury-instructions-v1/README.md). Keep one protocol rather than competing parameter lists.

Use the existing bundle manifests: baseline bytes exactly match Mercury calibration, while candidate adds the byte-identical six-point ATeam subset already versioned for Gemini. Native/runtime controls and graders are unchanged. Two repetitions of three tasks counterbalance arm order over six pairs. The run is bounded by attempt, time, provider-request and accounted-token controls; no new orchestration code is needed.

Tradeoffs: the calibrated fixtures may saturate correctness; instruction length and content are one treatment; two observations per arm/task do not support significance or equivalence. Provider caching, opaque backend behavior and buffered capture limit timing interpretation. Existing grading-infrastructure errors are retained but do not cause an immediate automatic stop. The protocol names these limits instead of changing the runner or tasks in a planning request.

Validation uses credential-free resolution, exact byte/hash and schedule/control comparisons to calibration, independent read-only review, doctor and full clean-tree verification. No production logic or test oracle changes are introduced; direct configuration validation is appropriate, with existing runner tests covering the reused seams.
