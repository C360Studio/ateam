## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#264). Evidence: on 2026-10-30 the owner picked "Approve both, quick track" in Enjoy; Claude wrote the header from that answer.

## 2. Implementation

- [x] 2.1 Print `openspec: strict ok (<path>, <version>, <n> changes)` after a passing strict run (`internal/state/openspec.go`, `CheckOpenSpec`). Acceptance: `TestCheckOpenSpecFindsValidatorOnPATH` covers PATH with a version, local with `version unknown`, skip, and failure; dropping the line reports `detection` (#264). Evidence: `ateam mutate -pkg ./internal/state -test TestCheckOpenSpecFindsValidatorOnPATH -file internal/state/openspec.go -expect-text "success line missing"` reported `detection`, SHA-256 of `openspec.go` unchanged; in this repository `ateam state spec-check` printed `openspec: strict ok (.../node_modules/.bin/openspec, 1.14.3, 0 changes)`.
- [x] 2.2 Name the success line in `docs/brownfield.md` § OpenSpec tracks (#264).
- [x] 2.3 `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green. Evidence: `task verify` exit 0 on a clean tree at `498bf08` (`test-spec: all discovered supported tests have valid citations`, `selftest_passed: true`); `./bin/ateam doctor --check` `passed: true`, `violations: null`; `./bin/ateam bench --selftest` `selftest_passed: true`.
- [x] 2.4 Archive the change before marking the PR ready (#264). Evidence: archived to `openspec/changes/archive/2026-10-30-spec-check-success-264/`.
