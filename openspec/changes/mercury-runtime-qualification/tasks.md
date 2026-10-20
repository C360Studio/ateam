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

- [ ] Hold: owner authorization required before bridge installation, downloaded-code execution or adapter implementation; freeze full dependency/API/config contract first.
- [ ] Prove isolated actual-runtime request translation, controls, cancellation, permissions, accounting and replay using offline stubs and mutation checks.
- [ ] Hold: reconcile/land PR #43 and PR #32 before depending on their implementation; keep stopped Gemini evidence frozen.
- [ ] Complete independent implementation review and required checks before claiming a qualified runtime.
- [ ] Hold: separately reviewed frozen canary/A/A protocol and explicit new spending authorization required before any paid call.
