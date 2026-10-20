# Mercury benchmark integration

Addresses #58/#16 in PR #77; follows [archived offline evidence](../archive/2026-10-20-mercury-runtime-qualification/README.md). The owner superseded the standalone cancellation research with the [minimal Go integration plan](integration-plan.md).

## Current delivery

- [x] Reconcile the user steer, existing Go execution/dry-run/replay seams and independent plan review.
- [x] Remove uncommitted Python guardian work from the deliverable; preserve earlier archived evidence.
- [x] Integrate the pinned Mercury bridge with the existing Claude benchmark path, Go cleanup and authoritative usage replay.
- [ ] Complete focused Go/mutation tests, actual offline schema-path check, independent review and full repository checks; deliver PR #77 ready for cross-service review.
- [ ] Review and explicitly authorize the [bounded canary/A/A proposal](../../../benchmarks/mercury-canary-plan.md) before paid execution.
  Hold: no paid calls are authorized yet. Missing usage or failed attempts stop the run; a spending target is not a hard monetary cap.

General tool/permission matrices, new runtimes and cross-runtime comparisons are deferred beyond the current single-file tasks. Native runtimes and frozen Gemini evidence remain unchanged.
