# Adopting ATeam in a repository

ATeam is a Go-first lab for testing agent harness configuration. Its portable coordination mechanisms can be useful in other projects, but it does not provide a validated end-to-end workflow for every language. Choose its workflow intentionally: the generated instructions favor Ponytail’s smallest-complete-change discipline, OpenSpec, worktrees, issue-linked draft PRs, and explicit handoffs.

See [how the harness runs](hook-flow.md) for the hook lifecycle and which workflow rules rely on agent cooperation.

## Feature and language boundaries

| Feature | Current behavior |
| --- | --- |
| `ateam init` | Writes the same templates regardless of language. Does not inspect manifests to select a toolchain, create a Taskfile, install tools, or migrate existing configuration. Existing files are preserved unless `--force` is used; missing files are added and ignore entries are appended. |
| Hooks, modes, pickup/handoff, forge state, board | Operate on repository/session metadata rather than a language compiler. This is architectural portability, not evidence of complete support for every project or agent version. |
| `ateam test-spec` | Validates attached `// spec:` references on Go Test/Fuzz/executable Example and Vitest declarations against exact OpenSpec requirement headings. This is syntactic attribution, not semantic coverage. Other test frameworks/languages need project-owned enforcement; initialization does not create a Taskfile gate. See [the protocol](../.agents/protocol.md#test-to-spec-attribution). |
| `ateam review` | Optional project-approved model comparisons and independent review evidence. Requires explicit policy; does not launch models, enforce a runtime sandbox, or install branch protection. See [model review](model-review.md). |
| `ateam mutate` | Requires a non-test `.go` target, an external mutant copy, and a working Go test environment. Rejects other source extensions before starting tests. There is no Python, Rust, or JavaScript mutation runner. |
| `ateam state debt` | Scans only `.go`, `.ts`, `.js`, `.svelte`, `.sh`, `.yml`, `.yaml`. Skips `_test.go` and `.test.ts`, plus generated/dependency directories such as `vendor`, `node_modules`, `dist`, and `build`. Python, Rust, Java, JSX, and TSX files are skipped, not diagnosed as unsupported. |
| Shortcut syntax | A line must begin with optional whitespace followed by `//`, `#`, or `--`, then `shortcut:`. The first comma separates the limit from its revisit trigger. This is a text scan, not a language parser; trailing code comments and HTML comments are not recognized. |
| `ateam task` / `ateam_task` | Requires a project-owned `Taskfile.yml` and the Task executable to run targets. Target discovery handles two-space-indented top-level tasks, not the full YAML/includes feature set. Missing Taskfiles or unknown targets produce errors. |
| `ateam doctor` | Checks local file/configuration parity, schema budget, and measured hook latency. Does not validate your language toolchain, task coverage, or actual live-agent integration. The current latency failure threshold is 50ms; <5ms is the design target. |
| `ateam bench` | Runs built-in offline self-tests, with or without `--selftest`. Go fixtures use behavioral tests; the Svelte fixture uses source-text checks; continuity fixtures exercise harness mechanisms. This does not benchmark your project or compare agents/configurations on real work. |

Sources: [scanner](../internal/state/shortcut.go), [initialization](../internal/bootstrap/bootstrap.go), [task runner](../internal/taskrunner/taskrunner.go), [mutation runner](../internal/mutate/mutate.go), [CLI](../cmd/ateam/main.go), and [benchmark fixtures](../internal/bench/bench.go).

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
| Review evidence check | Git, a clean checkout matching the live PR head and containing the current default-branch base, plus network access to the forge API. GitHub accepts `GH_TOKEN`, then `GITHUB_TOKEN`, with optional `gh auth token --hostname github.com` fallback only for the default GitHub API. Gitea accepts `GITEA_TOKEN`, then `ATEAM_GITEA_TOKEN`; it does not use `tea` for review evidence. Public readable evidence may work without authentication. |
| Serve the board | The compiled binary with embedded UI assets; Node is a build dependency, not a requirement to serve those assets. |

## Adoption steps

1. Build/install ATeam and make the executable available to your agents. Building ATeam is separate from building the project you want agents to edit.
2. Review your existing agent instructions and settings. Run `ateam init` in the intended repository; review its diff before using `--force`, which overwrites generated file paths.
3. Adapt the instructions to your language and actual delivery policy. Generated references to Go tooling, mutation checks, and verification targets are not automatic implementations of those checks. Keep canonical instructions and adapter/rule-index parity consistent.
4. Create a `Taskfile.yml` exposing your project's checks. For example, a Python project's `verify` task could run its existing lint/type/test commands. Do not copy ATeam's own Taskfile unchanged: it expects this repository's `cmd/`, `internal/`, `go.mod`, and Svelte `ui/` layout.
5. Run `ateam doctor --check`, inspect `ateam task list`, and execute the project checks. Separately verify that your installed agent versions actually load the intended instructions, hooks, and MCP settings. A parity pass is not an integration certification.
6. Use pickup and handoff for continuity. Handoff can post to the open PR; use `--no-post` for a local-only checkpoint.

For non-Go mutation evidence, the shared protocol describes a manual backup/checksum approach. That is a workflow responsibility, not a built-in equivalent of `ateam mutate` for another language.

## Optional model review

Initialization installs canonical developer/reviewer contracts but leaves model policy unconfigured. To adopt the stronger-reviewer gate:

1. Have the owner approve exact model identities, explicit stronger-than relationships, reviewer/owner forge accounts, and required checks. Provider separation is optional; it never substitutes for a stronger reviewer. At the model ceiling, approval requires an explicit scoped owner exception.
2. Follow [model-review setup](model-review.md#configure-an-approved-policy), run `ateam review setup --policy approved-policy.json`, and inspect the generated policy, cards, and pointers. Select the specified model in each native client yourself.
3. Land the initial configuration in a separate owner-authorized PR to the repository default branch. The checker reads policy from that trusted baseline; the adoption PR cannot authorize itself with candidate policy.
4. For subsequent work, run `ateam review route`, including every actual implementation model after escalation. Have an independent reviewer record current verification and post fresh evidence, then run `ateam review check` on the clean current PR head. See [review evidence and exceptions](model-review.md#review-and-record-evidence).

The commands validate eligibility and supplied evidence; they do not automatically enforce a forge merge restriction. Adopting projects must choose where to require the check in their delivery workflow. See [upgrade and troubleshooting](model-review.md#upgrade-and-troubleshooting) for existing installations and stale provider pointers.

## Ponytail and measurement

The decision ladder is guidance for choosing the smallest complete solution. Its Go implementation injects compact prompts and records a repository discipline mode. `lite`, `full`, and `ultra` vary the wording; `off` suppresses ladder injection but does not erase repository rules or stop all state bookkeeping. These modes do not select a model, change its reasoning budget, or enforce obedience.

The benchmark self-tests establish whether the supplied fixtures and checks behave as expected. They do not establish causal effects on correctness, code size, latency, cost, or handoff quality in live agent work. Those comparisons need a separate experimental design and remain future lab work. `bench --rescore` currently reads an existing report and formats it; it does not rerun candidates or apply a new evaluator.
