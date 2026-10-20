# ATeam (`ateam`)

**Bootstrap a reproducible configuration, show which settings differ between experiments, and measure the outcome through one workflow.** Users keep their preferred coding-agent runtime. ATeam helps teams replace assumptions about its configuration with evidence.

`ateam` is an **opinionated lab for experimenting with coding-agent harnesses, primarily in Go projects**. It provides a command-line tool, MCP server, and local web board for work with **Claude Code**, **Codex CLI**, **Gemini CLI**, and **Antigravity (`agy`)**.

The questions are practical: does a change to shared instructions, skills, hooks or verification improve outcomes within a runtime? With the same model and ATeam policy, do different runtimes behave differently? Shared bootstrap files alone do not prove that each runtime loaded or executed the same configuration.

Today, the benchmark workflow supports versioned instruction comparisons with shared runner/model settings, independent grading, retained failures and offline replay. Native skill/hook activation evidence and same-model cross-runtime comparisons remain roadmap work. Existing fixtures and small API calibrations do **not** establish that ATeam improves agent performance. See the [benchmark roadmap](https://github.com/C360Studio/ateam/issues/16) for delivered capabilities, open work and evidence limits.

For the event-by-event lifecycle, state changes, and design rationale, read [how the harness runs](docs/hook-flow.md). For optional stronger-model review, setup, and scoped owner exceptions, read [model routing and independent review](docs/model-review.md). For paired benchmark runs, native API adapters, and versioned instruction bundles, read [benchmark experiments](benchmarks/README.md).

## Opinions and Boundaries

ATeam adapts Ponytail’s decision ladder and supporting mechanisms into Go: question whether code needs to exist, reuse what is already available, prefer the standard library and installed dependencies, and finish the smallest complete change. Its workflow also favors isolated worktrees, issue-linked draft PRs, OpenSpec task lists, explicit handoffs, and tests demonstrated to catch wrong behavior. These are deliberate choices that adopters should review.

Ponytail modes (`lite`, `full`, `ultra`, `off`) change injected guidance. They do not guarantee agent compliance or tune model/provider settings; `off` does not remove written repository instructions or disable all harness bookkeeping.

**Go is the primary development target, not merely the implementation language.** Mutation checks only accept Go source. Coordination features are largely language-independent, but initialization does not detect or adapt to a project’s language, shortcut scanning covers a fixed set of extensions, and each adopting project must supply its own verification tasks. Other languages are not validated as complete supported workflows.

Read the [adoption and language-support guide](docs/adoption.md) before initializing another repository. The build and Taskfile commands below describe developing ATeam itself unless explicitly stated otherwise.

## The Problem

When you use more than one coding agent on a repository—or even run multiple sessions with the same agent—four practical problems show up quickly:

1. **Duplicate, drifting config files:** Claude Code reads `CLAUDE.md`, `.claude/skills/`, `.claude/settings.json`, and `.mcp.json`. Codex CLI reads `AGENTS.md`, `.agents/skills/`, `.codex/hooks.json`, and `.codex/config.toml`. Gemini CLI reads `.gemini/settings.json` and `.gemini/commands/*.toml`. Antigravity (`agy`) reads `AGENTS.md`, `.agents/skills/`, `.agents/hooks.json`, and `.agents/plugins/`. Maintaining separate copies by hand means your rules, skills, hooks, and context settings drift apart as soon as you edit one of them.
2. **Lost state between sessions:** When a session ends, compacts its context window, or hands work to another agent, the next session has to guess which branch and commit are active, whether another session is still editing the same checkout, which tasks are blocked, and what checks still need to run.
3. **Unnecessary code and weak tests:** Without explicit guardrails, coding agents often add unrequested wrapper functions, extra abstractions, or new dependencies, and write tests that pass even when the code under test is broken.
4. **Slow hooks and large tool schemas:** Lifecycle hooks that call `git` or network APIs on every prompt make the CLI sluggish. MCP servers that register dozens of tools consume thousands of context tokens on every turn before the agent does any work.

## Why `ateam` Is Built This Way

Every part of `ateam` is designed to solve one of those four problems directly:

- **One source of truth for instructions and skills:** You maintain a single `AGENTS.md` (under 24 KB) and canonical skills in `.agents/skills/<name>/SKILL.md`. `CLAUDE.md` is a one-line import (`@AGENTS.md`), and the files under `.claude/`, `.codex/`, `.gemini/`, and `.agents/{hooks.json,plugins/ateam/}` are thin pointers generated by `ateam init` and verified by `ateam doctor`.
- **Preconfigured ~500k auto-compaction defaults where supported:** To reduce long-context attention drift on 1M-token models and fire pre-compaction checkpoints before quality degrades, `ateam init` scaffolds ~500k token context/auto-compaction settings in `.claude/settings.json` (`"autoCompactWindow": 500000`), `.codex/config.toml` (`model_context_window = 500000` and `model_auto_compact_token_limit = 450000`), and `.gemini/settings.json` (`"model": {"chatCompression": {"contextPercentageThreshold": 0.5}}`). Antigravity (`agy`) does not expose a configurable compaction threshold key in `settings.json` and relies on built-in checkpointing plus manual `/compact`, `/context`, `/btw`, and subagents.
- **Shared truth in Git, OpenSpec, and GitHub/Gitea—cached locally for speed:** Issues and pull requests live on **GitHub** or **Gitea**. Specs and task checklists live in `openspec/`. Active work lives in git worktrees. `ateam` caches a local snapshot in `.ateam/state.json` (gitignored) so lifecycle hooks can read repository state without making network calls on every prompt. Sub-5ms hooks are a design target, not a guarantee for every repository size.
- **One static Go binary (`ateam`):** Lifecycle hooks (`ateam hook`), the MCP server (`ateam mcp`), state inspection and optional `herdr` integration (`ateam state`), model review checks (`ateam review`), test-to-spec attribution (`ateam test-spec`), mutation checks (`ateam mutate`), offline self-tests and paired experiments (`ateam bench`), and the web board (`ateam serve`) share one compiled binary. Hooks avoid external subprocesses, while other commands require tools such as Git, Go, or Task; see the [dependency table](docs/adoption.md#command-dependencies).
- **Two MCP tools under 300 tokens:** Instead of one MCP tool per action, `ateam mcp` exposes only `ateam_state` (pickup, handoff, debt, mode, OpenSpec checks, cached board) and `ateam_task` (list and run `Taskfile.yml` targets). Tool definitions are approximately 177 tokens by the bytes/4 estimate. See the [MCP setup and tool guide](docs/mcp.md) for registration, prompts/resources, side effects, and connection checks.
- **Source-code comments instead of separate debt files:** Intentional shortcuts are marked right where they happen (`// shortcut: <what is limited>, <when to upgrade>`) and scanned from source by `ateam state debt`. Task blockers are marked inline in `openspec/changes/<id>/tasks.md` with `Hold: <reason>`.
- **Non-destructive mutation testing (`ateam mutate`):** To verify that a unit test actually catches a bug, `ateam mutate` runs `go test -overlay` against a temporary modified file outside your repository—never touching your working tree or running destructive `git checkout` / `git restore` commands.

---

## Prerequisites and Installation

To build `ateam` from source, you need:

- **Go 1.26.4+** (the version declared in `go.mod`)
- **Node.js 20+ and npm** (used to build the static web UI embedded into the Go binary)
- **[Task](https://taskfile.dev) (`task`)** (optional, for running `Taskfile.yml` commands)
- **`gh`** (GitHub CLI) or **`tea` / `GITEA_TOKEN`** (optional, for syncing issues and PR comments with GitHub or Gitea)

Clone the repository and build the binary:

```bash
git clone https://github.com/C360Studio/ateam.git
cd ateam
(cd ui && npm ci)
task build
```

If you do not have `task` installed, you can build directly with `npm` and `go`:

```bash
(cd ui && npm ci && npm run build)
rm -rf internal/web/dist && cp -R ui/build internal/web/dist
go build -o bin/ateam ./cmd/ateam
```

Expose `./bin/ateam` on your `PATH`—for example, symlink it into `~/.local/bin/ateam` (`ln -sf "$PWD/bin/ateam" ~/.local/bin/ateam`) so `brew doctor` is unaffected and desktop hosts (such as Enjoy) pick up rebuilds in place—so Claude Code, Codex CLI, Gemini CLI, and Antigravity (`agy`) can invoke `ateam` from their hook and MCP configurations.

---

## Getting Started in 5 Steps

### 1. Initialize a repository (`ateam init`)

Run `ateam init` inside your project repository (or pass `--dir /path/to/repo`):

```bash
ateam init
```

This creates missing harness files. Existing client configuration is always preserved; `--force` can replace other generated files such as documents and adapters. `.gitignore` entries are appended separately. It installs the same opinionated templates for every language and does **not** create a `Taskfile.yml` or install project tools. Review the generated instructions and supply your own verification tasks before asking an agent to follow them:

- `AGENTS.md` and `CLAUDE.md` (`@AGENTS.md`)
- `.agents/protocol.md`, `.agents/rules-index.md`, `.agents/README.md`, and `.agents/skills/{pickup,handoff,preflight,ponytail}/SKILL.md`
- `.agents/contracts/developer.md` and `.agents/contracts/reviewer.md`
- `.agents/hooks.json` and `.agents/plugins/ateam/{plugin.json,mcp_config.json}` (Antigravity lifecycle hooks and workspace MCP plugin)
- `.claude/settings.json` (hooks + `"autoCompactWindow": 500000`), `.claude/skills/*/SKILL.md`, and `.mcp.json` (MCP registration)
- `.codex/hooks.json` and `.codex/config.toml` (MCP registration + `model_context_window = 500000` / `model_auto_compact_token_limit = 450000`)
- `.gemini/settings.json` (hooks, MCP registration, `contextFileName: "AGENTS.md"`, and `contextPercentageThreshold: 0.5`) and `.gemini/commands/*.toml`
- `openspec/config.yaml`, `openspec/specs/`, and `openspec/changes/`
- `.gitignore` entries for `.ateam/`, `.worktrees/`, `.enjoy-logs/`, and `benchmarks/runs/`

If your repository is hosted on GitHub or Gitea and you want to create the standard issue labels (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`), pass `--github` or `--gitea`:

```bash
ateam init --github
# or
ateam init --gitea
```

### 2. Check your setup (`ateam doctor`)

Check local instruction/configuration parity, estimated MCP schema size, and measured hook latency. This does not prove that each installed agent CLI consumes the configuration correctly or that your project verification tasks exist:

```bash
ateam doctor --check
```

Example output:

```json
{
  "hook_latency_us": 2761,
  "mcp_schema_bytes": 706,
  "mcp_schema_tokens": 177,
  "passed": true,
  "violations": null
}
```

Model review is opt-in: `init` installs the role contracts but does not choose models or install a policy. Follow the [adoption steps](docs/adoption.md#optional-model-review) to approve model relationships, configure routes, and establish a trusted baseline before checking review evidence.

### 3. Start or resume work (`ateam state pickup`)

When starting a session—or picking up work from an earlier session or another agent—run:

```bash
ateam state pickup
```

Or use the `/pickup` skill inside Claude Code, Codex CLI, Gemini CLI, or Antigravity (`agy`). `pickup` reconciles your live git branch, `HEAD` SHA, uncommitted changes, active git worktrees, open `Hold:` items, `// shortcut:` comments, and any commits made since the last recorded handoff.

### 4. Write code, track blockers, and verify tests

While working on a change:

- **Track multi-step specs and blockers in `openspec/`:** Put change proposals and task checklists in `openspec/changes/<change-id>/tasks.md`. If a task is blocked, add `Hold: <reason>` indented under that task item:
  ```markdown
  ## Tasks

  - [ ] 1.1 Wire webhook signature verification
    Hold: waiting on secret rotation in staging
  ```
  Inspect all in-flight changes and holds with `ateam state spec-queue` (or `task spec:queue`), and validate spec formatting with `ateam state spec-check` (or `task spec:check`).
- **Mark temporary code limits inline:** If you intentionally defer an edge case, add a comment in the code:
  ```go
  // shortcut: only first page of comments is fetched, paginate when issue comments exceed 30
  ```
  Run `ateam state debt` at any time to list matching comments in supported source files, with file and line number. The [scanner limits](docs/adoption.md#feature-and-language-boundaries) matter for other languages.
- **Prove a test fails when the code is broken (`ateam mutate`):** Copy the file you want to test to `/tmp`, introduce a bug in the copy, and run `ateam mutate` (or `task mutate:check`):
  ```bash
  ateam mutate \
    -pkg ./internal/state \
    -test TestVerifyWebhookHMAC \
    -file internal/state/forge.go \
    -mutant /tmp/forge_bad.go \
    -expect-text "VerifyWebhookHMAC"
  ```
  `ateam mutate` runs `go test -overlay` so your working tree is never modified, and reports one of four outcomes:
  - `detection`: baseline passed, mutant failed with the expected message, and post-check passed.
  - `survivor`: the mutated line was executed by the test, but the test still passed.
  - `invalid`: the mutant failed to compile, or the test never executed the mutated line.
  - `inconclusive`: the baseline test failed, timed out, or failed with an unrelated error.
- **Attribute tests to requirements:** Attach `// spec: openspec/specs/<capability>/spec.md#Requirement: Exact heading` to each Go test/fuzz/executable example and each Vitest test. Run `task test:spec` to catch missing or stale references. This checks attribution syntax and existing targets; reviewers still assess whether assertions prove the cited behavior. See [the attribution protocol](.agents/protocol.md#test-to-spec-attribution).
- **Run the full verification suite (`task verify`):**
  ```bash
  task verify
  ```
  This checks Go formatting, OpenSpec validity, `go mod tidy`, `go vet`, `revive` linting, Go unit tests with the race detector, offline benchmark self-tests, and UI type/unit tests.

### 5. Hand off work at the end of a session (`ateam state handoff`)

Before ending a session, switching agents, or compacting context, record a structured handoff:

```bash
ateam state handoff \
  --goal "Add Gitea webhook signature check (#12)" \
  --addresses "PR #15, branch claude/gitea-hmac" \
  --constraints "None" \
  --done-when "task verify passes and PR #15 is merged" \
  --checks "task verify"
```

This updates `.ateam/state.json`, releases the active session's writer lease (`.ateam/writer.json`), posts the 5-field handoff comment to the open PR on GitHub or Gitea (pass `--no-post` to stay local-only), and prints the 5-field pickup prompt for the next session.

---

## Using the Web Board (`ateam serve`)

Start the embedded project management board:

```bash
ateam serve
```

Then open `http://127.0.0.1:8420` in your browser (use `--addr 127.0.0.1:9000` to pick a different port).

From the web board you can:

- View and manage work items in a **Kanban board** (`Backlog`, `In Progress`, `Review`, `Done`) or **Table view**.
- Sync issues and PRs from **GitHub** or **Gitea** (`Sync Forge` button or `ateam state sync`).
- Inspect active **OpenSpec changes**, **`Hold:` blockers**, and **`// shortcut:` items**.
- Open the top-right **User Profile & Settings** menu (auto-detected from `git config user.name` / `user.email`) to switch the repository's **Agent Discipline Mode** (`lite`, `full`, `ultra`, `off`) or configure `ATEAM_API_TOKEN`.
- Receive live issue and pull request webhooks at `/api/webhooks/github` (`X-Hub-Signature-256`) and `/api/webhooks/gitea` (`X-Gitea-Signature`) when `ATEAM_WEBHOOK_SECRET` is set.

Repository selection is shared by clients of one server. Writes include `X-ATeam-Repo` to reject stale selections (optional for single-repository clients). Registered paths survive restart; the active selection resets to the served checkout. Webhooks always target the served checkout. Forge settings support GitHub.com and configured Gitea hosts; custom GitHub API URLs are currently unsupported. The read-only model routing panel follows the selected repository; [routing states and limits](docs/model-review.md#dashboard-routing-display).

---

## Command Reference

### `ateam` CLI

| Command | What it does |
| --- | --- |
| `ateam init [--force] [--github\|--gitea]` | Create multi-agent instruction files, skills, hooks, MCP configs, ~500k context defaults, and optional GitHub/Gitea labels |
| `ateam doctor [--check]` | Verify multi-agent file parity, `<300` token MCP schema budget, and measured hook latency (50ms failure threshold today; <5ms target) |
| `ateam review setup --policy FILE` | Install approved model policy and role cards/pointers, refusing existing conflicts |
| `ateam review route [--developer-models IDS]` | Check the configured reviewer against every supplied implementation model; launch no agents |
| `ateam review check --pr N --record-comment ID [--exception-comment ID]` | Validate current review evidence and optional scoped owner exception against trusted baseline policy |
| `ateam state status` | Print the full `.ateam/state.json` snapshot (git, worktrees, holds, shortcuts, cards, handoff) |
| `ateam state pickup` | Reconcile live git/worktree state with the last handoff and report drift or active holds |
| `ateam state handoff --goal "..." [...]` | Record a 5-field checkpoint, clear the session writer lease, and optionally comment on the open PR |
| `ateam state holds` | Scan `openspec/` and markdown files for `Hold:` items |
| `ateam state debt` | Scan source files for `// shortcut: <limit>, <when to upgrade>` comments |
| `ateam state spec-queue` | List in-flight `openspec/changes/*` with task counts and caveats (`HALT`, `RED`, `BLOCKED`, `WONTDO`, `OPEN-Q`) |
| `ateam state spec-check` | Validate `openspec/` specs and changes and ensure `Hold:` lines live inside task items |
| `ateam state sync [--forge github\|gitea]` | Sync open issues and PRs from GitHub or Gitea into `.ateam/state.json` |
| `ateam state mode [lite\|full\|ultra\|off]` | Get or set the active discipline level |
| `ateam state herdr <open-pane\|on-event>` | Open `pickup`/`spec-queue` herdr popups or emit caveat/Hold toasts when running inside `herdr` (`HERDR_ENV=1`) |
| `ateam task [list\|<target>]` | List or run targets defined in `Taskfile.yml` |
| `ateam test-spec` | Check test attribution comments against exact OpenSpec requirement headings |
| `ateam mutate -pkg ... -test ... -file ... -mutant ...` | Run a non-destructive `go test -overlay` mutation check |
| `ateam bench [--selftest] [--rescore FILE] [--markdown] [--save]` | Run or rescore offline benchmark self-tests and mutation checks |
| `ateam bench <run\|compare\|rescore> ...` | Run, compare, or rescore paired benchmark experiments and A/A calibrations (see [benchmarks/README.md](benchmarks/README.md)) |
| `ateam hook --event <event> [--agent claude\|codex\|gemini\|agy]` | Run lifecycle hook (`session-start`, `before-prompt`, `pre-compact`, `turn-end`) |
| `ateam mcp` | Start the 2-tool stdio MCP server (`ateam_state`, `ateam_task`) |
| `ateam serve [--addr 127.0.0.1:8420]` | Start the local web board HTTP server |

### `Taskfile.yml` Targets

| Target | What it does |
| --- | --- |
| `task review:route -- ...` | Check configured model routing, forwarding optional CLI arguments |
| `task review:check -- --pr N --record-comment ID [...]` | Validate current PR review evidence and an optional owner exception |
| `task fmt` / `task fmt:check` | Format Go files / fail if any Go file is unformatted |
| `task test:spec` | Validate attached test-to-spec references in Go and Vitest tests |
| `task spec:check` | Validate `openspec/` specs and changes |
| `task spec:queue` | Show in-flight `openspec/changes/*` progress and holds |
| `task tidy:check` | Fail if `go.mod` or `go.sum` is not tidy |
| `task vet` / `task lint` | Run `go vet` / `revive` linter |
| `task test:unit` | Run Go unit and parity tests with `-race` |
| `task mutate:check -- ...` | Run `ateam mutate` non-destructive mutation check |
| `task bench:selftest` | Run offline benchmark self-tests |
| `task ui:check` / `task ui:test` | Run SvelteKit type checks / Vitest unit tests |
| `task build` | Build static UI + `./bin/ateam` binary |
| `task verify` | Run all verification checks on a clean working tree |

---

## Repository Layout

```text
├── AGENTS.md                  # Canonical instructions read by Codex, Gemini, and Antigravity
├── CLAUDE.md                  # 1-line import (@AGENTS.md) read by Claude Code
├── .agents/
│   ├── README.md              # Cross-agent mapping table and parity checklist
│   ├── protocol.md            # Shared workflow, state, and testing protocol
│   ├── rules-index.md         # Enforcement matrix paired with AGENTS.md
│   ├── hooks.json             # Antigravity (agy) native lifecycle hook configuration
│   ├── plugins/ateam/         # Antigravity (agy) workspace MCP plugin
│   ├── contracts/             # Canonical developer and reviewer role contracts
│   ├── herdr/                 # Optional herdr plugin manifest (herdr-plugin.toml)
│   └── skills/                # Canonical skills (pickup, handoff, preflight, ponytail)
├── .mcp.json                  # Claude Code project MCP registration
├── .claude/                   # Claude Code hook/compaction settings and thin skill adapters
├── .codex/                    # Codex CLI hook, MCP, and context window settings
├── .gemini/                   # Gemini CLI hook, MCP, compression settings, and command adapters
├── benchmarks/                # Benchmark experiment configs, bundles, and calibration reports
├── docs/                      # Adoption, hook-flow, MCP, and model-review guides
├── openspec/                  # Spec-driven requirements (specs/) and active/archived changes
├── cmd/ateam/                 # CLI entrypoint
├── internal/
│   ├── bench/                 # Offline self-tests, paired experiment runner, and native adapters
│   ├── bootstrap/             # ateam init templates and ateam doctor parity checks
│   ├── hook/                  # <5ms lifecycle hook handler and session writer lease
│   ├── mcpserver/             # 2-tool stdio MCP server (<300 schema tokens)
│   ├── mutate/                # Non-destructive go test -overlay mutation runner
│   ├── review/                # Stronger-model review routing and forge evidence checks
│   ├── state/                 # Git, OpenSpec, shortcut, and GitHub/Gitea state engine
│   ├── taskrunner/            # Taskfile.yml parser and runner
│   ├── testspec/              # Go and Vitest test-to-spec attribution checker
│   └── web/                   # HTTP server, REST API, webhook handlers, and embedded UI
└── ui/                        # Svelte 5 + SvelteKit source for the web board
```

## Origins

Ponytail supplied the decision ladder and discipline ideas adapted into Go here. SemEngine informed shared-state handoff and pickup; OpenSpec supplies the spec/change workflow vocabulary. The lab combines these ideas to explore harness configuration, without claiming that their effectiveness has already been measured.
