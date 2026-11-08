# Design: `team-settings-333`

## Context

Opened in this session on `main` e5a6bc2. The owner ruled on #333 on 2026-11-08 (issue comment). The `required_checks` rename is split out to #336.

- `internal/bootstrap/reviewrecord.go:68-89`: `templateFiles` swaps whole text constants when `review_record: aip`. `:92-107` reads the policy. The only caller is `InitRepo` (`internal/bootstrap/bootstrap.go:67`), which skips files that already exist unless `--force` is given (`:89-96`).
- The squash wording lives at `internal/bootstrap/templates.go:637` `closeRuleRow` (`squash auto-merge`) and `:643` `protocolCloseStep` (`gh pr merge --squash --auto`, `{"do":"squash",…}`). ateam's own copies are at `AGENTS.md:90,100`, `.agents/rules-index.md:41`, `.agents/protocol.md:45`, `README.md:270,273`, and `docs/daily-use.md:88-89`. The AIP variants (`reviewrecord.go:22,28`) open no PR, so `merge_method` does not apply to them. No Go code merges a PR.
- Gitea `MergePullRequestForm.Do` accepts `In(merge,rebase,rebase-merge,squash,fast-forward-only,manually-merged)` (gitea `services/forms/repo_form.go`).
- `internal/state/commitstatus.go:61-62`: `CommitStatusesEnabled()` returns true unless `ATEAM_COMMIT_STATUSES` is `"0"`. Callers are `cmd/ateam/main.go:396,470` and `internal/mcpserver/server.go:353`.
- The forge list lives at `state.go:614-616` (`Forge`, `Forges`, `ForgesConfigured`). `refreshForgeConfig` (`:2438-2453`) re-seeds it from `DetectForge` (`forge.go:58-96`, which reads `ATEAM_FORGE`/`GITEA_URL`) until `ForgesConfigured` is set. `UpsertForge`/`RemoveForge` (`:2456-2505`) are called only by the board API (`internal/web/server.go:433-436`).
- Default-branch literals, found by grep because gopls does not index string literals:
  - Predicates: `prune.go:29` and `state.go:1057,2040,2141,2201,2205,2574,2964,3098`.
  - Detection: `:2550` (`origin/main`, then `origin/master`).
  - A hardcoded ref: `:2575` `"main..."+wt.Head`, which gives wrong ahead/behind counts in a `master` repo today.
  - Empty-branch labels: `:679,794,819,884,2528`.
  - `bench.go:672` is a fixture and out of scope.
- Readers of the detected default branch, from `gopls references`: `DefaultRemoteRef` (`state.go:562`) is read at `cmd/ateam/main.go:471`, `commitstatus.go:589`, and `state.go:1042,2552,2761,3046,3099`.
- `internal/review/json.go:84-86` rejects unknown fields, so a new key in `review-policy.json` breaks older binaries. `ValidatePolicy` (`policy.go:147-149`) requires tiers.

## Goals / Non-Goals

Goals:
- One committed, non-gate home for team settings: `merge_method`, `default_branch`, `commit_statuses`, and `forges`.
- A missing file changes nothing.

Non-goals:
- The `required_checks` rename (#336).
- ateam's own limits, branch prefixes, and the worktree directory.
- Any key that sets how strong a merge gate is.

## Decisions

**D1. File and parser.** The file is `.agents/ateam.json`: `{"version":1, "merge_method", "default_branch", "commit_statuses", "forges"}`, with every key optional.
- A) **Recommended:** add `internal/state/settings.go` with `LoadSettings(root) (Settings, bool, error)`. It decodes with `DisallowUnknownFields`, rejects trailing data, and caps the file at 64 KiB. It validates the version and the enums, and validates `forges` with the existing `normalizeForgeConfig`. Bootstrap already imports state. Cost: about 60 lines plus tests.
- B) A new `internal/settings` package. It gives a cleaner seam, but adds a package for four keys.
- C) Grow `review-policy.json`. Rejected by ruling 1: it is a gate file, decodes strictly, and requires tiers.
- Failure split: hooks and `Refresh` fall back to defaults when the file is invalid, and never block a session. Doctor `--check` fails on an invalid file, and `ateam init` returns its error.
- Adopter seam: a team that never writes the file gets today's behaviour. A typo surfaces in doctor, not in a hook.

**D2. `merge_method` rendering.** The values map as follows:
- `squash` (default) → `gh pr merge --squash --auto` / `"do":"squash"`
- `merge` → `--merge` / `"merge"`
- `rebase` → `--rebase` / `"rebase"`

