## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#341). Evidence: the owner approved in Enjoy on 2026-11-08 ("Approved — write Approved-by for me"); Claude wrote the header at their request.

## 2. Key and read order

- [x] 2.1 `Settings.AteamVersion` (`json:"ateam_version,omitempty"`) in `internal/state/settings.go`. `isReleaseTag` moves from `internal/bootstrap/pin.go` into state and is exported for bootstrap.
  - Acceptance: `TestLoadSettings` gains a valid case and malformed cases (`0.3.9`, `v0.3`, `v01.2.3`, `v0.4.0-rc.01`, `v0.3.9+b`).
  - Mutation that removes the key's validation is detected.
  - Evidence: `IsReleaseTag` now lives in `internal/state/release.go`. `Settings.AteamVersion` is validated in `LoadSettings`. `TestLoadSettings` gains the `ateam_version` case and five `bad_ateam_version` cases. `ateam mutate` reports `detection` when the validation is dropped.

- [x] 2.2 `PinStatus` (`pin.go:19`) reads the key, then the file. When both exist and disagree, it warns that the key wins.
  - Acceptance: a table test covers neither, key only, file only, both agree, and both disagree. Each case's expected pin and finding is written out by hand.
  - The existing `pin_test.go` cases pass unchanged.
  - Mutations that swap the read order or drop the disagreement warning are detected.
  - `cmd/ateam/pin_test.go` gains a doctor case for the key.
  - Evidence: `PinStatus` returns `(pin, violations, warnings)`, reading the key first and then the file. `runDoctor` appends the warnings. `TestPinStatusKeyAndFile` covers eight cases (neither, settings without the key, key at the pin, key with running differing, file only, both agree, both disagree, and an invalid settings file falling back to the file). `TestPinStatus` passes unchanged, apart from also asserting no warnings. `TestDoctorReportsDeclaredPin` adds the key-wins violation and the disagreement warning. `ateam mutate` reports `detection` when the read order is ignored and when the warning is dropped.

## 3. Docs and rule

- [x] 3.1 `README.md:98` shows `jq -r .ateam_version .agents/ateam.json` and marks the file as deprecated. `docs/adoption.md` § Team settings lists the key. The AGENTS.md §3 row and the `.agents/rules-index.md` row list `ateam_version`.
  - Acceptance: `ateam doctor --check` parity passes, and AGENTS.md stays under 24,000 bytes.
  - Evidence: README § Install a release shows `jq -r .ateam_version .agents/ateam.json` and marks the file deprecated. `docs/adoption.md` § Team settings gains a key row and adds the key to its example. The AGENTS.md and rules-index rows list `ateam_version`. AGENTS.md is 14,852 bytes.

## 4. Verify

- [ ] 4.1 `task verify` is green on a clean tree after `npm ci` at the root, so the pinned openspec 1.14.3 runs strict. `ateam doctor --check` and `ateam bench --selftest` exit 0. The change is archived before ready.
