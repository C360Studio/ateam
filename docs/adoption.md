# Adopting ATeam in a repository

ATeam is a Go-first lab for testing agent harness configuration. Its portable coordination mechanisms can be useful in other projects, but it does not provide a validated end-to-end workflow for every language. Choose its workflow intentionally: the generated instructions favor Ponytail’s smallest-complete-change discipline, OpenSpec, worktrees, issue-linked draft PRs, and explicit handoffs.

See [bringing ateam into an existing repository](brownfield.md) for a worked path through `init` and `doctor`, and [how the harness runs](hook-flow.md) for the hook lifecycle and which workflow rules rely on agent cooperation.

## Feature and language boundaries

| Feature | Current behavior |
| --- | --- |
| `ateam init` | Writes the same templates regardless of language across Claude Code, Codex CLI, Gemini CLI, and Antigravity (`agy` via `.agents/hooks.json` and `.agents/plugins/ateam/`), including ~500k token context/auto-compaction defaults where supported (`.claude/settings.json`, `.codex/config.toml`, `.gemini/settings.json`). Does not inspect manifests to select a toolchain, create a Taskfile, install tools, or migrate existing configuration. Existing client configuration is always preserved, including with `--force`; that flag replaces generated documents and adapters, first copying a hand-written `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` or `.agents/rules-index.md` to `<name>.ateam-backup-<YYYYMMDD-HHMMSS>`; missing files are added and ignore entries are appended. |
| Hooks, modes, pickup/handoff, forge state, board | Operate on repository/session metadata rather than a language compiler. This is architectural portability, not evidence of complete support for every project or agent version. |
| `ateam test-spec` | Validates attached `// spec:` references on Go Test/Fuzz/executable Example and Vitest declarations against exact OpenSpec requirement headings. This is syntactic attribution, not semantic coverage. Other test frameworks/languages need project-owned enforcement; initialization does not create a Taskfile gate. See [the protocol](../.agents/protocol.md#test-to-spec-attribution). |
| `ateam review` | Optional project-approved model comparisons and independent review evidence. Requires explicit policy; does not launch models, enforce a runtime sandbox, or install branch protection. See [model review](model-review.md). |
| `ateam mutate` | Requires a non-test `.go` target, an external mutant copy, and a working Go test environment. Rejects other source extensions before starting tests. There is no Python, Rust, or JavaScript mutation runner. |
| `ateam state debt` | Scans only `.go`, `.ts`, `.js`, `.svelte`, `.sh`, `.yml`, `.yaml`. Skips `_test.go` and `.test.ts`, plus generated/dependency directories such as `vendor`, `node_modules`, `dist`, and `build`. Python, Rust, Java, JSX, and TSX files are skipped, not diagnosed as unsupported. |
| Shortcut syntax | A line must begin with optional whitespace followed by `//`, `#`, or `--`, then `shortcut:`. The first comma separates the limit from its revisit trigger. This is a text scan, not a language parser; trailing code comments and HTML comments are not recognized. |
| `ateam task` / `ateam_task` | Requires a project-owned `Taskfile.yml` and the Task executable to run targets. Target discovery asks `task --list-all --json`, so Task v3 `includes:` targets are listed and accepted (#206); when the Task executable is absent or fails, it falls back to two-space-indented top-level tasks in the root `Taskfile.yml`. Missing Taskfiles or unknown targets produce errors. |
| `ateam doctor` | Checks local file/configuration parity, schema budget, and measured hook latency (warns above the 10ms target, fails `--check` above 100ms; best of three warm runs). `AGENTS.md` and `.agents/rules-index.md` must agree with each other on numbered rule sections and table rows, and `.agents/protocol.md` must keep the phrases the harness depends on, but the rules themselves are yours (#158). Only the six canonical skills (`pickup`, `handoff`, `preflight`, `ponytail`, `review-pr`, `design`) get strict checks; other skills are listed as `adopter_skills` and adapter aliases as `skill_aliases` (#159). It warns about completed but unarchived OpenSpec changes, duplicate `ateam serve` processes, and a running `ateam mcp` process that serves this repository and is older than the build it is compared with (`HEAD` in the ateam checkout, the on-disk `bin/ateam` elsewhere); processes serving other repositories are listed but not warned about, and the binary-versus-`HEAD` provenance check applies only in the ateam checkout (#150, #151). Does not validate your language toolchain, task coverage, or actual live-agent integration. |
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

The file must be valid JSON and carry every required `ateam hook --event <event>` command and the client's `--agent`, the same checks as the project file. Hooks split across files fail: each file is checked on its own, so a partial or malformed file never passes. Doctor's JSON names the file that satisfied each client under `hook_sources`. Unset, doctor reads only the project files, as before; any other value is a violation. `~` is the home directory from `os.UserHomeDir` (`$HOME` on Unix), and Codex's `$CODEX_HOME` is not read. Gemini's context file (the nested `context.fileName`; Gemini CLI ignores the top-level `contextFileName`) must be `AGENTS.md`: the project file's value wins when it sets one, as Gemini gives project settings precedence; only when it sets none and the knob is set does `~/.gemini/settings.json` decide, and the violation names the file whose value was checked. Doctor checks files, not whether the client loads them: Codex loads project hooks only from a trusted `.codex/` layer.

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
2. Review your existing agent instructions and settings. Run `ateam init` in the intended repository without `--force` and merge what `ateam doctor --check` lists, as in the [brownfield guide](brownfield.md); `--force` is not the migration path. It overwrites generated documents/adapters but preserves client settings. Before it replaces a hand-edited `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` or `.agents/rules-index.md`, it copies the old bytes to `<name>.ateam-backup-<YYYYMMDD-HHMMSS>` beside the file and lists those paths under `backed_up` in its JSON output; move your own rules back from there. Follow the [MCP migration guide](mcp.md#migrating-existing-configurations) to merge registration into existing files.
3. Adapt the instructions to your language and actual delivery policy. Generated references to Go tooling, mutation checks, and verification targets are not automatic implementations of those checks. Keep canonical instructions and adapter/rule-index parity consistent.
4. Create a `Taskfile.yml` exposing your project's checks. For example, a Python project's `verify` task could run its existing lint/type/test commands. Do not copy ATeam's own Taskfile unchanged: it expects this repository's `cmd/`, `internal/`, `go.mod`, and Svelte `ui/` layout.
5. Run `ateam doctor --check`, inspect `ateam task list`, and execute the project checks. Separately verify that your installed agent versions actually load the intended instructions, hooks, and MCP settings. A parity pass is not an integration certification.
6. Use pickup and handoff for continuity. Handoff can post to the open PR; use `--no-post` for a local-only checkpoint.

For non-Go mutation evidence, the shared protocol describes a manual backup/checksum approach. That is a workflow responsibility, not a built-in equivalent of `ateam mutate` for another language.

## Forge tokens

Forge reads (board, sync, pickup, review evidence, the merge check) and writes (PR handoff comments, label bootstrap) can use separate tokens: `ATEAM_TOKEN_READ` for reads and `ATEAM_TOKEN_WRITE` for writes. Either one alone serves both roles. When neither is set, `ateam doctor` warns that reads and writes share one token, in CI too; the warning is intended and does not fail `--check`. The merge check treats the variables differently on each forge:

- **Gitea:** `ateam review gate` and `ateam review check` resolve `ATEAM_TOKEN_READ`, then `ATEAM_TOKEN_WRITE`, then `GITEA_TOKEN`, then `ATEAM_GITEA_TOKEN`.
- **GitHub:** the merge check ignores the role variables and uses `gh`'s own resolution (`GH_TOKEN`, `GITHUB_TOKEN`, then `gh auth`). `ateam review check` does honor them first.

CI sets neither role variable, so both gates run on the job token. A role token with less access makes the check fail rather than pass.

## The merge gate on GitHub and Gitea

Call the released binary from your own CI or `task verify`; do not copy ateam's workflows or `Taskfile.yml`. `ateam review gate [PR]` exits 0 on pass, 1 on a failed rule, and 2 when a read was unavailable (`scripts/merge-check.sh` is only a wrapper around it). It checks that main requires a status check with the up-to-date rule, that a non-draft code PR carries `implemented-by:` and `reviewed-by:` with a record at its head, and that no `class:flake` issue is open.

| Behavior | GitHub | Gitea |
| --- | --- | --- |
| Required check | An active ruleset on `main` requires a status check named `Required` with `strict_required_status_checks_policy`. | Branch protection on `main` requires the context `CI / Required (pull_request)`: workflow name `CI`, job name `Required`. Renaming either breaks the gate. |
| Transport | Reads go through the `gh` CLI. | HTTPS only; plain HTTP is refused except on a loopback address. |
| Token in CI | Whatever `gh` finds (`GH_TOKEN` or `GITHUB_TOKEN`). | The Actions job token only, passed as `GITEA_TOKEN`. It cannot read the up-to-date setting, so the gate prints `strict=unverified` and a warning; a local run with an admin token checks it fully. |
| Draft pull requests | Every job is skipped on a draft and renamed off its required name, so a draft reports no result. `Gate files` still runs when a label is removed. | A `WIP:` title marks a draft. Jobs are skipped on it, and `edited` (leaving `WIP:`, or any other edit) starts a fresh run. Needs **Gitea 28.x**: older Gitea ignores the compound `always()` on `Required` and could pass it over a failed `Verify`, so keep the pre-#257 workflows there. |

Local forge selection: an exported `GITEA_URL` takes priority over a github.com `origin` (unless `ATEAM_FORGE=github`), so unset it when working on a GitHub repository. Forge transport errors print the full request URL, host included, so do not paste gate output from a private Gitea instance into a public place.

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

## Plain technical prose (STE-style anti-buzzword recipe)

This recipe is opt-in. `ateam` does not enable it in its own repository or its `init` templates. The prompt borrows the intent of Simplified Technical English (STE). It does not use the ASD-STE100 dictionary and does not claim ASD-STE100 conformance.

```text
STYLE [Simplified Technical English] — Write plain, concrete technical prose; zero buzzword soup:
1. Active voice, real actor: name the package, command, file:line, or role that acts ("CheckOpenSpec rejects...", not "validation is performed to ensure...").
2. Short units: one action or idea per sentence (<=20 words for steps, <=25 for explanations); <=6 sentences per paragraph, topic sentence first.
3. Unstack noun piles (<=3 modifiers): write "timeout for comment fetches in forge.go", not "forge comment fetch timeout orchestration configuration".
4. Condition before command: "When X fails, return exit 1." Use MUST / MUST NOT for requirements, never mushy "should appropriately".
5. Ban filler: cut leverage, utilize, robust, seamless, comprehensive, holistic, streamline, facilitate, ensure, and e.g./i.e./etc. (name the exact items). One term per concept; do not rotate synonyms.
```

Wire it in one of two ways. Neither needs Go code.

- **Per-artifact OpenSpec `rules:`.** Add the prompt to the artifacts people review in your `openspec/config.yaml`. Coding turns do not load it, so it costs no tokens there. `MUST` / `MUST NOT` satisfies both the prompt and `openspec validate --strict`; you do not need `SHALL`.

  ```yaml
  rules:
    proposal:
      - "STYLE [Simplified Technical English] — ... (paste the five rules above)"
    specs:
      - "STYLE [Simplified Technical English] — ..."
    design:
      - "STYLE [Simplified Technical English] — ..."
  ```

- **On-demand `/ste` adopter skill.** Put the prompt in `.agents/skills/ste/SKILL.md` with `name: ste` and a `description` in its front matter. `ateam doctor --check` lists it under `adopter_skills`. Optional thin adapters work like the canonical ones: `.claude/skills/ste/SKILL.md` links to `../../../.agents/skills/ste/SKILL.md`, and `.gemini/commands/ste.toml` sets `prompt = "Read .agents/skills/ste/SKILL.md fully and follow it."`. Doctor checks only that their links resolve.

The prompt relies on the model to follow it. For a mechanical check, run your own prose linter, such as `vale`, from your `Taskfile.yml`. If you license a controlled-English dictionary, keep it in your own repository, not in `ateam` templates.

## Ponytail and measurement

The decision ladder is guidance for choosing the smallest complete solution. Its Go implementation injects compact prompts and records a repository discipline mode. `lite`, `full`, and `ultra` vary the wording (`ultra` here is a Ponytail mode set with `ateam state mode`, unrelated to Claude's multi-agent mode of the same name); `off` suppresses ladder injection but does not erase repository rules or stop all state bookkeeping. These modes do not select a model, change its reasoning budget, or enforce obedience.

The benchmark self-tests (`ateam bench --selftest` and `--rescore <selftest.json>`) establish whether the supplied fixtures and checks behave as expected. Paired experiments (`ateam bench run|compare|rescore`, documented in [benchmarks/README.md](../benchmarks/README.md)) run counterbalanced A/A calibrations and A/B instruction or bundle comparisons against isolated Go tasks, and `ateam bench rescore <run.json>` reruns the Go grader against saved candidate source without invoking the adapter again. Small pilot tasks and calibration runs validate the measurement harness; they do not prove general productivity or quality gains on arbitrary production repositories.