Gitea's `rebase-merge` has no `gh` equivalent and is not offered.
- A) **Recommended:** in `templateFiles`, after the AIP branch and only when `record != aip`, run a `strings.NewReplacer` over the three exact substrings in `closeRuleRow` and `protocolCloseStep`: the `gh` flag, the Gitea `do` value, and `squash auto-merge`. Apply it to AGENTS.md, rules-index and protocol. With `squash` the replacer changes nothing, so the output stays byte-identical.
- B) Template functions such as `closeRuleRowFor(method)`. Clearer, but they touch every user of the constants and the AIP swap keys.
- ateam's own README and daily-use docs are not generated. They are edited by hand to say "the configured method (`squash` by default)".
- Adopter seam: changing the key does not rewrite docs that are already committed. Doctor **warns** when `.agents/protocol.md` lacks the expected `gh pr merge --<method> --auto`, and names `ateam init --force` as the fix.

**D3. `default_branch`.** Add `GitStatus.DefaultBranch`, resolved in `SnapshotGit`, and a helper `st.isDefaultBranch(b)`.
- A) **Recommended:**
  - When it is configured, only that name is the default branch, detection probes `origin/<name>`, and `:2575` counts against it.
  - When it is not configured, the predicate keeps today's `{main, master}` set and probe order, and labels fall back to `"main"`.

  A missing file stays byte-identical.
- B) The literal reading of ruling 3: when nothing is configured, only the name detected from `DefaultRemoteRef` counts. A repo with both branches would then stop treating `master` as default, which changes behaviour without any file (OQ1).
- The handoff slot keys at `:679,794,819,884` use the resolved name, so a repo that sets `trunk` gets `agent@trunk` slots.

**D4. Precedence for moved keys.**
- `commit_statuses`: the env var `ATEAM_COMMIT_STATUSES` set to `"0"`/`"1"` beats the file, and the file beats the default `true`. Other env values count as unset, which matches today. The signature becomes `CommitStatusesEnabled(root string)`, with three callers.
- `forges`:
  - A) **Recommended:** a committed list is authoritative. `Refresh` copies it into the cache and sets `ForgesConfigured`, so detection and `ATEAM_FORGE` no longer set the list. The board API returns 409 "forges are committed in .agents/ateam.json". With no committed list, today's detect-then-UI path stays.
  - B) The UI writes `ateam.json`. A web form would then dirty the tree (OQ2).
  - Tokens never go in the file. `ATEAM_FORGE` in merge-check (`cmd/ateam/review.go:211`) is untouched.

**D5. Rule.** Add one `review` row to §3 of `AGENTS.md` and to `.agents/rules-index.md`, taking §3 from 12 to 13 rows (AGENTS.md is 14,356 of 24,000 bytes). The row names homes for four kinds of value:
- `.agents/ateam.json` holds committed team settings.
- `review-policy.json` or gate-protected workflow env holds keys that can weaken a merge gate.
- `ATEAM_*` env vars hold per-machine, per-session, secret or test values, or a documented override.
- `.ateam/` is cache only.

The generated templates are not changed (OQ3).

## Risks / Trade-offs

- With an invalid file, hooks fall back to defaults. A broken `forges` list therefore drops to detection silently until doctor runs.
- Strict decoding means an older binary rejects a newer key. A version bump is the way out.
- If the constants' wording drifts, the D2 substring swaps silently match nothing. A test asserts that each substring appears exactly once.

## Migration Plan

No migration is required: no file means no change. A team that wants a non-default method writes the file, then either runs `ateam init --force` (which backs up the harness docs) or edits the Close text by hand. Doctor's warning points there. A forge list already in `state.json` is ignored only once a committed list exists.

## Open Questions

1. **(ruled 2026-11-08: A)** D3: when nothing is configured, keep today's `{main, master}` set (A, recommended, honours ruling 1), or use only the detected name (B, the literal reading of ruling 3)?
2. **(ruled 2026-11-08: A)** D4 forges: should the UI refuse with 409 once the list is committed (A, recommended), or write `ateam.json` (B)?
3. **(ruled 2026-11-08: no)** Should the generated adopter AGENTS.md also get a settings-home row? Recommended: no, since `docs/adoption.md` and doctor cover it.
4. **(verify, no ruling)** Does the board UI render a 409 body? Checked in task 4.2.
