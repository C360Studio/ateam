## 1. Approval

- [x] 1.1 Owner approves `proposal.md` and `design.md` by filling `Approved-by: <name> (<YYYY-MM-DD>)` and removing this hold in the same edit, ruling on design Open Questions 1-3 (#333). Evidence: the owner approved in Enjoy on 2026-11-08 and took the recommended option on each: OQ1 A (unconfigured keeps `{main, master}`), OQ2 A (the board refuses with 409 and names the file), OQ3 no (no row in the generated adopter AGENTS.md). Claude wrote the header at their request.

## 2. Settings file

- [x] 2.1 `internal/state/settings.go` `LoadSettings` (D1 A).
  - Premise: no settings reader exists. The strict-decode precedent is `internal/review/json.go:84-86`, and forge validation uses `normalizeForgeConfig` (`state.go:2459`).
  - Acceptance: table tests (`// spec: openspec/specs/harness/spec.md#Requirement: Committed team settings file`) cover these cases:
    - a missing file returns defaults with `ok=false`;
    - an unknown key, version 2, a bad `merge_method`, trailing data, a file over 64 KiB, and an invalid forge each return an error.
  - `task mutate:check` reports `detection` when `DisallowUnknownFields` is removed, when the version check is removed, and when an enum case is added.
  - Evidence: `TestLoadSettingsMissingFile` and `TestLoadSettings` (21 cases) pass with `-race`. `task mutate:check` verdicts: `detection` for dropping `DisallowUnknownFields` (`TestLoadSettings/unknown_key`), dropping the version check (`TestLoadSettings/version_2`), and adding `"rebase-merge"` to the enum (`TestLoadSettings/bad_merge_method`).
- [x] 2.2 Doctor: `CheckParity` (`internal/bootstrap/bootstrap.go:229-238`) fails on an invalid file and warns when `protocol.md` names a different merge command (D2).
  - Acceptance: one doctor test for each case, and none when the file is absent.
  - Mutation that removes the mismatch check is detected.
  - Evidence: `TestDoctorTeamSettings` (absent, invalid, valid, mismatch, match) passes with `-race`; the warning is `bootstrap.CheckMergeMethodDocs`, appended in `runDoctor`. `task mutate:check`: `detection` for forcing the mismatch check off (`.../protocol_naming_another_method_warns`) and for dropping `checkTeamSettings` from `CheckParity` (`.../invalid_file_fails_the_check`).

## 3. merge_method

- [x] 3.1 Replacer in `templateFiles` (`reviewrecord.go:68-89`) over `templates.go:637,643` (D2 A).
  - Acceptance: with no file, and with `squash`, output is byte-identical to `defaultTemplateFiles()`.
  - `merge` yields `--merge --auto` / `"do":"merge"`, and `rebase` yields `--rebase --auto` / `"do":"rebase"`, in AGENTS.md, rules-index and protocol.
  - `review_record: aip` output is unaffected.
  - Each substring appears in the constants exactly once.
  - Mutation that swaps the `gh`/Gitea mapping is detected.
  - Evidence: `TestInitMergeMethod` (no file, no key, squash: byte-identical to `defaultTemplateFiles()`; merge, rebase: rule and protocol text, no `squash` left, no doctor warning), `TestInitMergeMethodAIPUnaffected`, `TestInitInvalidSettingsFails`, and `TestMergeMethodSwapTextsAppearOnce` pass with `-race`. `task mutate:check`: `detection` for swapping the `gh`/Gitea replacements (`TestInitMergeMethod/merge`).
- [x] 3.2 Hand edits to `README.md:270,273`, `docs/daily-use.md:88-89`, and `docs/adoption.md` (the file, its keys, precedence). Acceptance: `task spec:check` passes, and a reviewer reads the edits.
  - Evidence: README § 8 (now at `:285,294`), `docs/daily-use.md:88-89`, and a new `docs/adoption.md` "Team settings" section plus the commit-statuses override line; `task spec:check` prints `openspec: strict ok` and `holds: ok`. Reviewer read pending.

## 4. Moved keys

- [ ] 4.1 `CommitStatusesEnabled(root)` (`commitstatus.go:61-62`; callers `cmd/ateam/main.go:396,470` and `internal/mcpserver/server.go:353`) follows the D4 precedence.
  - Acceptance: a test runs env {unset, `0`, `1`, `yes`} × file {absent, true, false} against an independent truth table.
  - Mutations that flip the env-first order or the default are detected.
- [ ] 4.2 A committed `forges` list is authoritative in `Refresh` and `refreshForgeConfig` (`state.go:2319-2324`, `:2438-2453`), and the board API returns 409 (`internal/web/server.go:433-436`).
  - Acceptance: a state test shows a committed list beating detection and `ATEAM_FORGE`, and no list keeping today's behaviour.
  - A web test covers the 409, and a UI check confirms the error is shown (OQ4).
  - Mutation that drops the committed-list branch is detected.

## 5. default_branch

- [ ] 5.1 `GitStatus.DefaultBranch` plus `isDefaultBranch` replace the literals at `prune.go:29` and `state.go:679,794,819,884,1057,2040,2141,2201,2205,2528,2550,2574,2575,2964,3098` (D3, per OQ1).
  - Acceptance: `git grep '"main"\|"master"' internal/state ':!*_test.go'` lists only the fallback constant, and `gopls references` on `isDefaultBranch` covers every predicate site.
  - A `trunk` fixture repo labels `trunk` as the default branch, prunes nothing on it, and counts ahead/behind against it.
  - The existing main/master tests pass unchanged.
  - Mutation of the fallback set is detected.

## 6. Rule and verify

- [ ] 6.1 Add the D5 row to `AGENTS.md` §3 and `.agents/rules-index.md`. Acceptance: `ateam doctor --check` row-count parity passes, and AGENTS.md stays under 24,000 bytes.
- [ ] 6.2 Final checks:
  - `task verify` is green on a clean tree, and `ateam doctor --check` and `ateam bench --selftest` exit 0.
  - The tests carry `// spec:` references that `task test:spec` resolves.
  - The change is archived before the PR is marked ready.
  - No gate file is touched, so the PR needs no gate label.
