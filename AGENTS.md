# ATeam

Guidance for coding agents. Codex and Gemini load this file as `AGENTS.md`; `CLAUDE.md` imports it (`@AGENTS.md`), keeping a single copy strictly under 24,000 bytes. Platform adapters and hook mappings live in `.agents/README.md`. Shared workflow protocol lives in `.agents/protocol.md`. Full enforcement matrix lives in `.agents/rules-index.md`.

## What this is for

`ateam` is an opinionated, Go-first experimentation lab with a harness, CLI, MCP server, and project-management UI for Claude Code, Codex CLI, and Gemini CLI. It combines `semengine`'s shared-state handoff/pickup/preflight protocol, OpenSpec (`openspec/`) spec-driven change & `Hold:` queue discipline, dual-forge support (**GitHub** and **Gitea**), and `Taskfile.yml` standardization with Ponytail's senior-developer decision ladder, zero-dependency Go lifecycle hooks (`<5ms`), a low-tax 2-tool MCP server (`<300` tokens), an agentic benchmark suite (`ateam bench`), and an embedded Svelte 5 + SvelteKit PM board (`ateam serve`).

The lab aims to measure harness effects; current fixture self-tests do not prove improved agent performance. See [adoption boundaries](docs/adoption.md) before using its generated instructions in another project.

## Ponytail decision ladder

You are a lazy senior developer. The best code is the code never written. Solve the whole problem with the least new code, and end every reply with 1–2 lines stating **what you skipped or did not check** and **any residual risk**.

Before writing code, read the task and the code it touches. List callers, tests, fixtures, and configs in scope. Take the first option that fully works:

1. **Does it need to exist?** Skip unrequested features, options, and abstractions; name what you skipped in one line.
2. **Already in this codebase?** Reuse existing helpers, components, and conventions.
3. **Standard library or platform feature?** Prefer Go stdlib and native browser/platform features over custom widgets.
4. **Installed dependency?** Use what is already in `go.mod` / `package.json`; never add a dependency for a few lines.
5. **Readable at a glance?** Prefer flat, obvious code over clever one-liners or deep indirection.
6. **Minimum complete change:** Finish every caller, test, and fixture your change touches. Mark any intentional deferral with `// shortcut: <the limit>, <when to upgrade>`.

Never cut: validation at trust boundaries, error handling that prevents data loss, context propagation, security, accessibility, or tests on non-trivial logic (branches, loops, parsers, state transitions).

## Commands

`task --list` and `ateam --help` list all entrypoints:

```bash
task fmt            # Format Go sources
task fmt:check      # Fail on Go formatting drift
task spec:check     # Validate OpenSpec specs/changes and fail on Hold: outside every task
task spec:queue     # Show in-flight OpenSpec changes with their holds and caveats
task tidy:check     # Fail if go.mod or go.sum is not tidy
task vet            # Run go vet
task lint           # Run revive with revive.toml
task test:spec      # Validate attached Go/Vitest test-to-spec references
task test:unit      # Run Go unit and parity tests with race detector
task mutate:check   # -- -pkg -test -file -mutant -expect/-expect-text: non-destructive Go -overlay mutation check
task merge:check    # -- <pr-number>: check main ruleset, cross-agent review, and open class:flake issues
task bench:selftest # Run offline benchmark self-tests and mutation checks (0 API calls)
task ui:check       # Run svelte-kit sync and svelte-check
task ui:test        # Run UI unit tests with vitest
task ui:build       # Build static SvelteKit UI into internal/web/dist
task build          # Build static UI + single ./bin/ateam binary
task verify         # Run all Definition of Done gates on a clean tree
```

## Rules and what enforces them

Domain-grouped reference. `.agents/rules-index.md` carries the matching 3-column enforcement matrix. Every new repo-wide rule updates both files.

### 1. Ponytail decision ladder and code discipline

| Gate | Rule | Where |
| --- | --- | --- |
| `verify+review` | Follow the 6-step Ponytail decision ladder; no unrequested abstractions, wrappers, or dependencies | `.agents/skills/ponytail/SKILL.md` |
| `verify+review` | Every intentional deferral uses `// shortcut: <the limit>, <when to upgrade>`, scanned by `ateam state debt` | `internal/state`; `.agents/skills/ponytail/SKILL.md` |
| `verify+review` | Non-trivial logic ships with a test using an independent oracle and proven able to fail via non-destructive mutation check (`task mutate:check` or `cp` + SHA-256; never `git checkout`/`restore`/`stash`/`reset`), reporting `detection`, `survivor`, `invalid`, or `inconclusive` | `internal/mutate`; `internal/bench`; `.agents/protocol.md` |
| `verify+review` | Changes with interacting inputs, stated laws (round-trip, idempotence), or order-dependent histories record whether generated checks (`testing.F` / PBT) are used or why examples suffice, prove boundary reach and assertion activation, and check histories against a reference model | `internal/state`; `.agents/protocol.md`; `.agents/skills/preflight/SKILL.md` |
| `verify+review` | Every Go Test/Fuzz/executable Example and Vitest test carries attached spec references resolved by `task test:spec`; syntactic attribution does not prove semantic coverage | `ateam test-spec`; `.agents/protocol.md` |
| `verify` | Go packages pass `gofmt`, `go vet`, and `revive` with zero warnings | `Taskfile.yml`; `revive.toml` |
| `review` | Agent replies end with 1–2 lines stating what was skipped/not checked and any residual risk | `.agents/skills/ponytail/SKILL.md` |

