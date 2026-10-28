## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md`, fills `Approved-by: Coby (2026-10-28)`, and removes the approval hold in the same edit. Rulings: Open Question 1 → D1 A (doctor-only tag resolution; the hook gains no subprocess), 2 → D2 A (`LookPath` of the Claude hook command's argv[0]), 3 → pseudo-versions in scope (#251).

## 2. Implementation

- [ ] 2.1 Module-install label (`internal/state/state.go:25-61`). Make `BuildRevision` testable over an injected `*debug.BuildInfo`. Return the tag form when `Main.Version` is a release tag and no vcs settings exist. Acceptance: a table test over fake build infos (tagged, pseudo-version if ruled in, `(devel)`, vcs-stamped) asserts each label, and a mutation that drops the tag branch reports `detection`.
- [ ] 2.2 Stale-binary warning (`internal/bootstrap/bootstrap.go:960-1038`, `internal/hook/hook.go:346-359`). Use the tag form per D1 and the D3 advice with the resolved path. Acceptance: doctor resolves the tag through a temp repo carrying that tag and reports behind-HEAD. The hook test asserts the module-install line and that no `git` process runs (fake `PATH` `git` that fails the test). Mutation: drop the tag resolution → `detection`.
- [ ] 2.3 `hook_binary` in doctor (`cmd/ateam/main.go:574-583`, near `DiskBuildStatus`) per D2. Acceptance: a fake `PATH` `ateam` script printing a different version from `./bin/ateam` warns, and equal versions do not. Outside the ateam checkout, nothing warns. Mutation: drop the `PATH` lookup → `detection`.
- [ ] 2.4 Behind-origin (`internal/state/state.go:2400-2406`, `:2625`, `:2873`) per D4. Acceptance: pickup on a temp repo whose `main` trails its origin by 2 prints `behind=2` on the `main` line and on `[ateam]`, and prints neither when they are even. Mutation: swap the left/right counts → `detection`.
- [ ] 2.5 Docs: one sentence in `docs/brownfield.md` § 1 on qualifying a candidate with `hook_binary`, `disk_build`, and `HEAD`. Update `README.md:75` where it describes what `go install` prints.
- [ ] 2.6 `task verify` green on a clean tree, `ateam doctor` parity green, `hook_latency_us` within budget, and `ateam bench --selftest` green.
- [ ] 2.7 Archive the change before marking the PR ready (#251).
