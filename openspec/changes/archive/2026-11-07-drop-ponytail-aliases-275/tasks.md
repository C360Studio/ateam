## 1. Approval

- [x] 1.1 Owner approves `proposal.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit (#275). Evidence: the owner asked in Enjoy on 2026-11-07 to pick up #275, whose body states this quick-track change; Claude wrote the header at their request.

## 2. Remove the aliases

- [x] 2.1 `NormalizeMode` (`internal/state/state.go:455-464`) drops `full`, `lite`, `ultra` and the `// shortcut:`; `modeCommandRe` (`internal/hook/hook.go:61`) matches `on|off` (#275). Acceptance: `TestNormalizeModeOnOffAliases`, `TestDetectModeSwitchAndLadderPrompt`, and `TestModeAPIRejectsLegacyLevels` (400 for each legacy word); `task mutate:check` reports `detection` with `lite` re-added. Evidence: the three tests pass with the words rejected and the legacy `{"mode":"ultra"}` file loading as `on`; `go run ./cmd/ateam mutate` reports `detection` with `lite` re-added to `NormalizeMode`. The same re-add in `modeCommandRe` reports `invalid`: the regex is a package variable outside the coverage blocks, and `detectModeSwitch` (`hook.go:426-429`) passes the capture through `NormalizeMode`, so that mutant changes no behavior.
- [x] 2.2 `docs/hook-flow.md`, `.agents/skills/ponytail/SKILL.md`, and the harness mode-switch scenario drop the alias text (#275). Acceptance: `task spec:check` and `ateam doctor --check` pass.

## 3. Verify

- [x] 3.1 `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0 (#275). Evidence: all three exit 0 at b2d7197; the first `task verify` run failed lint (revive function-length, 85 of 80 statements in `TestServerEndpoints`), fixed by moving the legacy-mode check into `TestModeAPIRejectsLegacyLevels`. `ateam state mode lite` exits 1 with `invalid mode "lite"`, and `ateam state debt` no longer lists the #270 shortcut.
