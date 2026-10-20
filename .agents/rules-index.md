# Rules Enforcement Index

3-column audit matrix backing `AGENTS.md` § "Rules and what enforces them". Numbered sections and row counts are checked against `AGENTS.md` by `TestAgentSetupParity` (`internal/bootstrap/parity_test.go`) and `ateam doctor`.

## 1. Ponytail decision ladder and code discipline

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| `verify+review` | Follow the 6-step Ponytail decision ladder; no unrequested abstractions, wrappers, or dependencies | `internal/hook/hook_test.go` (`TestBeforePromptInjectsLadder`); code review |
| `verify+review` | Every intentional deferral uses `// shortcut: <the limit>, <when to upgrade>`, scanned by `ateam state debt` | `internal/state/shortcut_test.go` (`TestScanShortcuts`) |
| `verify+review` | Non-trivial logic ships with a test using an independent oracle and proven able to fail via non-destructive mutation check (`task mutate:check` or `cp` + SHA-256; never `git checkout`/`restore`/`stash`/`reset`), reporting `detection`, `survivor`, `invalid`, or `inconclusive` | `internal/mutate/mutate_test.go` (`TestRunOutcomes`, `TestValidateRefusals`); `internal/bench/bench_test.go` (`TestSelfTestAndMutation`) |
| `verify+review` | Changes with interacting inputs, stated laws (round-trip, idempotence), or order-dependent histories record whether generated checks (`testing.F` / PBT) are used or why examples suffice, prove boundary reach and assertion activation, and check histories against a reference model | `internal/state/prop_fuzz_test.go` (`FuzzScanHoldsMarkdown`, `FuzzReadFastGitHeadRefs`, `TestPropMergeForgeCardsHistory`); review |
| `verify+review` | Every Go Test/Fuzz/executable Example and Vitest test carries attached spec references resolved by `task test:spec`; syntactic attribution does not prove semantic coverage | `ateam test-spec`; `.agents/protocol.md` |
| `verify` | Go packages pass `gofmt`, `go vet`, and `revive` with zero warnings | `Taskfile.yml` (`fmt:check`, `vet`, `lint`) |
| `review` | Agent replies end with 1–2 lines stating what was skipped/not checked and any residual risk | `.agents/skills/ponytail/SKILL.md`; review |

## 2. Hooks, MCP budget, and multi-agent parity

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| `verify` | `AGENTS.md` stays under 24,000 bytes, `CLAUDE.md` only imports `@AGENTS.md`, and rule row counts match `.agents/rules-index.md` | `internal/bootstrap/parity_test.go` (`TestAgentSetupParity`, `TestAgentSetupParitySensitivity`) |
| `verify` | Canonical skills in `.agents/skills/*/SKILL.md` have matching thin adapters in `.claude/skills/` and `.gemini/commands/` | `internal/bootstrap/parity_test.go` (`TestAgentSetupParity`) |
| `verify` | Hook configs in `.claude/settings.json`, `.codex/hooks.json`, `.gemini/settings.json`, and `.agents/hooks.json` map native events to `ateam hook` | `internal/bootstrap/parity_test.go` (`TestAgentSetupParity`) |
| `verify` | `ateam mcp` exposes only the 2 multiplexed tools (`ateam_state`, `ateam_task`) with `<300` total schema tokens | `internal/mcpserver/server_test.go` (`TestMCPTokenBudget`) |
| `verify` | `ateam hook` executes normalized lifecycle events in `<5ms` with zero external runtime dependencies | `internal/hook/hook_test.go` (`TestHookLatencyBudget`) |

## 3. State, OpenSpec, Forge (GitHub/Gitea), and Definition of Done

| Gate | Rule | Enforcement & Test |
| --- | --- | --- |
| `review` | Clarify Deliverable, Scope/non-goals, Definition of Done, and Stop boundary before mutating state | `.agents/protocol.md` § "Work Lifecycle" |
| `verify+review` | Shared state syncs with GitHub (`gh`/API) or Gitea (`tea`/API v1) and git worktrees, cached in gitignored `.ateam/state.json` | `internal/state/forge_test.go` (`TestForgeGitHubAndGitea`); `internal/state/state_test.go` |
| `verify` | OpenSpec changes/specs validate via `spec:check`, and `Hold:` / caveats (`HALT`, `RED`, `BLOCKED`, `WONTDO`, `OPEN-Q`) are parsed by `spec:queue` | `internal/state/openspec_test.go` (`TestOpenSpecQueueAndCheck`) |
| `verify` | `ateam bench --selftest` verifies good/bad reference fixtures and mutation detection before any live benchmark run | `internal/bench/bench_test.go` (`TestSelfTest`) |
| `verify` | `task verify` runs formatting, OpenSpec check, test-to-spec attribution, lint, unit/parity tests, benchmark self-test, and UI checks on a clean tree | `Taskfile.yml` (`verify`) |
| `verify+review` | Configured model review requires an independent stronger reviewer, current evidence, and explicit owner exceptions at the model ceiling; effort and provider are not model rank | `internal/review` policy/evidence tests; `internal/bootstrap/review_test.go` setup/drift tests; `ateam review check`; substantive review |
