# Adopting ATeam in a repository

ATeam is a Go-first lab for testing agent harness configuration. Its portable coordination mechanisms can be useful in other projects, but it does not provide a validated end-to-end workflow for every language. Choose its workflow intentionally: the generated instructions favor Ponytail’s smallest-complete-change discipline, OpenSpec, worktrees, issue-linked draft PRs, and explicit handoffs.

See [bringing ateam into an existing repository](brownfield.md) for a worked path through `init` and `doctor`, and [how the harness runs](hook-flow.md) for the hook lifecycle and which workflow rules rely on agent cooperation.

## Feature and language boundaries

| Feature | Current behavior |
| --- | --- |
| `ateam init` | Writes the same templates regardless of language across Claude Code, Codex CLI, Gemini CLI, and Antigravity (`agy` via `.agents/hooks.json` and `.agents/plugins/ateam/`), including ~500k token context/auto-compaction defaults where supported (`.claude/settings.json`, `.codex/config.toml`, `.gemini/settings.json`). Leaves existing role contracts under `.agents/contracts/` in place; [link them to the base contracts](brownfield.md#existing-specialist-contracts). Does not inspect manifests to select a toolchain, create a Taskfile, install tools, or migrate existing configuration. Existing client configuration is always preserved, including with `--force`; that flag replaces generated documents and adapters, first copying a hand-written `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` or `.agents/rules-index.md` to `<name>.ateam-backup-<YYYYMMDD-HHMMSS>`; missing files are added and ignore entries are appended. |
| Hooks, modes, pickup/handoff, forge state, board | Operate on repository/session metadata rather than a language compiler. This is architectural portability, not evidence of complete support for every project or agent version. |
| OpenSpec tracks (`spec-queue`, `spec-check`) | Language-independent workflow files under `openspec/`. Non-trivial changes declare `quick` (`proposal.md` + `tasks.md`, no `design.md`) or `design` (adds `design.md` via `/design` and `.agents/contracts/architect.md` when a Requirement, cross-capability change, adopter surface, boot/durability path, or gate file is in scope), gated on `Approved-by:` in `proposal.md` and Task 1's `Hold: OPEN-Q proposal awaits owner approval`. `ateam state spec-check` enforces task-scoped `Hold:` placement in Go, warns without failing past 60 lines (`proposal.md`) or 120 lines (`design.md`), and runs `openspec validate --all --strict --no-interactive` when `node_modules/.bin/openspec` or `openspec` on `PATH` exists (printing a skip notice otherwise). `ateam init` writes `openspec/config.yaml` as `schema: spec-driven` plus the `rules: design:` that `.agents/skills/design/SKILL.md` also carries, so `openspec instructions design` shows them; an existing `openspec/config.yaml` is left untouched. Triggers are reviewed by humans/agents, not diff-parsed, and `ateam review gate` prints a non-blocking note when a code PR's OpenSpec change lacks `Approved-by:`. |
| `ateam test-spec` | Validates attached `// spec:` references on Go Test/Fuzz/executable Example and Vitest declarations against exact OpenSpec requirement headings. This is syntactic attribution, not semantic coverage. A brownfield repository starts with `--base <ref>`, which gates only tests changed since `<ref>` ([brownfield guide](brownfield.md#5-gate-new-tests-first)). Other test frameworks/languages need project-owned enforcement; initialization does not create a Taskfile gate. See [the protocol](../.agents/protocol.md#test-to-spec-attribution). |
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
| Install a release | Go only: `go install github.com/c360studio/ateam/cmd/ateam@<tag>` (no Node or Task). The board UI is not embedded in a module install. |
| Build ATeam from source | Go at the version in `go.mod` (currently 1.26.4), Node/npm for the embedded UI, and Task if following the Taskfile build path. Install UI dependencies first with `npm ci` in `ui/`. |
| Run hooks | The `ateam` executable the hook command names (a bare `ateam`, as `init` writes it, resolves on the agent's PATH), readable repository files, and writable local state for events that save it. No subprocesses or network calls in the hook handler. |
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

Forge reads (board, sync, pickup, review evidence, the merge check) and writes (PR handoff comments, commit statuses, label bootstrap) can use separate tokens: `ATEAM_TOKEN_READ` for reads and `ATEAM_TOKEN_WRITE` for writes. Either one alone serves both roles. When neither is set, `ateam doctor` warns that reads and writes share one token, in CI too; the warning is intended and does not fail `--check`. The merge check treats the variables differently on each forge:

- **Gitea:** `ateam review gate` and `ateam review check` resolve `ATEAM_TOKEN_READ`, then `ATEAM_TOKEN_WRITE`, then `GITEA_TOKEN`, then `ATEAM_GITEA_TOKEN`.
- **GitHub:** the merge check ignores the role variables and uses `gh`'s own resolution (`GH_TOKEN`, `GITHUB_TOKEN`, then `gh auth`). `ateam review check` does honor them first.

CI sets neither role variable, so both gates run on the job token. A role token with less access makes the check fail rather than pass.

## The merge gate on GitHub and Gitea

Call the released binary from your own CI or `task verify`; do not copy ateam's workflows or `Taskfile.yml`. `ateam review gate [PR]` exits 0 on pass, 1 on a failed rule, and 2 when a read was unavailable (`scripts/merge-check.sh` is only a wrapper around it). It checks that main requires a status check with the up-to-date rule, that a non-draft code PR carries `implemented-by:` and `reviewed-by:` with a record at its head, and that no `class:flake` issue is open.

| Behavior | GitHub | Gitea |
| --- | --- | --- |
| Required check | An active ruleset on `main` requires a status check named `Required` with `strict_required_status_checks_policy`. | Branch protection on `main` requires the context `CI / Required (pull_request)`: workflow name `CI`, job name `Required`. Renaming either breaks the gate. |
| More required checks | Set `ATEAM_REQUIRED_CHECKS` to a comma-separated list, for example `Required,Merge check`, and the gate requires every name; unset, it requires `Required` alone. | The same, with full contexts, for example `CI / Required (pull_request),Merge check / Merge check (pull_request)`; unset, `CI / Required (pull_request)` alone. |
| Transport | Reads go through the `gh` CLI. | HTTPS only; plain HTTP is refused except on a loopback address. |
| Token in CI | Whatever `gh` finds (`GH_TOKEN` or `GITHUB_TOKEN`). | The Actions job token only, passed as `GITEA_TOKEN`. It cannot read the up-to-date setting, so the gate prints `strict=unverified` and a warning; a local run with an admin token checks it fully. |
| Draft pull requests | Every job is skipped on a draft and renamed off its required name, so a draft reports no result. `Gate files` still runs when a label is removed. | A `WIP:` title marks a draft. Jobs are skipped on it and renamed off their required names, so rerunning a WIP run cannot pass a required check; `edited` (leaving `WIP:`, or any other edit) starts a fresh run. Needs **Gitea 28.x**: older Gitea evaluates neither the job-name expression nor the compound `always()` on `Required`, so keep the pre-#257 workflows there. |

Local forge selection: an exported `GITEA_URL` takes priority over a github.com `origin` (unless `ATEAM_FORGE=github`), so unset it when working on a GitHub repository. Without it, the API URL comes from the `origin` host: an `ssh://` remote's port is dropped because it is the SSH port, and an `http(s)://` remote's port is kept. A forge serving HTTPS on a non-default port behind an SSH remote needs `GITEA_URL`. Forge transport errors print the full request URL, host included, so do not paste gate output from a private Gitea instance into a public place.

## Informational commit statuses

`ateam state sync` and a posting handoff (`ateam state handoff` without `--no-post`, or the MCP `ateam_state` handoff without `post: false`) set three commit statuses on the open PR head, so GitHub, Gitea, and any client that shows a PR's checks list shows them too. Sync covers each open PR whose branch has a worktree on this machine; a handoff covers its own branch's PR. They describe the PR; no gate reads them, none is a required check, and `task merge:check` is unchanged. Hooks, pickup, prune, board, the MCP pickup, and the UI never write them.

| Context | Condition (first match wins) | State and description |
| --- | --- | --- |
| `ateam/proposal` | the PR touches no `openspec/changes/` directory | not posted |
| `ateam/proposal` | a touched change has an empty `Approved-by:` or an open `OPEN-Q` hold at the head | `pending` `awaiting owner approval (OPEN-Q): <id>` |
| `ateam/proposal` | every touched change is approved | `success` `approved by <Approved-by>: <id>` |
| `ateam/handoff` | no handoff on the branch | not posted |
| `ateam/handoff` | the handoff checkpoint is not the PR head | `pending` `handoff stale: at <sha7>, head <sha7>` |
| `ateam/handoff` | the handoff tree was dirty | `pending` `handoff at <sha7> has uncommitted changes` |
| `ateam/handoff` | otherwise | `success` `handoff at <sha7>, <time UTC>` |
| `ateam/review-record` | no review record | not posted |
| `ateam/review-record` | the record names another commit | `pending` `record at <sha7> is behind head <sha7>` |
| `ateam/review-record` | `APPROVE` or `REQUEST CHANGES` | `success` or `failure` `<verdict> by <agent> (<model>, <effort>) at <sha7>` |
| `ateam/review-record` | any other verdict, or none | `pending` `record at <sha7> has no single verdict` |

A context is written only when its state or description differs from the latest status with that context on the head, so a repeated sync posts nothing. Writes use the write token (`ATEAM_TOKEN_WRITE`, see above), which needs permission to set commit statuses. A rejected write prints one warning and never changes an exit code; `ateam doctor` then warns from the cache, naming the context and HTTP or `gh` exit code but never the token, and `--check` still passes. `REQUEST CHANGES` shows as a red check although nothing requires it. Statuses refresh only on sync and posting handoffs, so they lag a bare `git push` until the next one. Set `ATEAM_COMMIT_STATUSES=0` to turn all of this off.

## Optional external task lease

A daemon that assigns tasks to checkouts can show its own lease to agents through one file. ATeam reads the file and never writes, refreshes, or clears it, makes no call to the daemon, and takes no ownership decision from it: the writer lease, the handoff slot, and prune ignore it. If the daemon writes nothing, nothing changes.

Write `.ateam/task-lease.json` atomically (temp file plus rename), or point `ATEAM_TASK_LEASE_FILE` at an absolute or root-relative path:

```json
{"id":"0f3a","issue":284,"holder":"worker-1","expires_at":"2026-11-03T12:00:00Z","source":"factory"}
```

All five fields are required and unknown fields are ignored. `issue` is a positive integer, `expires_at` is RFC 3339, and each string is at most 256 bytes with no control characters. The file is at most 64 KiB. Pickup and `session-start` print `Task lease: issue #284 holder=worker-1 expires=2026-11-03T12:00:00Z (source factory)`, with `expired` appended once `expires_at` passes. A file that breaks these rules prints one `WARNING: task lease file unreadable (<reason>)` line and pickup continues.

A handoff can name its lease in `Addresses` as `aip-lease:<id>`. The prefix is case-insensitive, and the token reads `A-Z`, `a-z`, `0-9`, `_`, and `-`, so keep the `id` to those. On a different `id`, pickup prints `handoff lease <id> is not the current task lease <id2>`. Only the first token is compared.

Before assigning a task, run `ateam state writer --json` in the checkout. It prints one object with `state`, a fixed-text `reason`, and, when a lease was read, the stored `agent`, `session_id`, `pid`, `last_seen`, `herdr_pane_id`, `herdr_workspace_id`, `host`, and `pid_ns`. All four states exit 0; a non-zero exit means the command failed, and bad usage exits 2. Treat `unknown` as occupied.

| `state` | Meaning |
| --- | --- |
| `none` | No lease file. |
| `live` | Same host and PID namespace, live PID, within the 10-minute TTL. |
| `stale` | Same host and namespace with an exited PID, or a lease from elsewhere (or with no host) past the TTL. |
| `unknown` | A lease from elsewhere within the TTL, a live PID past the TTL on the same host, or an unreadable or incomplete lease. An idle session left open reads this until it exits. |

Limits: `last_seen` changes only at session start, each prompt, and turn end, so a turn longer than ten minutes on another host reads `stale`. Off Linux there is no `pid_ns`, so two containers sharing a hostname compare as one place. `stale` across hosts assumes clocks roughly agree.

The check is advisory. ATeam is not the final authority: only `session-start` checks for another writer, and `before-prompt` and `turn-end` overwrite the lease without a check, so a second session takes it at its first prompt and the last writer wins. Nothing blocks edits, and a session yields only if its agent acts on the warning. A session can also take the lease between your check and your assignment. Give each task its own worktree, because that is what keeps writers apart.

To tell whether the command exists, run `ateam version --json`, which prints `version`, `commit`, and a `capabilities` list containing `state-writer-json` and `task-lease-file`. Older versions ignore the flag and print plain text with exit 0, so output that does not parse as JSON means an older ATeam.

## Optional herdr adapter

When agents run inside [herdr](https://herdr.dev) panes, ATeam delegates three things and keeps the rest local:

| Delegated to herdr | Stays local and authoritative |
| --- | --- |
| Pane identity: the writer lease records `HERDR_PANE_ID` and `HERDR_WORKSPACE_ID`, and the different-writer warning names the pane. | Hooks: the hook path only reads env; it never calls herdr. |
| Worktree creation: `herdr worktree create --cwd "$PWD" --branch <agent>/<topic> --path "$PWD/.worktrees/<agent>-<topic>" --no-focus` groups the claim as a herdr workspace. | Worktree truth: `git worktree list` and the agent-prefixed branch. |
| Toasts: the plugin runs `ateam state herdr on-event` on `worktree.created` and `pane.agent_status_changed`, toasting each new `HALT`/`RED`/`BLOCKED` caveat or Hold once. Actions open pickup and spec-queue popups. | State and Holds: `.ateam/state.json`, `openspec/` tasks, and `Hold:` lines. |

Link the plugin once with `herdr plugin link "$PWD/.agents/herdr"`; `ateam` must be on herdr's PATH. `ateam doctor` does not read the manifest; herdr reports manifest problems in `herdr plugin list`. Plugin command output goes only to `herdr plugin log`, which is why pickup and spec-queue are popups. Verified against a headless herdr 0.9.5 server: event payloads, `pane get` output, popup cwd, plugin state dir, and the toast calls. Seeing a toast or popup render in an attached TUI was waived by the owner.

## T3 Code hosts

[T3 Code](https://github.com/pingdotgg/t3code) in worktree-per-thread mode creates each thread's worktree under T3's own home directory, not `.worktrees/`. The directory is T3's `worktreesDirectory` setting (default `~/.t3/worktrees/<project>/`). The thread starts on a temporary `t3/<hex8>` branch, and T3 renames it after the first message by its `branchNamingMode` setting (`static`, `semantic`, or `custom`, with the prefix `branchNamePrefix`, default `t3`). In the live run on 2026-11-07, the default naming renamed `t3/cd3b2a8b` to `t3/adoption-docs-correction` while the first turn ran. The session-start `[ateam]` line therefore names the temporary branch, so read the branch again before you open the draft PR. That first line also shows `forge=` empty and `synced=never`, because the hook runs before the new worktree's first sync; run `/pickup` before you rely on forge state. Keep the `t3/` prefix for every provider. The prefix is one value per project, not one value per provider, so a static `claude` prefix would also name Codex threads `claude/…`. The writer lease, the handoff, and `implemented-by:` record the agent. The Claim gate exempts the host-created worktree and branch; the agent still opens the draft PR and the OpenSpec change. Pickup lists the worktree through `git worktree list`, and because `t3/` names no agent, its line names the owner from the worktree's handoff, else its writer lease, else `owner unknown`. `ateam state prune` keeps it as `no agent prefix (host-created branch)`; remove it from T3 or with `git worktree remove`. Put `npm ci` or `ateam doctor` in the repository's `t3.json` `runOnWorktreeCreate` script.

Do not use T3's revert-to-checkpoint, its commit dialog, or its PR refresh on a claimed branch. These user actions run `git restore`, `git reset`, and `git clean -fd`, which can delete uncommitted work and `.ateam/` state. Commit and push from the thread instead. T3's checkpoints are commits on hidden refs written through a temporary index, so they do not change `HEAD`, the real index, or `git status`. T3's storage cleanup can remove a thread's worktree. With `worktreeKeepWhen: tracked-changes`, it also runs `git clean -ffdx` in that worktree, which deletes `.ateam/state.json`. It never deletes a branch. Run `/handoff` before you archive a thread. In the live run on 2026-11-07, archiving a thread with T3's default storage settings left its worktree, its `.ateam/state.json`, and its local and remote branch in place.

T3 does not remove `CLAUDE_CODE_SESSION_ID`. T3 starts the Claude CLI with a new `--session-id` and exports that id, and it sends the thread id separately as `T3CODE_THREAD_ID`. Handoffs record the CLI session id. `ateam review request` reads `return_to` from `T3CODE_THREAD_ID`. T3 also sets `CLAUDE_CODE_ENTRYPOINT=t3-code`. On macOS, T3's desktop server process found `ateam` in `~/.local/bin`, and `gh` authenticated through the keyring with `GH_TOKEN`, `GITHUB_TOKEN`, and `ATEAM_TOKEN_*` all unset. Not checked: a machine with no keyring login, where the hooks can fail and show no error.

Whether ATeam's hooks fire depends on the provider. The table comes from T3 Code's source at main @ 6577c6b6 on 2026-11-06. The Claude row was checked in a live run on 2026-11-07: hooks, skills, and `T3CODE_THREAD_ID` reached the model in a project-root thread and in a worktree thread. No live session was run for the other providers:

| Provider | How T3 runs it | Hooks, skills, `T3CODE_THREAD_ID` |
| --- | --- | --- |
| Claude | Agent SDK with no `settingSources`, which defaults to every source | hooks and skills load; the thread id is in the process env |
| Codex | one shared `codex app-server`, the thread's cwd per thread | project hooks run once `.codex/hooks.json` is trusted in Codex (T3 does not bypass hook trust); `AGENTS.md` and skills load; the thread id reaches only the shells Codex runs |
| Antigravity | T3's own ACP runtime with a private `GEMINI_HOME` that isolates hooks | no ATeam hooks and no thread id: run `ateam state pickup` at the start and `/handoff` at the end by hand; `.agents/skills` is probably visible |
| Gemini CLI | no built-in driver, only through T3's remote ACP registry | unverified |

T3 also names threads and branches with short `codex exec` runs in the thread's cwd that leave hooks enabled. With Codex as T3's text-generation model, those runs fire ATeam's hooks. In the live run on 2026-11-07, a Codex session that had already exited held the writer lease in a new Claude worktree thread, and it saved no rollout file. Pickup then named the worktree's owner as `codex` until the Claude session ran `/handoff`. The cause is probably the branch-naming run, but this was not traced. Run `/handoff` early in a T3 worktree thread, so that pickup takes the owner from the handoff and not from the lease. Unverified: whether Codex's hook processes see the thread id.

## Optional review launch hosts

The review policy's `review_launch` says who starts a cross-agent reviewer: `manual` (the default) or `agent` ([ADR 0002](adr/0002-model-tiers-and-effort-bands.md#addendum-review-launch-owner-ruling-on-295-2026-11-06)). `ateam review request <pr>` prints one JSON object naming the eligible reviewer runs, the prompt `/review-pr <n>`, the head sha the record must bind to, the detected host and `return_to`. The head sha is always the PR's, read from the forge, never the local checkout's `HEAD`. With no `--implementer` flags it also reads the PR's `implemented-by:` line; it exits 2 when it cannot read the PR. Under `manual` it never runs anything.

The host-specific part is an optional data file, `.agents/hosts/<host>.toml`, with a `[launch]` and a `[notify]` table. Each holds `command`, an argument array run without a shell, or `tool`, text printed for the agent to follow where the host has no one-command form. Placeholders `{pr}`, `{prompt}`, `{sha}`, `{agent}`, `{model}`, `{effort}` and `{return_to}` are replaced inside each argument in one pass. `{agent}`, `{model}` and `{effort}` come from `--reviewer` or the first eligible run. A missing manifest means print only. A malformed one fails with fixed text that does not echo the file. Manifests decide what `agent` mode executes, so this repository lists `.agents/hosts/` with its gate files.

| Host | Detected by | `return_to` |
| --- | --- | --- |
| override | `ATEAM_HOST=<host>`, checked first | per host |
| T3 Code | `T3CODE_THREAD_ID` (Claude's process and Codex's shells; never Antigravity) | the thread id |
| herdr | `HERDR_ENV=1` | `HERDR_PANE_ID` |
| Enjoy | `ENJOY_MCP_ENDPOINT` | none in env: pass `--return-to <conversation id>` |
| terminal | none of the above (`cli`) | none: a terminal reviewer such as `codex exec` runs synchronously |

A reviewer is read-only, so it should not take the writer lease of the checkout it starts in. With `ATEAM_ROLE=review` in its environment, its hooks write no lease at session start, before a prompt, or at turn end. The herdr launch sets the variable with `herdr pane split --env`. Start a terminal reviewer as `ATEAM_ROLE=review codex exec …`. The T3 Code launch sets no variable, because a per-thread value cannot reach Codex's hooks through T3's shared `codex app-server`. Instead, the requester creates the detached `.worktrees/<agent>-review-pr-<n>` checkout and starts the thread there with `workspaceStrategy` `existing_worktree`, so the reviewer's lease stays in its own checkout. Two cases are not covered. Enjoy's `start_conversation` takes no environment or working directory, so an Enjoy reviewer still takes the project checkout's lease; run `/handoff` in that checkout afterward. T3's own `codex exec` naming runs still take a thread worktree's lease (see T3 Code hosts).

When the manifest's `launch` names `{return_to}` and none is known from `--return-to` or the host's environment, `ateam review request` exits 2 naming the flag, so no reviewer starts without a way back; `--dry-run` prints the same line as a warning and exits 0.

The reviewer finishes with `ateam review notify --record <file> --return-to <id> <pr>`. Notify sends only `review record for PR #<n> landed at <sha>; run ateam state sync, then ateam state pickup` through the manifest's `notify` command, as its last argument. On a host with only a notify tool it prints the tool to call. On `cli` it sends nothing. The record body never enters the message. `ateam state sync` fetches comments only for open PRs whose branch has a local worktree, within the existing comment budget, and pickup appends `; review record at <sha>, <verdict>, <agent> (<model>, <effort>)` to that worktree's line, with `, stale` once the head moves past the sha. A fetch that fails or runs out of budget leaves the line as it was. Each checkout caches in its own `.ateam/state.json`, so the reviewer's sync never reaches the developer: the record arrives when the developer's checkout syncs. `ateam state pickup` skips the fetch while that cache is under 30 seconds old and session-start never fetches, which is why the notify message asks for `ateam state sync` first.

This repository ships `enjoy.toml`, `herdr.toml`, and `t3code.toml` as references. Enjoy is MCP-only (`start_conversation`; `queue_message` within a project, `message_project_conversation` across projects). herdr 0.9.5 starts an agent only in an existing pane and takes the prompt afterwards, so its launch is `tool` text while its notify is the command `herdr agent prompt {return_to}`; herdr's manifest has not been run live (#299). T3 Code's `t3` CLI has `t3 thread start <project> <message> --provider --model` and `t3 thread send <thread> <message>`, which queues the message without interrupting a busy thread. Its launch is `tool` text for the in-thread MCP tool `t3_thread_launch`, whose `modelSelection.options` sets effort before the first turn (`effort` for `claudeAgent`, `reasoningEffort` for `codex`); `t3 thread start` takes no effort and starts at the model's default, which the merge gate can reject. Its notify is the command `t3 thread send {return_to}`. Not run live.

## Optional model review

Initialization installs canonical architect, developer, and reviewer contracts (`.agents/contracts/{architect,developer,reviewer}.md`) but leaves model policy unconfigured (`.agents/review-policy.json` routes only `developer` and `reviewer`; this repository's role catalog and default tiers live in [ADR 0003](adr/0003-roles-and-routes.md)). To adopt the stronger-reviewer gate:

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

The decision ladder is guidance for choosing the smallest complete solution. Its Go implementation injects a compact prompt and records a repository discipline mode, `on` or `off`; `off` suppresses ladder injection but does not erase repository rules or stop all state bookkeeping. These modes do not select a model, change its reasoning budget, or enforce obedience.

The benchmark self-tests (`ateam bench --selftest` and `--rescore <selftest.json>`) establish whether the supplied fixtures and checks behave as expected. Paired experiments (`ateam bench run|compare|rescore`, documented in [benchmarks/README.md](../benchmarks/README.md)) run counterbalanced A/A calibrations and A/B instruction or bundle comparisons against isolated Go tasks, and `ateam bench rescore <run.json>` reruns the Go grader against saved candidate source without invoking the adapter again. Small pilot tasks and calibration runs validate the measurement harness; they do not prove general productivity or quality gains on arbitrary production repositories.
