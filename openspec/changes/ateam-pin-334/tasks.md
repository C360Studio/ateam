## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)`, ruling on design Open Questions 1–3, and removing this hold in the same edit (#334). Evidence: the owner approved in Enjoy on 2026-11-08 ("Approve, go ahead") with every Open Question as recommended: OQ1 `.agents/ateam-version`, OQ2 the linked-worktree README fix goes in task 3.1, OQ3 a violation. Claude wrote the header at their request.

## 2. Pin check

- [x] 2.1 Doctor reads `.agents/ateam-version` (D1) and reports `pin`. It adds a violation when `state.ReleaseVersion()` or `hook_binary.version` differs from the pin (D2, D3), when the file is malformed (D2), or when the hook binary is `unreadable` (D4). It adds none for an `absent` hook binary (D4), and skips the check in the ateam checkout (D5). Premises: `cmd/ateam/main.go:615`, `:622`; `internal/bootstrap/bootstrap.go:1133`, `:1147`; `internal/state/state.go:106`. Acceptance: a doctor test in a scratch adopter repository covers every case in the proposal's Definition of Done, and `task mutate:check` reports `detection` for the release comparison, the hook comparison, the format check, the `absent` exemption, and the ateam-checkout skip (#334). Evidence: `bootstrap.PinStatus` (`internal/bootstrap/pin.go`) is called after `HookBinaryStatus` in `runDoctor`, which adds `pin` to the report only when it is set. `TestPinStatus` covers no file, a match, running and hook mismatches, a `v0.3.9-8-gbd6941f` hook against `v0.3.9`, absent and unreadable hook binaries, a pre-release pin, seven malformed files, and the ateam checkout. `TestDoctorReportsDeclaredPin` runs `runDoctor` with no pin (no `pin` key) and with `v99.0.0` (one added violation, a non-zero exit). `ateam mutate` reports `detection` for each mutant: the checkout skip, the format check, the running comparison, the absent exemption, the unreadable case, the hook comparison, the empty-part and empty-pre-release checks, and in `runDoctor` the dropped violations and an unconditional `pin` key.

## 3. Docs and spec

- [x] 3.1 README § *Install a release* and `docs/brownfield.md` §1 say how to declare the pin and read it from a Taskfile, and drop "does not yet compare … (#334)". `openspec/specs/harness/spec.md` gains *Doctor checks a declared ateam pin*. Premise: `README.md:96`. Acceptance: `task spec:check` and `task test:spec` pass (#334). Evidence: the README § *Install a release* adds the linked-worktree note (install per worktree, or a committed wrapper that execs the main checkout's `.tools/bin/ateam`; design OQ2) and the pin paragraph. `docs/brownfield.md` §1 names the pin file. The Requirement merges into the base spec on archive.

## 4. Verify

- [ ] 4.1 `task verify` is green on a clean tree, `ateam doctor --check` and `ateam bench --selftest` exit 0, and the change is archived before ready (#334).
