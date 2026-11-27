# Tasks: archive-after-approval-412

## 1. Approval

- [x] 1.1 Owner approves the proposal (`Approved-by: Coby Leuschke (2026-11-27) sha256:648da44c7e60`).

## 2. Implementation

- [x] 2.1 Close step in `.agents/protocol.md` § 2 step 5 and `protocolCloseStep` (`internal/bootstrap/templates.go:720`): mark ready unarchived, append review-round tasks to the open change, archive in one document-only commit after the approving record, then arm auto-merge; Gitea archives before ready; a PR with no OpenSpec change arms at ready. Keep the three `squash` spots `merge_method` rewrites (`settings_test.go:179-188`) and the `openspec/changes/archive/` phrase doctor requires.
- [x] 2.2 `AGENTS.md` Close bullet and Close rule row, `.agents/rules-index.md` row, `closeRuleRow` (`templates.go:714`); keep the parity prefix `| \`review\` | A PR archives` (`parity_test.go:162`) or update it; `AGENTS.md` stays under 24,000 bytes.
- [x] 2.3 `docs/daily-use.md` § 5 step 4 and § 6 steps 1–2.
- [x] 2.4 Bootstrap test on the generated protocol and Close row for the new order and the Gitea exception, with a mutation check per sentence.
- [x] 2.5 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`.
- [x] 2.6 After the approving review record: archive this change in a document-only commit, then arm auto-merge.
