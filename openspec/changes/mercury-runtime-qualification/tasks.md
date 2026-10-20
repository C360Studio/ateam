# Tasks

## Claim and static qualification

- [x] Reconcile pickup, current guidance, main, #16/#58/#59/#28 and open PR review evidence.
- [x] Isolate this documentation claim on `codex/mercury-offline-qualification`.
- [x] Inspect documented launcher and pinned bridge source; identify effective-control and evidence blockers.
- [x] Define scoped compatibility plan, source pins, offline checks and stop boundary.
- [x] Complete independent read-only plan review and record repository verification in draft PR.
- [x] Publish the scoped draft PR with `Addresses #58` and `Addresses #16`: [PR #61](https://github.com/C360Studio/ateam/pull/61).

Initial planning head `c50248803995e0e0e2dde659a9d5b76a61a6bb15` passed independent technical review, full clean-tree `task verify && task doctor`, separate strict OpenSpec validation and diff checks. Exact-head follow-up review/check evidence lives in PR #61. This is plan review only; it is not configured stronger-model or cross-provider approval. The initial sandbox loopback failure remains recorded separately from the successful approved rerun.

## Later qualification gates

PR #64 merged the offline blocker report. The next usage-provenance slice is claimed on `codex/mercury-usage-provenance`; see [its scope and checks](usage-provenance-plan.md). Issue #58 remains open.

- [x] Freeze the inspected offline stack, dependency hashes and synthetic API/config fixtures before isolated bridge installation and offline implementation.
  Owner authorized the scoped isolated offline setup and implementation by requesting review-ready delivery of PR #64. Production adapter delivery remains out of scope.
- [x] Exercise the actual pinned CLI/bridge against isolated fixtures and deliver the [offline blocker report](offline-report.md), retained evidence, replay checks and negative-control detections.
- [ ] Resolve provider-usage provenance and then finish tool histories, permissions, cancellation, error handling and broader replay qualification.
  Hold: the missing-usage fixture became successful numeric bridge/CLI counters; the route remains unqualified. See offline-report.md.
- [x] Reconcile landed PR #43 (`8e52e719`) and PR #32 (`b789cf62`) from main; keep stopped Gemini evidence frozen.
- [ ] Complete independent implementation review and required checks before claiming a qualified runtime.
- [ ] Freeze and review the canary/A/A protocol before live calibration.
  Hold: explicit new spending authorization required before any paid call.

## Provider-usage provenance continuation

- [ ] Audit supported pinned bridge controls and independently review the scoped evidence-contract plan.
- [ ] Implement and test the offline contract against retained and synthetic provider-wire evidence; address the sentinel cwd nit.
- [ ] Complete independent technical review, explicit Python evidence checks and clean-tree repository verification; publish review-ready PR.
