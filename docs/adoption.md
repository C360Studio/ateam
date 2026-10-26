# Adopting ATeam in a repository

ATeam is a Go-first lab for testing agent harness configuration. Its portable coordination mechanisms can be useful in other projects, but it does not provide a validated end-to-end workflow for every language. Choose its workflow intentionally: the generated instructions favor Ponytail’s smallest-complete-change discipline, OpenSpec, worktrees, issue-linked draft PRs, and explicit handoffs.

See [how the harness runs](hook-flow.md) for the hook lifecycle and which workflow rules rely on agent cooperation.

## Feature and language boundaries

| Feature | Current behavior |
| --- | --- |
| `ateam init` | Writes the same templates regardless of language across Claude Code, Codex CLI, Gemini CLI, and Antigravity (`agy` via `.agents/hooks.json` and `.agents/plugins/ateam/`), including ~500k token context/auto-compaction defaults where supported (`.claude/settings.json`, `.codex/config.toml`, `.gemini/settings.json`). Does not inspect manifests to select a toolchain, create a Taskfile, install tools, or migrate existing configuration. Existing client configuration is always preserved, including with `--force`; that flag may overwrite other generated files; missing files are added and ignore entries are appended. |
| Hooks, modes, pickup/handoff, forge state, board | Operate on repository/session metadata rather than a language compiler. This is architectural portability, not evidence of complete support for every project or agent version. |
| `ateam test-spec` | Validates attached `// spec:` references on Go Test/Fuzz/executable Example and Vitest declarations against exact OpenSpec requirement headings. This is syntactic attribution, not semantic coverage. Other test frameworks/languages need project-owned enforcement; initialization does not create a Taskfile gate. See [the protocol](../.agents/protocol.md#test-to-spec-attribution). |
| `ateam review` | Optional project-approved model comparisons and independent review evidence. Requires explicit policy; does not launch models, enforce a runtime sandbox, or install branch protection. See [model review](model-review.md). |
| `ateam mutate` | Requires a non-test `.go` target, an external mutant copy, and a working Go test environment. Rejects other source extensions before starting tests. There is no Python, Rust, or JavaScript mutation runner. |
| `ateam state debt` | Scans only `.go`, `.ts`, `.js`, `.svelte`, `.sh`, `.yml`, `.yaml`. Skips `_test.go` and `.test.ts`, plus generated/dependency directories such as `vendor`, `node_modules`, `dist`, and `build`. Python, Rust, Java, JSX, and TSX files are skipped, not diagnosed as unsupported. |
| Shortcut syntax | A line must begin with optional whitespace followed by `//`, `#`, or `--`, then `shortcut:`. The first comma separates the limit from its revisit trigger. This is a text scan, not a language parser; trailing code comments and HTML comments are not recognized. |
| `ateam task` / `ateam_task` | Requires a project-owned `Taskfile.yml` and the Task executable to run targets. Target discovery handles two-space-indented top-level tasks, not the full YAML/includes feature set. Missing Taskfiles or unknown targets produce errors. |
| `ateam doctor` | Checks local file/configuration parity, schema budget, measured hook latency, whether the running `ateam` binary's embedded VCS commit is behind `HEAD`, and whether duplicate `ateam serve` processes are running. Does not validate your language toolchain, task coverage, or actual live-agent integration. The current latency failure threshold is 25ms; <10ms is the design target. |
| `ateam bench` | Runs built-in offline self-tests (`--selftest`, `--rescore`) and paired Go benchmark experiments (`run`, `compare`, `rescore`) with command or native API adapters (`claude`, `codex`, `gemini`) and versioned instruction bundles ([benchmarks/README.md](../benchmarks/README.md)). Does not benchmark arbitrary non-Go repositories or prove general agent performance gains on real project work. |

Sources: [scanner](../internal/state/shortcut.go), [initialization](../internal/bootstrap/bootstrap.go), [task runner](../internal/taskrunner/taskrunner.go), [mutation runner](../internal/mutate/mutate.go), [CLI](../cmd/ateam/main.go), and [benchmark fixtures](../internal/bench/bench.go).

## Hooks outside the project settings

A project that pins its client settings file (for example a review manifest that hashes `.claude/settings.json`) can register ateam's hooks elsewhere and set `ATEAM_HOOK_SCOPE=user` in the environment that runs `ateam doctor`. `ateam hook` itself reads no settings file. With the knob set, a client whose project hook file lacks the hooks passes when one of its local or user-scope files carries them all (`.claude/settings.local.json` is project-local and untracked, not user scope):

| Client | Project file | Local and user-scope files doctor reads, in order |
| --- | --- | --- |
| Claude Code | `.claude/settings.json` | `.claude/settings.local.json`, `~/.claude/settings.json` |
| Codex CLI | `.codex/hooks.json` | `~/.codex/hooks.json` |
| Gemini CLI | `.gemini/settings.json` | `~/.gemini/settings.json` |
| Antigravity | `.agents/hooks.json` | none; doctor reports that `agy` has no user-scope hooks doctor reads |

The file must be valid JSON and carry every required `ateam hook --event <event>` command and the client's `--agent`, the same checks as the project file. Hooks split across files fail: each file is checked on its own, so a partial or malformed file never passes. Doctor's JSON names the file that satisfied each client under `hook_sources`. Unset, doctor reads only the project files, as before; any other value is a violation. `~` is the home directory from `os.UserHomeDir` (`$HOME` on Unix), and Codex's `$CODEX_HOME` is not read. Gemini's context file (the nested `context.fileName`, else the top-level `contextFileName`) must be `AGENTS.md`: the project file's value wins when it sets one, as Gemini gives project settings precedence; only when it sets none and the knob is set does `~/.gemini/settings.json` decide, and the violation names the file whose value was checked. Doctor checks files, not whether the client loads them: Codex loads project hooks only from a trusted `.codex/` layer.

## Command dependencies

A single compiled executable does not mean every command works without external tools.

| Operation | Requirements |
| --- | --- |
| Build ATeam from source | Go at the version in `go.mod` (currently 1.26.4), Node/npm for the embedded UI, and Task if following the Taskfile build path. Install UI dependencies first with `npm ci` in `ui/`. |
| Run hooks | The `ateam` executable on the agent's PATH, readable repository files, and writable local state for events that save it. No subprocesses or network calls in the hook handler. |
| Inspect live Git/worktree state | Git and a repository. Some hook paths read Git metadata directly; full reconciliation invokes Git. |
| Execute project tasks | Task plus the tools used by those tasks. Go is not required merely because a project task runs a Python or JavaScript test command. |
| Test-to-spec attribution | Go-only scanning uses the compiled binary; Vitest scanning also needs Node and TypeScript at the target repository’s `ui/node_modules/typescript`. Supported test filenames and wrapper limitations are in the protocol. |
| Go mutation / benchmark behavioral self-tests | Go toolchain and a usable build cache. These tests spawn `go test`; offline means no model API calls, not that a toolchain can be absent. |
| Forge state/issue operations | Network access and suitable authentication; GitHub state operations use `gh`, while Gitea supports its API/token and `tea` paths. Local-only work does not require forge access. |
| Review setup / route | Local files and the compiled binary; no network or model launch. The owner supplies approved model identities and comparisons. |
| Review evidence check | Git, a clean checkout matching the live PR head and containing the current default-branch base, plus network access to the forge API. GitHub accepts `GH_TOKEN`, then `GITHUB_TOKEN`, with optional `gh auth token --hostname github.com` fallback only for the default GitHub API. Gitea accepts `GITEA_TOKEN`, then `ATEAM_GITEA_TOKEN`; it does not use `tea` for review evidence. On either forge `ATEAM_TOKEN_READ` (else `ATEAM_TOKEN_WRITE`) is used first; forge writes (PR handoff comments, label bootstrap) use `ATEAM_TOKEN_WRITE` (else `ATEAM_TOKEN_READ`), and `ateam doctor` warns while one token serves both roles. Public readable evidence may work without authentication. |
| Serve the board | The compiled binary with embedded UI assets; Node is a build dependency, not a requirement to serve those assets. |
| herdr plugin (optional) | herdr 0.9.5 or newer, `ateam` on the PATH herdr gives plugin commands, and a one-time `herdr plugin link "$PWD/.agents/herdr"` (absolute path). Without herdr nothing changes: every herdr behavior is gated on `HERDR_ENV=1`. |

## Adoption steps

1. Build/install ATeam and make the executable available to your agents. Building ATeam is separate from building the project you want agents to edit.
2. Review your existing agent instructions and settings. Run `ateam init` in the intended repository; review its diff before using `--force`, which overwrites generated documents/adapters but preserves client settings. Before `--force` replaces a hand-edited `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` or `.agents/rules-index.md`, it copies the old bytes to `<name>.ateam-backup-<YYYYMMDD-HHMMSS>` beside the file and lists those paths under `backed_up` in its JSON output; move your own rules back from there. Follow the [MCP migration guide](mcp.md#migrating-existing-configurations) to merge registration into existing files.
3. Adapt the instructions to your language and actual delivery policy. Generated references to Go tooling, mutation checks, and verification targets are not automatic implementations of those checks. Keep canonical instructions and adapter/rule-index parity consistent.
4. Create a `Taskfile.yml` exposing your project's checks. For example, a Python project's `verify` task could run its existing lint/type/test commands. Do not copy ATeam's own Taskfile unchanged: it expects this repository's `cmd/`, `internal/`, `go.mod`, and Svelte `ui/` layout.
5. Run `ateam doctor --check`, inspect `ateam task list`, and execute the project checks. Separately verify that your installed agent versions actually load the intended instructions, hooks, and MCP settings. A parity pass is not an integration certification.
6. Use pickup and handoff for continuity. Handoff can post to the open PR; use `--no-post` for a local-only checkpoint.

For non-Go mutation evidence, the shared protocol describes a manual backup/checksum approach. That is a workflow responsibility, not a built-in equivalent of `ateam mutate` for another language.

## Optional herdr adapter

When agents run inside [herdr](https://herdr.dev) panes, ATeam delegates three things and keeps the rest local:

| Delegated to herdr | Stays local and authoritative |
| --- | --- |
| Pane identity: the writer lease records `HERDR_PANE_ID` and `HERDR_WORKSPACE_ID`, and the different-writer warning names the pane. | Hooks: the hook path only reads env; it never calls herdr. |
| Worktree creation: `herdr worktree create --cwd "$PWD" --branch <agent>/<topic> --path "$PWD/.worktrees/<agent>-<topic>" --no-focus` groups the claim as a herdr workspace. | Worktree truth: `git worktree list` and the agent-prefixed branch. |
| Toasts: the plugin runs `ateam state herdr on-event` on `worktree.created` and `pane.agent_status_changed`, toasting each new `HALT`/`RED`/`BLOCKED` caveat or Hold once. Actions open pickup and spec-queue popups. | State and Holds: `.ateam/state.json`, `openspec/` tasks, and `Hold:` lines. |

Link the plugin once with `herdr plugin link "$PWD/.agents/herdr"`; `ateam` must be on herdr's PATH. `ateam doctor` does not read the manifest; herdr reports manifest problems in `herdr plugin list`. Plugin command output goes only to `herdr plugin log`, which is why pickup and spec-queue are popups. Verified against a headless herdr 0.9.5 server: event payloads, `pane get` output, popup cwd, plugin state dir, and the toast calls. Seeing a toast or popup render in an attached TUI was waived by the owner.

## Optional model review

Initialization installs canonical developer/reviewer contracts but leaves model policy unconfigured. To adopt the stronger-reviewer gate:

1. Have the owner approve exact model identities, explicit stronger-than relationships, reviewer/owner forge accounts, and required checks. Provider separation is optional; it never substitutes for a stronger reviewer. At the model ceiling, approval requires an explicit scoped owner exception.
2. Follow [model-review setup](model-review.md#configure-an-approved-policy), run `ateam review setup --policy approved-policy.json`, and inspect the generated policy, cards, and pointers. Select the specified model in each native client yourself.
3. Land the initial configuration in a separate owner-authorized PR to the repository default branch. The checker reads policy from that trusted baseline; the adoption PR cannot authorize itself with candidate policy.
4. For subsequent work, run `ateam review route`, including every actual implementation model after escalation. Have an independent reviewer record current verification and post fresh evidence, then run `ateam review check` on the clean current PR head. See [review evidence and exceptions](model-review.md#review-and-record-evidence).

The commands validate eligibility and supplied evidence; they do not automatically enforce a forge merge restriction. Adopting projects must choose where to require the check in their delivery workflow. See [upgrade and troubleshooting](model-review.md#upgrade-and-troubleshooting) for existing installations and stale provider pointers.

## Ponytail and measurement

The decision ladder is guidance for choosing the smallest complete solution. Its Go implementation injects compact prompts and records a repository discipline mode. `lite`, `full`, and `ultra` vary the wording; `off` suppresses ladder injection but does not erase repository rules or stop all state bookkeeping. These modes do not select a model, change its reasoning budget, or enforce obedience.

The benchmark self-tests (`ateam bench --selftest` and `--rescore <selftest.json>`) establish whether the supplied fixtures and checks behave as expected. Paired experiments (`ateam bench run|compare|rescore`, documented in [benchmarks/README.md](../benchmarks/README.md)) run counterbalanced A/A calibrations and A/B instruction or bundle comparisons against isolated Go tasks, and `ateam bench rescore <run.json>` reruns the Go grader against saved candidate source without invoking the adapter again. Small pilot tasks and calibration runs validate the measurement harness; they do not prove general productivity or quality gains on arbitrary production repositories.
