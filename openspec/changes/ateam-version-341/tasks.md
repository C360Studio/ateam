## 1. Approval

- [ ] 1.1 Owner approves `proposal.md` by filling in `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#341). Hold: OPEN-Q proposal awaits owner approval

## 2. Key and read order

- [ ] 2.1 `Settings.AteamVersion` (`json:"ateam_version,omitempty"`) in `internal/state/settings.go`. `isReleaseTag` moves from `internal/bootstrap/pin.go` into state and is exported for bootstrap.
  - Acceptance: `TestLoadSettings` gains a valid case and malformed cases (`0.3.9`, `v0.3`, `v01.2.3`, `v0.4.0-rc.01`, `v0.3.9+b`).
  - Mutation that removes the key's validation is detected.
- [ ] 2.2 `PinStatus` (`pin.go:19`) reads the key, then the file. When both exist and disagree, it warns that the key wins.
  - Acceptance: a table test covers neither, key only, file only, both agree, and both disagree. Each case's expected pin and finding is written out by hand.
  - The existing `pin_test.go` cases pass unchanged.
  - Mutations that swap the read order or drop the disagreement warning are detected.
  - `cmd/ateam/pin_test.go` gains a doctor case for the key.

## 3. Docs and rule

- [ ] 3.1 `README.md:98` shows `jq -r .ateam_version .agents/ateam.json` and marks the file as deprecated. `docs/adoption.md` § Team settings lists the key. The AGENTS.md §3 row and the `.agents/rules-index.md` row list `ateam_version`.
  - Acceptance: `ateam doctor --check` parity passes, and AGENTS.md stays under 24,000 bytes.

## 4. Verify

- [ ] 4.1 `task verify` is green on a clean tree after `npm ci` at the root, so the pinned openspec 1.14.3 runs strict. `ateam doctor --check` and `ateam bench --selftest` exit 0. The change is archived before ready.
