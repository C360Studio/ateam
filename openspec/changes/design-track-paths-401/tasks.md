# Tasks: design-track-paths-401

## 1. Approval

- [ ] 1.1 Owner approves proposal.md and design.md together and rules on Q1 and Q2. Hold: OPEN-Q proposal awaits owner approval

## 2. Implementation

- [ ] 2.1 Parse and validate `design_paths` where D1 rules (premise: `internal/review/policy.go:12-24` or `internal/state/settings.go:31-38`). Check: an invalid glob fails at load, naming the entry.
- [ ] 2.2 Add `internal/state/designpaths.go`: diff paths against the base (reuse the `internal/testspec/diff.go:40-61` resolution), match per D3, detect a design per D4. Check: the DoD fixtures in the proposal.
- [ ] 2.3 Wire `--base` into `state spec-check` (premise: `cmd/ateam/main.go:141,209`), and wire CI if Q2 says so (`.github/workflows/ci.yml:32,59`, `.gitea/workflows/ci.yml`).
- [ ] 2.4 Mutation checks: drop the design.md detection, and match nothing. Both should detect.

## 3. Docs and close

- [ ] 3.1 Docs per D1, `.agents/protocol.md:31` and its template copy in `internal/bootstrap/templates.go`.
- [ ] 3.2 Run `task verify`, `ateam doctor --check` and `ateam bench --selftest`, then archive this change.