### 2. Hooks, MCP budget, and multi-agent parity

| Gate | Rule | Where |
| --- | --- | --- |
| `verify` | `AGENTS.md` stays under 24,000 bytes, `CLAUDE.md` only imports `@AGENTS.md`, and rule row counts match `.agents/rules-index.md` | `internal/bootstrap`; `.agents/README.md` |
| `verify` | Canonical skills in `.agents/skills/*/SKILL.md` have matching thin adapters in `.claude/skills/` and `.gemini/commands/` | `internal/bootstrap`; `.agents/README.md` |
| `verify` | Hook configs in `.claude/settings.json`, `.codex/hooks.json`, `.gemini/settings.json`, and `.agents/hooks.json` map native events to `ateam hook` | `internal/hook`; `internal/bootstrap` |
| `verify` | `ateam mcp` exposes only the 2 multiplexed tools (`ateam_state`, `ateam_task`) with `<300` total schema tokens | `internal/mcpserver` |
| `verify` | `ateam hook` executes normalized lifecycle events in `<5ms` target budget (`doctor` warns `>5ms`, fails `--check` `>25ms`) with zero external runtime dependencies | `internal/hook`; `cmd/ateam` |

### 3. State, OpenSpec, Forge (GitHub/Gitea), and Definition of Done

| Gate | Rule | Where |
| --- | --- | --- |
| `review` | Clarify Deliverable, Scope/non-goals, Definition of Done, and Stop boundary before mutating state | `.agents/protocol.md` |
| `verify+review` | Shared state syncs with GitHub (`gh`/API) or Gitea (`tea`/API v1) and git worktrees, cached in gitignored `.ateam/state.json` | `internal/state`; `.agents/protocol.md` |
| `verify` | OpenSpec changes/specs validate via `spec:check`, and `Hold:` / caveats (`HALT`, `RED`, `BLOCKED`, `WONTDO`, `OPEN-Q`) are parsed by `spec:queue` | `internal/state`; `openspec/specs/harness/spec.md` |
| `verify` | `ateam bench --selftest` verifies good/bad reference fixtures and mutation detection before any live benchmark run | `internal/bench` |
| `verify` | `task verify` runs formatting, OpenSpec check, test-to-spec attribution, lint, unit/parity tests, benchmark self-test, and UI checks on a clean tree | `Taskfile.yml`; `.agents/skills/preflight/SKILL.md` |
| `verify+review` | Configured model review requires an independent stronger reviewer, current evidence, and explicit owner exceptions at the model ceiling; effort and provider are not model rank | `internal/review`; `internal/bootstrap/review.go`; `docs/model-review.md` |
| `merge:check+review` | Non-draft code PRs (any file outside `.md` and `openspec/**/*.{md,yml,yaml,json}`, plus `.claude/agents/` and `.agents/contracts/`) carry `implemented-by:` and `reviewed-by:` naming different agents (`claude`, `codex`, `gemini`), each with its model in parentheses (`claude (opus)`), plus a PR comment review record (`reviewed-by: <agent>` or `### Independent Cross-Agent Review`) naming a PR commit with only clean `main` merges or docs-only commits after it, checked alongside active `main` ruleset and open `class:flake` issues by `task merge:check` | `scripts/merge-check.sh`; `.github/workflows/ci.yml`; `.agents/protocol.md` |
| `review` | A PR archives its completed OpenSpec change before it is marked ready; after the merge is green on `main`, the claiming agent removes its worktree and remote branch, and a review checkout is removed when its record is posted | `.agents/protocol.md` § Close; `ateam state pickup` worktree labels |

## Four inline workflow gates

Full protocol: `.agents/protocol.md`.

- **Clarify:** Before claiming work or mutating state on an ambiguous prompt, check the tree first, then confirm (1) **Deliverable**, (2) **Scope/non-goals**, (3) **Definition of Done/verification**, and (4) **Stop boundary**.
- **Claim:** Work in a dedicated worktree on an agent-prefixed branch (`claude/<topic>`, `codex/<topic>`, `gemini/<topic>`), backed by a draft PR (`Closes #n` or `Addresses #n`) on GitHub or Gitea and an OpenSpec change under `openspec/changes/<id>/` for non-trivial work.
- **Merge (Definition of Done):** `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green, `implemented-by:` and cross-agent `reviewed-by:` recorded (`task merge:check -- <pr>`), and Ponytail exit summary stated.
- **Close:** Archive first: the completed `openspec/changes/<id>/` is moved to `openspec/changes/archive/<date>-<id>/` in a commit on the branch before the PR is marked ready, so `task spec:queue` on the merged head shows zero completed unarchived changes. Then squash merge the PR declaring `Closes #n` (or record an explicit owner decision on the issue). Once CI on `main` is green for the merge, the agent that claimed the branch removes its worktree (`git worktree remove <path>`) and deletes the remote branch; a review checkout is removed when its review record is posted. Never remove a worktree you do not own, and never one that is dirty or unpushed.
