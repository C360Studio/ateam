# ATeam (`ateam`)

**A daily-driver workflow and harness for Go development across coding agents.**

`ateam` is a single static Go binary, low-overhead MCP server, and local web board built for day-to-day Go work with **Claude Code**, **Codex CLI**, **Gemini CLI**, and **Antigravity (`agy`)**. Write your instructions, skills, and workflow rules once; run one agent or switch between them across git worktrees without losing state, context, or test discipline.

In daily use, `ateam` gives a Go repository:

- **One shared agent setup:** Maintain a single `AGENTS.md` and canonical skills in `.agents/skills/`. `ateam init` and `ateam doctor` keep thin adapters, `<10ms` lifecycle hooks, `<300`-token MCP registration, and ~500k context compaction defaults in sync across all four agent CLIs.
- **Session continuity across worktrees and agents:** `ateam state pickup` and `ateam state handoff` reconcile live git worktrees, writer leases, **GitHub** or **Gitea** issues and draft PRs, OpenSpec task queues (`Hold:`), and inline debt markers (`// shortcut:`).
- **Go-first engineering and test guardrails:** Built-in Ponytail scope discipline (`on`, `off`), non-destructive `go test -overlay` mutation checks (`ateam mutate`), test-to-spec attribution (`ateam test-spec`), and commit-bound cross-agent or stronger-model PR review gates (`ateam review`).
- **Built-in harness benchmarking (`ateam bench`):** When you change shared instructions, skills, hooks, or models, `ateam` doubles as an experimentation lab to test whether those changes alter outcomes on reproducible Go tasks.

For day-to-day developer workflow and what to expect in an initialized repository, read [the daily developer guide](docs/daily-use.md). To bring `ateam` into a repository that already has its own instructions, hooks, and Taskfile, follow [the brownfield guide](docs/brownfield.md). For the event-by-event lifecycle, state changes, and design rationale, read [how the harness runs](docs/hook-flow.md). For optional stronger-model review, setup, and scoped owner exceptions, read [model routing and independent review](docs/model-review.md). For paired benchmark runs, native API adapters, and versioned instruction bundles, read [benchmark experiments](benchmarks/README.md).

## Opinions and Boundaries

ATeam adapts Ponytail’s decision ladder and supporting mechanisms into a daily Go workflow and experimentation lab: question whether code needs to exist, reuse what is already available, prefer the standard library and installed dependencies, and finish the smallest complete change. Its workflow also favors isolated worktrees, issue-linked draft PRs, OpenSpec task lists, explicit handoffs, and tests demonstrated to catch wrong behavior. These are deliberate choices that adopters should review.

