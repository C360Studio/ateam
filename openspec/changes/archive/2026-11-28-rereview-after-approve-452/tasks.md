# Tasks: rereview-after-approve-452

## 1. Approval

- [x] 1.1 Owner approves the proposal.

## 2. Implementation

- [x] 2.1 `rereview` sends the re-review after an `APPROVE` record at an older sha (premise: `case rec.Verdict != "REQUEST CHANGES": return "record verdict is " ...`, `cmd/ateam/reviewlaunch.go` `rereview`). Accept: an `APPROVE` record behaves as `REQUEST CHANGES` does; a missing or other verdict still returns `record verdict is <verdict>`.
- [x] 2.2 `rereviewText` names the verdict and, after `APPROVE`, drops the confirm-findings clause (premise: the text hard-codes `after your REQUEST CHANGES at %s; ... confirm each finding of your record is closed`, `cmd/ateam/reviewlaunch.go` `rereviewText`). Accept: `REQUEST CHANGES` text byte-identical to today's.
- [x] 2.3 `TestReviewRequestRereview` (`cmd/ateam/reviewlaunch_test.go`): `approve` case sends the APPROVE text, a `no-verdict` case falls back; mutation restoring the refusal shown as `detection`. `docs/daily-use.md` step 11 names the catch-up case. `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [x] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
