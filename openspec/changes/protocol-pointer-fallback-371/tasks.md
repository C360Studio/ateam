## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing the approval hold in the same edit (#371).
  - Hold: OPEN-Q proposal awaits owner approval

## 2. Fallback

- [ ] 2.1 Add `If your protocol has no such numbered sections, read all of it.` after the pointer sentence in the three contracts and the three skills, and in their template copies.
  - Premise: `.agents/contracts/{architect,developer,reviewer}.md:3`, `.agents/skills/{pickup,handoff,preflight}/SKILL.md:10`, `internal/bootstrap/templates.go:351,397,434,482`, `internal/bootstrap/review.go:19,30`.
  - Acceptance: `ateam doctor --check` parity and `go test ./internal/bootstrap` are green.
- [ ] 2.2 Have `readingViolations` report a document that contains `read §` without the fallback sentence.
  - Premise: `internal/bootstrap/reading_test.go:25`.
  - Acceptance: `task mutate:check` gives `detection` for dropping the sentence from one template.
- [ ] 2.3 Update the protocol row in `docs/brownfield.md:75`.

## 3. Close

- [ ] 3.1 Archive the change before the PR is marked ready, with `task verify` green and `openspec: strict ok`.
