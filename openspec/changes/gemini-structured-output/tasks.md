# Tasks

## Implementation
- [x] Add explicit native schema mode while preserving legacy Gemini replay.
- [x] Correct native operational stop reasons and reject reused sessions before grading.
- [x] Add meaningful RED regressions, mutation proof and independent review.

## Delivery
- [x] Run full clean-tree verification and retry the bounded native canary.
- [x] Report outcomes and immutable replay, then publish through PR #13.

The schema canary stopped after one 120-second timeout with HTTP 503 evidence. Full Gemini A/A remains incomplete; no further run is scheduled. Structural review follow-ups 3 and 5 remain in the linked PR review and design notes.
