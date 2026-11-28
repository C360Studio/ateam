# ATeam (`ateam`)

**A harness for coding agents: one shared setup, durable session state, and review gates across Claude Code, Codex CLI, Gemini CLI, and Antigravity (`agy`).**

`ateam` is a single static Go binary with lifecycle hooks, a two-tool MCP server, and a local web board. You write your instructions, skills, and workflow rules once, then run one agent or switch between several across git worktrees without losing state, context, or test discipline. It is an experimentation lab: its fixtures check the harness itself and do not show that it improves agent performance (see [adoption boundaries](docs/adoption.md)).

## Languages today

Go is the primary target. Other languages get the language-neutral layer plus what this table lists:

| Language | Supported | Not supported |
| --- | --- | --- |
| Go | Every command: `ateam mutate`, `ateam test-spec`, `ateam state debt`, `ateam bench` tasks | None; see the limits in the adoption guide |
| Rust | `state debt` scans `.rs` and skips `target/`; `test-spec` gates `#[test]`-style attributes with no Rust toolchain (macro-generated tests fail closed) | `ateam mutate` is Go only: run `cargo-mutants` by hand ([recipe](docs/adoption.md#rust-repositories)); no Rust developer or reviewer contract; no Rust `bench` tasks |
| Python | `state debt` scans `.py` | `test-spec`, mutation checks, `bench` tasks |
| Any | Hooks, MCP, pickup and handoff, forge state, OpenSpec tracks, the merge gate, and `ateam doctor` read repository and session metadata, not a compiler | `ateam init` detects no language and writes no `Taskfile.yml`; you supply the `verify` target |

Limits and scanner details: [feature and language boundaries](docs/adoption.md#feature-and-language-boundaries).

## What you get

- **One shared agent setup.** A single `AGENTS.md` and canonical skills in `.agents/skills/`. `ateam init` and `ateam doctor` keep the thin adapters, `<10ms` hooks, MCP registration (under 300 schema tokens), and ~500k compaction defaults in sync for all four agent CLIs ([how the harness runs](docs/hook-flow.md), [MCP](docs/mcp.md)).
- **Session continuity.** `ateam state pickup` and `ateam state handoff` reconcile live git worktrees, writer leases, GitHub or Gitea issues and draft PRs, `Hold:` blockers in OpenSpec tasks, and `// shortcut:` debt markers. Pickup also names the next issue in milestone order (`next=`) and counts queue drift (`queue=`).
- **Right-sized planning.** [OpenSpec](https://github.com/Fission-AI/OpenSpec) changes take a `quick` or `design` track, trivial edits skip them, and one owner `Approved-by:` checkpoint gates the code.
- **Guardrails.** Ponytail, a "smallest complete change" decision ladder, as an `on` or `off` mode; non-destructive Go mutation checks (`ateam mutate`); test-to-spec attribution (`ateam test-spec`); and commit-bound review gates that need a reviewer not weaker than the implementer (`ateam review`).
- **Optional coordinator.** The `/coordinate` skill (a slash command in your agent CLI, not an `ateam` subcommand) moves a milestone through developer and reviewer runs and asks you once for every owner decision. Hold-my-beer mode hands some of those decisions to a judge ([daily guide](docs/daily-use.md#7-coordinate-a-milestone-optional)).
- **Benchmarks.** `ateam bench` tests whether harness changes alter outcomes on reproducible Go tasks ([benchmarks/README.md](benchmarks/README.md)).

Ponytail modes change injected guidance, not model settings or agent compliance. Native skill and hook activation evidence and same-model cross-runtime comparisons are [roadmap work](https://github.com/C360Studio/ateam/issues/16).

## Prerequisites and Installation

### Install a release (adopters)

Go alone installs a tagged release; no clone, Node, or Task is needed:

```bash
go install github.com/c360studio/ateam/cmd/ateam@v0.4.7
ateam version   # ateam v0.4.7 (module)
```

`(module)` marks a `go install` of a tag. Such a binary has every command and hook, but its embedded board UI is empty, so `ateam serve` serves only the JSON API; build from source for the board. GitHub releases carry no prebuilt binaries. To pin one release per repository, point hooks at a repository-local binary, or start sessions below the checkout root, read [pinning a release](docs/adoption.md#pinning-a-release-per-repository).

### Build from source

You need Go 1.26.4+ (the version in `go.mod`) and Node.js 20+ with npm, which builds the board UI embedded in the binary. [Task](https://taskfile.dev), and `gh` or `tea` with `GITEA_TOKEN` for forge sync, are optional ([forge tokens](docs/adoption.md#forge-tokens)).

```bash
git clone https://github.com/C360Studio/ateam.git
cd ateam
(cd ui && npm ci)
task build
```

Without Task, build the UI and binary by hand:

```bash
(cd ui && npm ci && npm run build)
rm -rf internal/web/dist && cp -R ui/build internal/web/dist
go build -o bin/ateam ./cmd/ateam
```

Put `./bin/ateam` on the `PATH` of your agents, for example `ln -sf "$PWD/bin/ateam" ~/.local/bin/ateam`. Prefer `task build`: it stamps the release and commit that `ateam version` prints.

## Quick start

Run these in the repository you want agents to work in. A repository that already has its own instructions or hooks should follow the [brownfield guide](docs/brownfield.md) instead.

1. **Initialize.** `--github` or `--gitea` also creates the standard issue labels.

   ```bash
   ateam init
   ```

   It writes `AGENTS.md`, `CLAUDE.md`, `.agents/` (protocol, rules index, seven skills, five role contracts), the hook, MCP, and compaction settings for each agent CLI, and `openspec/`. Existing client configuration is kept, even with `--force`. It writes no `Taskfile.yml`; [`.agents/README.md`](.agents/README.md) maps every file to its agent.

2. **Add a `verify` target.** The shipped skills and protocol run `task verify`, and `ateam doctor --check` fails without one. Put your own checks in `Taskfile.yml`:

   ```yaml
   version: '3'

   tasks:
     verify:
       desc: Run every check a pull request must pass
       cmds:
         - go test ./...   # replace with your project's own checks
   ```

3. **Check the setup.** It exits 1 on a violation, and warnings never fail it. It checks local files and settings, not that each agent CLI loads them or that your `verify` target is meaningful.

   ```bash
   ateam doctor --check
   ```

   Abridged output from an empty repository:

   ```json
   {
     "hook_latency_budget_us": 10000,
     "hook_latency_us": 83,
     "mcp_schema_bytes": 1027,
     "mcp_schema_tokens": 257,
     "passed": true,
     "violations": null,
     "warnings": [
       "forge reads and writes share one token; set ATEAM_TOKEN_READ and ATEAM_TOKEN_WRITE to separate tokens scoped to each role"
     ]
   }
   ```

4. **Start working.** `ateam state pickup` (or `/pickup` in any agent CLI) reconciles live git, worktrees, forge state, and holds. `ateam state handoff` (or `/handoff`) records the checkpoint before you stop. The whole flow, from clarify and claim through review, merge, and cleanup, is in the [daily developer guide](docs/daily-use.md).

## Documentation map

| Read | When |
| --- | --- |
| [docs/adoption.md](docs/adoption.md) | Before initializing another repository: language boundaries, Rust, command dependencies, pinning, team settings, the merge gate |
| [docs/brownfield.md](docs/brownfield.md) | The repository already has `AGENTS.md`, `CLAUDE.md`, hooks, or a `Taskfile.yml` |
| [docs/daily-use.md](docs/daily-use.md) | Day to day: planning tracks, claim, handoff, review, merge, `/coordinate` |
| [docs/hook-flow.md](docs/hook-flow.md) | What each lifecycle hook reads, writes, and returns |
| [docs/mcp.md](docs/mcp.md) | Registering and checking the two MCP tools |
| [docs/model-review.md](docs/model-review.md) | Model routing, review records, `design_paths`, hold-my-beer, `review_record` |
| [docs/adr/](docs/adr/0001-review-gate-scope.md) | Why: review gate scope (0001), tiers and bands (0002), role catalog (0003) |
| [benchmarks/README.md](benchmarks/README.md) | Paired benchmark runs, native adapters, instruction bundles |

## Using the Web Board (`ateam serve`)

```bash
ateam serve   # then open http://127.0.0.1:8420 (--addr 127.0.0.1:9000 picks another port)
```

- View and manage work items as a **Kanban board** (`Backlog`, `In Progress`, `Review`, `Done`) or a **table**.
- Sync issues and PRs from GitHub or Gitea (`Sync Forge`, or `ateam state sync`).
- Inspect OpenSpec changes, `Hold:` blockers, and `// shortcut:` items. The spec panel's **Approve** writes the owner `Approved-by:` edit to the change's branch through the forge, never to a local worktree.
- Switch the Ponytail mode (`on`, `off`) or set `ATEAM_API_TOKEN` from the profile menu. Without a token the board answers only requests addressed as `localhost`, `127.0.0.1` or `[::1]`; reaching it by another hostname (a tunnel, a LAN name) needs the token.
- Receive live webhooks at `/api/webhooks/github` (`X-Hub-Signature-256`) and `/api/webhooks/gitea` (`X-Gitea-Signature`) when `ATEAM_WEBHOOK_SECRET` is set. Webhooks always target the served checkout.

Writes carry `X-ATeam-Repo` to reject stale repository selections. The read-only routing panel is described in [model-review](docs/model-review.md#dashboard-routing-display).

## Command Reference

`ateam --help`, `ateam state --help`, and `ateam review` print flags; `task --list` prints the Taskfile targets.

| Command | What it does |
| --- | --- |
| `ateam init [--force] [--github\|--gitea]` | Create missing harness files; `--force` backs up replaced documents first |
| `ateam doctor [--check]` | Check file parity, the under-300-token MCP schema, hook latency, and a `verify` target in `Taskfile.yml` |
| `ateam version [--json]` | Release and build commit; `--json` adds a capabilities list |
| `ateam state pickup [--agent A --session S]` | Reconcile live git, worktrees, forge, and holds with the last handoff; prints `next=`, `queue=`, `since-review=` |
| `ateam state handoff --goal "..." [--no-post]` | Record the 5-field checkpoint, release the writer lease, comment on the open PR |
| `ateam state next` | Print the next issue in milestone order, why earlier ones were skipped, and unplaced issues |
| `ateam state queue-check` | List queue findings (stale blockers, unplaced issues, overloaded milestones); advice only |
| `ateam state read-handoff <agent>@<branch>` | Print one slot's handoff without changing state |
| `ateam state prune [--dry-run]` | Remove your clean, merged worktrees and their branches |
| `ateam state status\|board\|sync\|mode\|writer\|herdr` | Print the cache; print the board (`--scope --limit --after`); sync the forge; get or set `on\|off`; print the writer lease (`--json`); herdr popups and toasts (`HERDR_ENV=1`) |
| `ateam state holds\|debt\|spec-queue\|spec-check [--base REF]` | List `Hold:` items; list `// shortcut:` markers; show in-flight OpenSpec changes; validate them |
| `ateam review setup\|route\|check\|compare\|gate [--awaiting]` | Model policy, routing, review evidence, reviewer strength, and the merge check (`task merge:check`) |
| `ateam review request\|notify\|whole` | Start a reviewer and send its result back; record a whole-repository review as a tag |
| `ateam test-spec [--base REF]` | Check test citations against OpenSpec requirement headings (Go, Vitest, Rust); `--base` gates only changed tests |
| `ateam mutate -pkg P -test T -file F -mutant M` | Non-destructive `go test -overlay` mutation check (Go only) |
| `ateam bench [--selftest]`, `ateam bench run\|compare\|rescore\|export` | Offline self-tests; paired benchmark experiments |
| `ateam hook --event <event> [--agent A]` | Lifecycle hook: `session-start`, `before-prompt`, `pre-compact`, `turn-end` |
| `ateam mcp`, `ateam serve`, `ateam task [list\|<target>]` | Two-tool stdio MCP server; local web board; list or run `Taskfile.yml` targets |

Targets you will use most: `task verify` (all gates on a clean tree), `task build`, `task merge:check -- <pr>`, `task spec:check`, `task spec:queue`, `task test:spec`, and `task mutate:check -- ...`.

## Repository Layout

```text
├── AGENTS.md, CLAUDE.md       # Canonical instructions; CLAUDE.md imports @AGENTS.md
├── .agents/                   # README, protocol, rules-index, skills/ (7), contracts/ (5), hooks.json,
│                              #   plugins/ateam/ (Antigravity); this repo also has hosts/, herdr/, review-policy.json
├── .claude/ .codex/ .gemini/  # Hook, MCP, and compaction settings and thin skill adapters
├── .mcp.json                  # Claude Code MCP registration
├── openspec/                  # specs/ (current truth) and changes/ (in flight, archived)
├── docs/                      # Guides and docs/adr/ decision records
├── benchmarks/                # Experiment configs, bundles, and reports
├── cmd/ateam/, internal/      # CLI entrypoint; bench, bootstrap, hook, mcpserver, mutate, review, state,
│                              #   taskrunner, testspec, web
├── ui/                        # Svelte 5 + SvelteKit source for the board
├── scripts/, .github/, .gitea/  # merge-check and gate-files scripts; CI workflows
└── Taskfile.yml, go.mod, revive.toml, package.json
```

## Origins

Ponytail supplied the decision ladder and discipline ideas adapted into Go here. SemEngine informed shared-state handoff and pickup. [OpenSpec](https://github.com/Fission-AI/OpenSpec) supplies the spec and change workflow. [BMAD](https://github.com/bmad-code-org/BMAD-METHOD) prompted the opt-in `design` track and the human approval checkpoint; ateam keeps a declared planning depth and a fixed-shape architecture artifact and omits personas, phases, and mandatory PRDs. The lab combines these ideas to explore harness configuration, without claiming that their effectiveness has already been measured.
