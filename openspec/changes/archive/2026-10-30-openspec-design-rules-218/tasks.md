## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#218). Evidence: on 2026-10-30 the owner asked in Enjoy for "#218: write the design rules into the adopter openspec/config.yaml", deciding the issue's ship-or-not question; Claude wrote the header from that request.

## 2. Implementation

- [x] 2.1 Replace the `schema: spec-driven` literal (`internal/bootstrap/templates.go:593`) with `openspecConfig`, which carries the six `rules: design:` lines. Acceptance: a fresh `ateam init` writes them and the sync test passes; `task mutate:check` editing one rule reports `detection` (#218). Evidence: `TestInitWritesDesignRulesMatchingThisRepository` (`internal/bootstrap/adopter_rules_test.go`); `ateam mutate` with "120 lines" changed to "200 lines" reported `detection`. In a temp `git init` repo, `ateam init` then `openspec validate --all --strict --no-interactive` exited 0, and `openspec instructions design --change demo` printed the rules under `<rules>`.
- [x] 2.2 Update `docs/adoption.md:13` and `docs/brownfield.md:132-137` (#218). Evidence: both describe the written rules, and say an existing config is untouched; "(#218)" removed.
- [x] 2.3 `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green. Evidence: `task verify` exit 0 on a clean tree at `798eec5` (`test-spec: all discovered supported tests have valid citations`); `./bin/ateam doctor --check` `passed: true`, `violations: null`; `./bin/ateam bench --selftest` `selftest_passed: true`.
- [x] 2.4 Archive the change before marking the PR ready (#218). Evidence: archived to `openspec/changes/archive/2026-10-30-openspec-design-rules-218/`.
