## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Questions 1-3 (#237). Evidence: the owner approved in Enjoy on 2026-10-30 with the recommended rulings (OQ1 D4 A, OQ2 D3 B, OQ3 D1 A); Claude wrote the header at their request.

## 2. Diff scope

- [ ] 2.1 Record each declaration's span (first attached comment line through the closing line): Go from `fn.End()` in `scanGo` (`internal/testspec/check.go:194-235`), Vitest from `node.getEnd()` beside the record push (`internal/testspec/vitest.mjs:96`). Acceptance: existing `TestGoAttribution`, `TestVitestIntegration`, and `TestDiscoveryAndDeterminism` pass unchanged (#237).
- [ ] 2.2 Add the D1/D3 changed-line set from `git merge-base <ref> HEAD`, `git diff -U0 --no-ext-diff <merge-base>`, and `git ls-files --others --exclude-standard`, and filter only `missing spec citation` (`check.go:178`) to declarations whose span meets a changed line (D4). `Check` takes the base (D5; callers `cmd/ateam/main.go:672` and `check_test.go`). Acceptance: a new fixture-repo test covers an added cited test (pass), an added uncited test (fail), an edited uncited legacy body (fail), an untouched uncited legacy test (pass), an untracked uncited test file (fail), and a dangling citation on an untouched legacy test (fail). `task mutate:check` reports `detection` with the span-intersection test inverted and with untracked files dropped (#237).
- [ ] 2.3 `runTestSpec` (`cmd/ateam/main.go:667-683`) parses `--base <ref>` and rejects other arguments with exit 2; unknown ref, missing merge base, and non-git root exit 2 naming `git fetch`/`fetch-depth: 0` (D6). Acceptance: CLI tests for each exit code, and `ateam test-spec` without arguments prints today's output on this repository (#237).

## 3. Docs and spec

- [ ] 3.1 `ateam --help` line (`cmd/ateam/main.go:94`), protocol § Test-to-spec attribution (`.agents/protocol.md:67-78`) and its bootstrap template, `docs/adoption.md:13`, and a new `docs/brownfield.md` section (after § 3, `:41`) give the D7 adopter path with a CI snippet. Acceptance: `ateam doctor --check` parity and `task spec:check` pass (#237).

## 4. Verify

- [ ] 4.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#237).