Ponytail modes (`on`, `off`) change injected guidance. They do not guarantee agent compliance or tune model/provider settings; `off` does not remove written repository instructions or disable all harness bookkeeping. Likewise, `ateam bench` supports versioned instruction comparisons with shared runner/model settings, independent grading, retained failures, and offline replay, while native skill/hook activation evidence and same-model cross-runtime comparisons remain [roadmap work](https://github.com/C360Studio/ateam/issues/16); existing fixtures and small API calibrations do **not** establish that ATeam improves agent performance.

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
- **Shared truth in Git, OpenSpec, and GitHub/Gitea—cached locally for speed:** Issues and pull requests live on **GitHub** or **Gitea**. Specs and task checklists live in `openspec/`. Active work lives in git worktrees. `ateam` caches a local snapshot in `.ateam/state.json` (gitignored) so lifecycle hooks can read repository state without making network calls on every prompt. Sub-10ms hooks are a design target, not a guarantee for every repository size.
- **One static Go binary (`ateam`):** Lifecycle hooks (`ateam hook`), the MCP server (`ateam mcp`), state inspection and optional `herdr` integration (`ateam state`), model review checks (`ateam review`), test-to-spec attribution (`ateam test-spec`), mutation checks (`ateam mutate`), offline self-tests and paired experiments (`ateam bench`), and the web board (`ateam serve`) share one compiled binary. Hooks avoid external subprocesses, while other commands require tools such as Git, Go, or Task; see the [dependency table](docs/adoption.md#command-dependencies).
- **Two MCP tools under 300 tokens:** Instead of one MCP tool per action, `ateam mcp` exposes only `ateam_state` (pickup, handoff, debt, mode, OpenSpec checks, cached board) and `ateam_task` (list and run `Taskfile.yml` targets). Tool definitions are approximately 190 tokens by the bytes/4 estimate. See the [MCP setup and tool guide](docs/mcp.md) for registration, prompts/resources, side effects, and connection checks.
- **Source-code comments instead of separate debt files:** Intentional shortcuts are marked right where they happen (`// shortcut: <what is limited>, <when to upgrade>`) and scanned from source by `ateam state debt`. Task blockers are marked inline in `openspec/changes/<id>/tasks.md` with `Hold: <reason>`.
- **Non-destructive mutation testing (`ateam mutate`):** To verify that a unit test actually catches a bug, `ateam mutate` runs `go test -overlay` against a temporary modified file outside your repository—never touching your working tree or running destructive `git checkout` / `git restore` commands.

---

## Prerequisites and Installation

To build `ateam` from source, you need:

- **Go 1.26.4+** (the version declared in `go.mod`)
- **Node.js 20+ and npm** (used to build the static web UI embedded into the Go binary)
- **[Task](https://taskfile.dev) (`task`)** (optional, for running `Taskfile.yml` commands)
- **`gh`** (GitHub CLI) or **`tea` / `GITEA_TOKEN`** (optional, for syncing issues and PR comments with GitHub or Gitea). To give each role its own scoped token, set `ATEAM_TOKEN_READ` (board, sync, pickup, review evidence) and `ATEAM_TOKEN_WRITE` (PR comments, label bootstrap); either one alone serves both, and with neither set the tokens above are used. `ateam doctor` warns while one token serves both roles.

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

Expose `./bin/ateam` on your `PATH`—for example, symlink it into `~/.local/bin/ateam` (`ln -sf "$PWD/bin/ateam" ~/.local/bin/ateam`) so `brew doctor` is unaffected and desktop hosts (such as Enjoy) pick up rebuilds in place—so Claude Code, Codex CLI, Gemini CLI, and Antigravity (`agy`) can invoke `ateam` from their hook and MCP configurations. Check which build you are running with `ateam version`, which prints the release and the commit it was built from. `task build` takes the release from `git describe`: a tag checkout prints `ateam v0.3.3 (<commit>)`, a commit after a tag prints `ateam v0.3.2-19-gfd9d205 (fd9d205)`, and a dirty tree appends `-dirty`. `go install github.com/c360studio/ateam/cmd/ateam@<tag>` prints `ateam <tag> (module)`, and `go run` prints `dev`. GitHub releases carry no prebuilt binaries yet.

---

## Getting Started in 8 Steps

### 1. Initialize a repository (`ateam init`)

Run `ateam init` inside your project repository (or pass `--dir /path/to/repo`):

```bash
ateam init
```

This creates missing harness files. Existing client configuration is always preserved; `--force` replaces generated documents and adapters, first copying a hand-written `AGENTS.md`, `CLAUDE.md`, `.agents/protocol.md` or `.agents/rules-index.md` to `<name>.ateam-backup-<timestamp>` and listing it under `backed_up`. `.gitignore` entries are appended separately. In a repository that already has its own instructions, run `init` without `--force` and follow [the brownfield guide](docs/brownfield.md). It installs the same opinionated templates for every language and does **not** create a `Taskfile.yml` or install project tools. Review the generated instructions and supply your own verification tasks before asking an agent to follow them:

- `AGENTS.md` and `CLAUDE.md` (`@AGENTS.md`)
- `.agents/protocol.md`, `.agents/rules-index.md`, `.agents/README.md`, and `.agents/skills/{pickup,handoff,preflight,ponytail,review-pr}/SKILL.md`
- `.agents/contracts/developer.md` and `.agents/contracts/reviewer.md`
- `.agents/hooks.json` and `.agents/plugins/ateam/{plugin.json,mcp_config.json}` (Antigravity lifecycle hooks and workspace MCP plugin)
- `.claude/settings.json` (hooks + `"autoCompactWindow": 500000`), `.claude/skills/*/SKILL.md`, and `.mcp.json` (MCP registration)
- `.codex/hooks.json` and `.codex/config.toml` (MCP registration + `model_context_window = 500000` / `model_auto_compact_token_limit = 450000`)
- `.gemini/settings.json` (hooks, MCP registration, `context.fileName: "AGENTS.md"`, and `contextPercentageThreshold: 0.5`) and `.gemini/commands/*.toml`
- `openspec/config.yaml`, `openspec/specs/`, and `openspec/changes/`
- `.gitignore` entries for `.ateam/`, `.worktrees/`, `.enjoy-logs/`, and `benchmarks/runs/`

If your repository is hosted on GitHub or Gitea and you want to create the standard issue labels (`type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`), pass `--github` or `--gitea`:

```bash
ateam init --github
# or
ateam init --gitea
```

### 2. Check your setup (`ateam doctor`)

Check local instruction/configuration parity, estimated MCP schema size, and measured hook latency (best of three warm runs; the 10ms target only warns, the 100ms ceiling fails `--check`). This does not prove that each installed agent CLI consumes the configuration correctly or that your project verification tasks exist:

```bash
ateam doctor --check
```

Example output (abridged):

```json
{
  "hook_latency_budget_us": 10000,
  "hook_latency_us": 4371,
  "mcp_schema_bytes": 759,
  "mcp_schema_tokens": 190,
  "passed": true,
  "violations": null,
  "warnings": [
    "forge reads and writes share one token; set ATEAM_TOKEN_READ and ATEAM_TOKEN_WRITE to separate tokens scoped to each role"
  ]
}
```

That token warning is intended for solo setups and never fails the check.

Model review is opt-in: `init` installs the role contracts but does not choose models or install a policy. Follow the [adoption steps](docs/adoption.md#optional-model-review) to approve model relationships, configure routes, and establish a trusted baseline before checking review evidence.

### 3. Start or resume work (`ateam state pickup`)

When starting a session—or picking up work from an earlier session or another agent—run:

```bash
ateam state pickup
```

Or use the `/pickup` skill inside Claude Code, Codex CLI, Gemini CLI, or Antigravity (`agy`). `pickup` reconciles your live git branch, `HEAD` SHA, uncommitted changes, active git worktrees, open `Hold:` items, `// shortcut:` comments, and any commits made since the last recorded handoff. Each worktree is listed with its PR status (`open PR #185`, `merged via #57`, `no PR`, and so on) and, indented beneath it, the handoffs recorded inside that worktree (`handoff: <goal> (<agent>@<branch>, <age>[, stale])`), so a session in `main` sees work parked in other worktrees.

### 4. Claim a worktree and draft PR (`Claim` gate)

Work in a dedicated worktree on an agent-prefixed branch, backed by a draft PR that says what it closes, and for non-trivial work an OpenSpec change:

```bash
git worktree add .worktrees/claude-my-topic -b claude/my-topic
cd .worktrees/claude-my-topic
gh pr create --draft --title "docs: my topic" --body "Closes #12"
```

Use `codex/<topic>` or `gemini/<topic>` for the other agents; on Gitea open the draft PR through its web UI, `tea`, or API. `ateam state pickup` then labels the worktree with its PR, and it is yours to remove once the PR lands.

### 5. Write Go code, track blockers, and verify tests

While working on a change:

- **Track multi-step specs and blockers in `openspec/`:** Put change proposals and task checklists in `openspec/changes/<change-id>/tasks.md`. If a task is blocked, add `Hold: <reason>` indented under that task item:

  ```markdown
  ## Tasks

  - [ ] 1.1 Wire webhook signature verification
    Hold: waiting on secret rotation in staging
  ```

  A hold that starts with `HALT`, `RED`, or `OPEN-Q` (for example `Hold: OPEN-Q proposal awaits owner approval`) shows under that marker; any other hold shows as `BLOCKED`.

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

### 6. Hand off work at the end of a session (`ateam state handoff`)

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

### 7. Prove it and get it reviewed (`Merge` gate)

In this repository the gates are `task verify`, `ateam doctor --check`, and `ateam bench --selftest`; an adopting project supplies its own verification entry point. A non-draft code PR (any file outside `.md` and `openspec/**/*.{md,yml,yaml,json}`) must carry `implemented-by:` and `reviewed-by:` lines in its description, each naming an agent with its model and effort, for example `claude (opus, high)`. The strength gate then runs `ateam review compare`: the reviewer may not be weaker than any implementer by model tier, then effort band, as ranked in `.agents/review-policy.json`; with `"cross_provider": true` (set in this repository, optional for adopters) it must also be a different agent. The reviewer, in a fresh context, posts a PR comment in the [reviewer contract](.agents/contracts/reviewer.md#record) grammar (`reviewed-by: <agent> (<model>, <effort>)` and `Reviewed at <sha>`); the `review-pr` skill (`/review-pr`) does this. A later commit that is not a clean merge of `main` or docs-only needs a new record. Check it all before arming auto-merge:

```bash
task merge:check -- 185
```

A PR that touches a gate file (`.agents/review-policy.json`, the merge-check scripts, the workflows, `internal/review/`) also fails the `Gate files` check until the owner applies the `gate-change-approved` label; an agent blocked by it hands off. Details: [model review](docs/model-review.md) and [the merge gate on GitHub and Gitea](docs/adoption.md#the-merge-gate-on-github-and-gitea).

### 8. Archive, merge, and clean up (`Close` gate)

Archive first: move the completed change in a commit on the branch, so the merged head has no completed unarchived change (`ateam doctor` warns about one). Then mark the PR ready and arm auto-merge, which fires when the required check is green on an up-to-date branch:

```bash
git mv openspec/changes/my-change openspec/changes/archive/$(date +%F)-my-change
git commit -m "chore(openspec): archive my-change"
gh pr ready
gh pr merge --squash --auto
```

On Gitea, arm it with `POST /repos/{owner}/{repo}/pulls/{n}/merge` and `{"do":"squash","merge_when_checks_succeed":true}`. After the merge lands, the agent that claimed the branch removes its worktree and deletes the remote branch with `ateam state prune` (`--dry-run` first lists what it would remove), or by hand with `git worktree remove <path>` and `git push origin --delete <branch>`; never one that is dirty, unpushed, or not yours.

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
- Open the top-right **User Profile & Settings** menu (auto-detected from `git config user.name` / `user.email`) to switch the repository's **Agent Discipline Mode** (`on`, `off`) or configure `ATEAM_API_TOKEN`.
- Receive live issue and pull request webhooks at `/api/webhooks/github` (`X-Hub-Signature-256`) and `/api/webhooks/gitea` (`X-Gitea-Signature`) when `ATEAM_WEBHOOK_SECRET` is set.

Repository selection is shared by clients of one server. Writes include `X-ATeam-Repo` to reject stale selections (optional for single-repository clients). Registered paths survive restart; the active selection resets to the served checkout. Webhooks always target the served checkout. Forge settings support GitHub.com and configured Gitea hosts; custom GitHub API URLs are currently unsupported. The read-only model routing panel follows the selected repository; [routing states and limits](docs/model-review.md#dashboard-routing-display).

---

## Command Reference

### `ateam` CLI

| Command | What it does |
| --- | --- |
| `ateam version` | Print the release and build commit (`ateam <git describe> (<commit>)`) |
| `ateam init [--force] [--github\|--gitea]` | Create missing multi-agent instruction files, skills, hooks, MCP configs, ~500k context defaults, and optional GitHub/Gitea labels; `--force` backs up replaced documents first |
| `ateam doctor [--check]` | Verify multi-agent file parity, `<300` token MCP schema budget, and measured hook latency (100ms `--check` ceiling; 10ms target) |
| `ateam review setup --policy FILE` | Install approved model policy and role cards/pointers, refusing existing conflicts |
| `ateam review route [--developer-models IDS]` | Check the configured reviewer against every supplied implementation model; launch no agents |
| `ateam review check --pr N --record-comment ID [--exception-comment ID]` | Validate current review evidence and optional scoped owner exception against trusted baseline policy |
| `ateam review compare --implementer A:M:E... (--reviewer A:M:E \| --reviewer-record FILE)` | Exit 0 when the reviewer is not weaker than every implementer by model tier then effort band, 1 on a finding, 2 when undecidable; `--reviewer-record` reads an AIP review record (`review_record: "aip"`) |
| `ateam review gate [PR]` | The merge check: up-to-date rule, `implemented-by:`/`reviewed-by:` lines and record, and open `class:flake` issues; exit 0 pass, 1 fail, 2 unavailable. `task merge:check` and CI call it |
| `ateam state status` | Print the full `.ateam/state.json` snapshot (git, worktrees, holds, shortcuts, cards, handoff) |
| `ateam state pickup` | Reconcile live git/worktree state with the last handoff (including handoffs recorded inside other worktrees) and report drift or active holds |
| `ateam state read-handoff <agent>@<branch>` | Print one slot's five fields from this checkout's or a linked worktree's cache, under a `STALE — <reasons>` banner when stale; writes nothing, leaves the writer lease alone, and calls no forge. A missing slot exits 1 and lists the slots that exist |
| `ateam state prune [--dry-run] [--agent <name>]` | Refresh like pickup, then remove each clean linked worktree on your agent prefix labelled `merged via #n` or `merged (ancestor of main)`, and each `<agent>-review-pr-<n>` checkout of a closed PR, with its local branch and its remote branch while that still points at the local head. Every other worktree is listed with the reason it was kept; pickup marks candidates `, prune candidate` |
| `ateam state handoff --goal "..." [...]` | Record a 5-field checkpoint, clear the session writer lease, and optionally comment on the open PR |
| `ateam state holds` | Scan `openspec/` and markdown files for `Hold:` items |
| `ateam state debt` | Scan source files for `// shortcut: <limit>, <when to upgrade>` comments |
| `ateam state spec-queue` | List in-flight `openspec/changes/*` with task counts and caveats (`HALT`, `RED`, `BLOCKED`, `WONTDO`, `OPEN-Q`) |
| `ateam state spec-check` | Validate `openspec/` specs and changes and ensure `Hold:` lines live inside task items; strict validation runs the repo-pinned `node_modules/.bin/openspec`, else `openspec` on `PATH`, and is skipped with a notice when neither exists |
| `ateam state sync [--forge github\|gitea]` | Sync open issues and PRs from GitHub or Gitea into `.ateam/state.json` |
| `ateam state mode [on\|off]` | Get or set the Ponytail discipline mode |
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
| `task merge:check -- <pr>` | Run the merge gate (`ateam review gate`) against a PR |
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
│   └── skills/                # Canonical skills (pickup, handoff, preflight, ponytail, review-pr)
├── .mcp.json                  # Claude Code project MCP registration
├── .claude/                   # Claude Code hook/compaction settings and thin skill adapters
├── .codex/                    # Codex CLI hook, MCP, and context window settings
├── .gemini/                   # Gemini CLI hook, MCP, compression settings, and command adapters
├── benchmarks/                # Benchmark experiment configs, bundles, and calibration reports
├── docs/                      # Adoption, brownfield, hook-flow, MCP, and model-review guides
├── openspec/                  # Spec-driven requirements (specs/) and active/archived changes
├── cmd/ateam/                 # CLI entrypoint
├── internal/
│   ├── bench/                 # Offline self-tests, paired experiment runner, and native adapters
│   ├── bootstrap/             # ateam init templates and ateam doctor parity checks
│   ├── hook/                  # <10ms lifecycle hook handler and session writer lease
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
