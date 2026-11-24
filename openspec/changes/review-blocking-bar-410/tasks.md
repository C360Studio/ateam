# Tasks: review-blocking-bar-410

## 1. Approval

- [ ] 1.1 Owner approves the proposal and design. Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Contract (premise `.agents/contracts/reviewer.md:7,11`, `internal/bootstrap/review.go:34,38`): bar, backstop credit, `Non-blocking:` list, dissent, later-round scope in both copies. Accept: bootstrap test asserts the bar sentence in the generated and repository contract.
- [ ] 2.2 Skill (premise `.agents/skills/review-pr/SKILL.md:35`, `internal/bootstrap/templates.go:535`): re-review paragraph confirms prior findings, reads `<record>..<head>`, widens only on new files or request. Accept: `parity_test.go:113` keeps both bodies equal.
- [ ] 2.3 Message (premise `cmd/ateam/reviewlaunch.go:388-391`, test `cmd/ateam/reviewlaunch_test.go:453`): new wording without "full base-to-head diff". Accept: literal expected string in the test.
- [ ] 2.4 `task mutate:check` on 2.1 and 2.3; `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
