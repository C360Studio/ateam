# Tasks: approve-recommended-450

## 1. Approval

- [x] 1.1 Owner approves the proposal.

## 2. Implementation

- [x] 2.1 Protocol § 2 step 1's Stop boundary bullet says each open question names a recommended option, the decisions section opens with `Approving accepts the recommended option of each decision unless you name another.`, and a different pick is written into the document and approved at its new digest (premise: the bullet says nothing about recommended options, `.agents/protocol.md` § 2 step 1, and its shipped copy, `internal/bootstrap/templates.go:294`). Accept: the sentence in both copies.
- [x] 2.2 Design skill § 2 marks the recommended option on each Decision and Open Question and opens the decisions section with the sentence (premise: "mark each one that needs an owner ruling", `.agents/skills/design/SKILL.md` § 2 and `templates.go:454`). Accept: the text in both copies.
- [x] 2.3 `openspec/config.yaml` `proposal:` and `design:` rules carry the same (premise: `openspec/config.yaml:9-18` and `openspecConfig`, `templates.go:341`, kept equal by `TestInitWritesDesignRulesMatchingThisRepository`). Accept: that test passes with both edited.
- [x] 2.4 Shipped-text test in `internal/bootstrap/adopter_rules_test.go` citing the new scenario; mutation of each template string shown as `detection`. Harness spec delta applied; `task verify`, `ateam doctor --check`, `ateam bench --selftest`.

## 3. Close

- [ ] 3.1 After the approving review record: archive this change in a document-only commit, then arm auto-merge when the coordinator says so.
