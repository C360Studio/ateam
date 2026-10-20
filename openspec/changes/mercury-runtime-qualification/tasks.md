# Tasks

## Claim and static qualification

- [x] Reconcile pickup, current guidance, main, #16/#58/#59/#28 and open PR review evidence.
- [x] Isolate this documentation claim on `codex/mercury-offline-qualification`.
- [x] Inspect documented launcher and pinned bridge source; identify effective-control and evidence blockers.
- [x] Define scoped compatibility plan, source pins, offline checks and stop boundary.
- [ ] Complete independent read-only plan review and record repository verification in draft PR.
- [ ] Publish the scoped draft PR with `Addresses #58` and `Addresses #16`.

## Later qualification gates

- [ ] Hold: owner authorization required before bridge installation, downloaded-code execution or adapter implementation; freeze full dependency/API/config contract first.
- [ ] Prove isolated actual-runtime request translation, controls, cancellation, permissions, accounting and replay using offline stubs and mutation checks.
- [ ] Hold: reconcile/land PR #43 and PR #32 before depending on their implementation; keep stopped Gemini evidence frozen.
- [ ] Complete independent implementation review and required checks before claiming a qualified runtime.
- [ ] Hold: separately reviewed frozen canary/A/A protocol and explicit new spending authorization required before any paid call.
