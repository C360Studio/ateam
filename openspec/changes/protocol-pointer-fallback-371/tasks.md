## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removing the approval hold in the same edit (#371).
  - Evidence: the owner approved in Enjoy on 2026-11-11 and asked Claude to write the header.

## 2. Fallback

- [x] 2.1 Add `If your protocol has no such numbered sections, read all of it.` after the pointer sentence in the three contracts and the three skills, and in their template copies.
  - Premise: `.agents/contracts/{architect,developer,reviewer}.md:3`, `.agents/skills/{pickup,handoff,preflight}/SKILL.md:10`, `internal/bootstrap/templates.go:351,397,434,482`, `internal/bootstrap/review.go:19,30`.
  - Acceptance: `ateam doctor --check` parity and `go test ./internal/bootstrap` are green.
  - Evidence: the sentence now follows the pointer in 12 places: the 6 repository files, 4 lines in `templates.go` and 2 in `review.go`. `go test ./internal/bootstrap` passes, and `ateam doctor --check` reports `violations: null`.
- [x] 2.2 Have `readingViolations` report a document that contains `read §` without the fallback sentence.
  - Premise: `internal/bootstrap/reading_test.go:25`.
  - Acceptance: `task mutate:check` gives `detection` for dropping the sentence from one template.
  - Evidence: `readingViolations` now reports `section pointer without the whole-protocol fallback`. `task mutate:check` gives `detection` for removing the sentence from the developer contract template in `review.go`, and the source SHA-256 was unchanged.
- [x] 2.3 Update the protocol row in `docs/brownfield.md:75`.
  - Evidence: the `.agents/protocol.md` row now names the numbered headings, the mapping option and the whole-read fallback.

## 3. Close

- [ ] 3.1 Archive the change before the PR is marked ready, with `task verify` green and `openspec: strict ok`.
