# Tasks: decision-line-grammar-421

## 1. Approval

- [x] 1.1 Owner approves the proposal and design (`Approved-by: Coby Leuschke (2026-11-27) sha256:6b4d59375760`), written by claude on the owner's instruction in the #421 developer conversation; rulings D1 B, D2 A.

## 2. Implementation

- [x] 2.1 Grammar (premise `internal/state/forge.go:884-896`, `findDecisionLine`): strip leading `*`/`_`, keep `Decision:`/`Ruling:`, add `Owner decision`/`Owner ruling` with a `YYYY-MM-DD` date on the line. Accept: table test over literal lines (each form accepted and rejected, bold, mixed case, undated `Owner decision needed:`), plus mutation checks dropping the date condition and the emphasis strip.
- [x] 2.2 Callers (premise `internal/state/queuecheck.go:237`, `internal/state/state.go:3168-3181`): Q3 and "Recent decisions" read the widened field unchanged. Accept: one queue-check test where an owner-ruling comment yields Q3 and one where an undated line does not.
- [x] 2.3 `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
