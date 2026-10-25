## 1. Implementation

- [ ] 1.1 `internal/testspec`: accept `//aip:verifies <IDs>` beside `spec:`, resolving each ID against `#### Scenario: <ID> ` headers in canonical and in-flight specs (not `archive/`), with the escape and UTF-8 checks; diagnostics for the spaced form, an empty ID list, and an unknown ID (#129).
- [ ] 1.2 Tests with an independent fixture oracle, and a `task mutate:check` mutant on the header match recorded here.
- [ ] 1.3 `.agents/protocol.md` § Test-to-spec attribution and the templated rules rows in `internal/bootstrap/templates.go` name both forms.
- [ ] 1.4 Spec delta: ADDED requirement `AIP scenario directive` in `specs/test-attribution/spec.md`, copied into the canonical spec before archive.
