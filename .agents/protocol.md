# Shared Work Protocol (Claude, Codex, and Gemini)

Canonical protocol for `ateam` and repositories bootstrapped with `ateam init`. `AGENTS.md` and `CLAUDE.md` point here and inline the four workflow gates (**Clarify, Claim, Merge, Close**).

## Adoption scope

ATeam is an opinionated experimentation lab primarily grounded in Go development. Initialization uses the same templates for all project languages; it does not detect language support or create a Taskfile. Review and adapt these instructions before adoption. Go mutation checks require non-test .go source; verification targets belong to the adopting project. Shortcut scanning covers only .go, .ts, .js, .svelte, .sh, .yml, and .yaml files (excluding Go tests and .test.ts files). Other languages are not validated as complete workflows. Ponytail modes change injected guidance, not model settings or guaranteed compliance. Benchmark self-tests validate reference fixtures, not improved agent performance.

See [the adoption guide](../docs/adoption.md) for command dependencies, scanner limits, and verification setup. The Go/Svelte gates below describe this repository; adopting projects must define suitable equivalents.

## 1. Hybrid State Architecture (GitHub + Gitea + OpenSpec + Local Cache)

Shared truth lives in the repository's versioned code, `openspec/` specs and changes, git worktrees, and the repository's remote forge (**GitHub** or **Gitea**), backed by a fast gitignored local cache (`.ateam/state.json`) so CLI lifecycle hooks execute in `<10ms` without blocking on network calls:

- **What is wanted & decided (GitHub & Gitea):** Issues with labels `type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`. `ateam` auto-detects **GitHub** (`gh` CLI or REST API) and **Gitea** (`GITEA_URL`/`GITEA_TOKEN` REST API v1 or `tea` CLI) from the git remote or environment, falling back to `.ateam/state.json` when offline.
- **Who has claimed what:** A draft PR on an agent-prefixed branch (`claude/<topic>`, `codex/<topic>`, `gemini/<topic>`) in its own worktree (`git worktree list`), with `Closes #n` or `Addresses #n`.
- **Target state, task truth & holds (`openspec/`):**
  - `openspec/specs/<capability>/spec.md` holds current truth; `openspec/changes/<id>/` (`proposal.md`, `design.md`, `tasks.md`) holds in-flight changes.
  - `task spec:queue` (`ateam state spec-queue`) scans `openspec/changes/*/tasks.md` and reports task progress (`done/total`) plus open-task caveats (`WONTDO` for `- [~]` or `deliberate`/`not done`, `HALT`, `RED`, `BLOCKED` for `Hold:` unless the hold leads with `HALT`, `RED`, or `OPEN-Q`, and `OPEN-Q` for `still open` or `Hold: OPEN-Q`).
  - `task spec:check` (`ateam state spec-check`) validates OpenSpec structure (running `openspec validate --all --strict --no-interactive` when installed plus native Go checks) and fails if any `Hold:` appears below the first `## ` heading outside a task block; it warns, without failing, when a change's `proposal.md` exceeds 60 lines or its `design.md` exceeds 120.
- **Intentional shortcuts (`// shortcut:`):** Deferred complexity is marked inline in source code as `// shortcut: <the limit>, <when to upgrade>` and scanned live by `ateam state debt`. Never maintain a separate tracked `debt-ledger.md` file that can drift from code.
- **Session handoff & pre-compaction cache:** `ateam state handoff` (and the `ateam_state` MCP tool) snapshots live git branch/HEAD/dirty status, OpenSpec queue/holds, forge issues/PRs, and `// shortcut:` items alongside the 5 semantic handoff fields (`Goal`, `Addresses`, `Constraints`, `Done when`, `Checks to run`) into `.ateam/state.json` and formats the 5-field pickup block.

## 2. Work Lifecycle & Definition of Done

1. **Clarify:** Before mutating state on an underspecified request, inspect the tree first, then confirm:
   - **Deliverable** (answer/audit, issue, OpenSpec design, or code/UI change) and which role drafts it ([ADR 0003](../docs/adr/0003-roles-and-routes.md) catalog)
   - **Scope and non-goals** (in-bounds packages/files and what stays untouched)
   - **Definition of Done and verification** (`task verify` gates and observable checks)
   - **Stop boundary** (pause after design/plan vs. run through to verified completion). For non-trivial work the default is **pause for proposal approval**: the OpenSpec change's `proposal.md` stays one page (≤ 60 lines: why, what changes, scope/non-goals, Definition of Done; detail goes in `tasks.md` or `design.md`; `task spec:check` warns past 60) and carries an empty `Approved-by:` line, and its first task carries `Hold: OPEN-Q proposal awaits owner approval` from creation. The owner fills `Approved-by: <name> (<YYYY-MM-DD>)` and removes the hold in the same edit; an agent with this stop boundary stops at the hold and hands off.
   - **Track** (non-trivial work): `quick` or `design`. A change takes `design` when it adds a spec Requirement, touches more than one capability under `openspec/specs/`, adds exported surface reachable from an adopting repository, or touches a boot path, durability/ack behavior, or a gate file; otherwise it is `quick` and has no `design.md`. On `design`, the [design skill](skills/design/SKILL.md) drafts `design.md` in OpenSpec's `spec-driven` sections under the `design` rules in `openspec/config.yaml` (≤ 120 lines; `task spec:check` warns past it), and the one `Hold: OPEN-Q proposal awaits owner approval` covers `proposal.md` and `design.md`: one owner edit lifts both. Scenarios need no extra gate: `openspec validate --strict` already rejects a Requirement without one.
2. **Claim:** Create or switch to an isolated worktree on an agent-prefixed branch (`claude/…`, `codex/…`, `gemini/…`; in Enjoy desktop, use `move_to_worktree`), backed by a draft PR on GitHub or Gitea.
   - Inside a herdr pane (`HERDR_ENV=1`), create the worktree with `herdr worktree create --cwd "$PWD" --branch <agent>/<topic> --path "$PWD/.worktrees/<agent>-<topic>" --no-focus` so herdr groups it as a workspace. The explicit `--path` keeps the `.worktrees/` layout; `git worktree list` stays the source of truth.
3. **Execute (Ponytail Decision Ladder):**
   - Take the smallest complete change that solves the task.
   - Reuse existing code and stdlib before adding helpers or dependencies.
   - Add a focused test for every new branch, loop, parser, or boundary condition using an independent oracle.
   - Tag any deliberate limitation with `// shortcut: <the limit>, <when to upgrade>`.
4. **Verify (4-Gate Definition of Done):**
   - **Gate 1 (`task verify`):** `fmt:check`, `spec:check`, `test:spec`, `tidy:check`, `vet`, `lint` (`revive`), `test:unit` (`-race`), `ui:check`, `ui:test`, and clean git tree check.
   - **Gate 2 (`ateam doctor` / Parity):** All canonical skills, `.claude/`, `.codex/`, `.gemini/` adapters/hooks, `<24 KB` `AGENTS.md` cap, `<300` token MCP tool schema budget, and warm best-of-3 hook latency (`<10ms` target budget, `100ms` `--check` ceiling) pass.
   - **Gate 3 (`ateam bench --selftest`, `task mutate:check` & `task merge:check`):** Good/bad reference fixtures and mutation checks pass offline, and non-draft code PRs record `implemented-by:` and a not-weaker `reviewed-by:` (`task merge:check -- <pr>`). The reviewer posts a PR comment starting `reviewed-by: <agent> (<model>, <effort>)` and naming one PR commit as `Reviewed at <sha>`; any later commit that is not a clean merge of `main` or docs-only needs a new record (on Gitea, any later commit). Template: [reviewer contract § Record](contracts/reviewer.md#record).
   - **Gate 4 (Ponytail Exit Summary):** Each task conclusion and handoff ends with 1–2 lines naming what was skipped/not checked and any residual risk; acknowledgements and progress updates need no footer but still name blockers, failed or unavailable checks, and changed risks.
5. **Handoff or Land (Close):** At a session boundary or before context compaction, run `ateam state handoff` (or `/handoff`) to record the 5-field pickup prompt. When a PR lands an OpenSpec change, the archive move (`openspec/changes/<id>/` → `openspec/changes/archive/<date>-<id>/`) is a commit on the branch before the PR is marked ready, and the change's `Hold: OPEN-Q proposal awaits owner approval` must be gone before that move; merging with a completed unarchived change is a Close-gate miss, not a follow-up. Mark the PR ready and arm auto-merge (`gh pr merge --squash --auto`; on Gitea, `POST /repos/{owner}/{repo}/pulls/{n}/merge` with `{"do":"squash","merge_when_checks_succeed":true}`); the merge fires when the required check is green on an up-to-date branch, and arming it past a previously red required job needs the cause fixed or an owner waiver. Once the merge has landed, remove the worktree you claimed and delete its remote branch; `ateam state prune` does both for every clean, merged worktree on your agent prefix (`--dry-run` first lists what it would remove). The `main` run is informational: a red one after a green up-to-date PR run is a tracked issue, not a reason to keep the worktree. Review checkouts (`.worktrees/<agent>-review-pr-<n>`, detached; `ateam state prune` also removes them once the PR is closed) are removed when the review record comment is posted. Pickup labels every merged and PR-less worktree; a finished worktree older than a day is yours to explain or remove.

## 3. Mutation & Property-Based Testing (PBT) Discipline

1. **Start with the wrong behavior & independent oracle:**
   - Before writing a test, state in one sentence the correct behavior and one plausible wrong behavior the test must catch.
   - Expected values come from an independent oracle (requirement, grammar, or small test-owned reference model), never by recomputing with the production algorithm.
2. **Show that the test can fail (`task mutate:check` / `ateam mutate`):**
   - **Never run destructive git commands** (`git checkout --`, `git restore`, `git stash`, `git clean`, `git reset --hard`) to undo a test mutation; they destroy uncommitted/untracked work.
   - For Go source files (`!_test.go`), write the mutant to a copy outside the repository (`${TMPDIR:-/tmp}/mutant.go`) and run `task mutate:check -- -pkg ./path/to -test TestName -file path/to/file.go -mutant "${TMPDIR:-/tmp}/mutant.go" -expect-text 'expected failure'`. It uses `go test -overlay` so zero repository files are modified.
   - For non-Go files (scripts, fixtures), back up with `cp path/to/file "${TMPDIR:-/tmp}/file.bak" && shasum -a 256 path/to/file` before and verify the SHA-256 checksum matches after restoring.
   - Report the exact outcome: **`detection`** (baseline passed, mutant failed with the expected assertion, after-run passed), **`survivor`** (mutant compiled and was reached, but test passed), **`invalid`** (mutant failed to build or test never reached the mutated lines), or **`inconclusive`** (timeout, signal, or unrelated failure). Never report a `survivor` or `inconclusive` run as a `detection`.
3. **Generated checks (Fuzz targets & Property-Based Testing):**
   - Decide and record whether generated checks (`func FuzzX(f *testing.F)` or PBT) are needed whenever a change touches:
     - **Interacting input grammars or size/count limits**
     - **A transformation with a stated law** (round-trip, idempotence, normalization, partitioning)
     - **An order-dependent history** (operations, replacement/deletion, retry/replay, shutdown)
   - Either write a generated check or name the deterministic examples and explain why they cover the combinations and orderings ("existing tests pass" or a test count is not a reason).
   - Ensure generators/seeds reach the exact boundary, prove **assertion activation** (what makes the assertion actually execute rather than skipping an empty loop/guard), and compare histories against a test-owned reference model updated from the requirement.
   - Record in the PR: `Decision`, `Rule`, `Inputs`, `Run` (commit, command, seed, completed checks from `-v` output; seed replay vs. `-fuzz` exploration separate), `Mutation` (wrong change + outcome), and `Not covered` (unresolved survivors or deferred checks).


## Test-to-spec attribution

Run `ateam test-spec` (or `task test:spec`) to deterministically validate every Go `Test`, `Fuzz`, and executable `Example` declaration and every Vitest `it`/`test` declaration. Attach one or more line comments directly to the declaration, naming existing repository-relative specifications and exact requirement headings:

```go
// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestOpenSpecQueueAndCheck(t *testing.T) {
    // assertions
}
```

Multiple requirements use separate `// spec:` lines. Go subtests inherit the top-level declaration; each Vitest test needs its own immediately preceding comment, without inheritance from `describe`. Missing, malformed, dangling, and escaping references fail the gate. Keep references current when requirements move or are renamed. The checker excludes benchmark functions and fixture source inside string literals; generated test source is not exempt. Dependency, VCS, worktree, and build-output directories are excluded.

A test may instead (or also) carry `//aip:verifies <IDs>` (no space after `//`; the spaced form is a diagnostic), naming one or more space-separated scenario IDs. Each ID resolves when a line outside code fences in a canonical `openspec/specs/*/spec.md` or in-flight `openspec/changes/<change>/specs/*/spec.md` (never `archive/`) is `#### Scenario: <ID>` followed by a space or end of line; an empty or unknown ID fails.

Vitest scanning requires Node and TypeScript installed at ui/node_modules/typescript in the target repository. Supported filenames end in .test or .spec plus .ts, .tsx, .js, or .jsx; .mjs/.cjs and other frameworks are outside this gate. Direct Vitest imports (including aliases), namespace calls, globals, modifiers, and parameterized declarations are recognized; arbitrary wrapper modules and runtime expansion counts are not covered. A blank line detaches a citation; block comments do not count as citation lines. Directory symlinks are not followed and symlink test files are rejected. The checker includes untracked supported tests and all Go build tags. It skips .git, .worktrees, node_modules, vendor, .svelte-kit, .enjoy-logs, ui/build, internal/web/dist, coverage, bin, benchmarks/runs, and .ateam. Other testdata files are not exempt.

Attribution validates syntax and reference existence, not whether assertions prove the requirement or whether all specifications have tests. Review must assess those claims separately. The current supported languages are Go and Vitest JavaScript/TypeScript; adopting projects must provide appropriate attribution enforcement for other frameworks and languages.
## 4. Configured Model Routing & Independent Review

**The live gate.** `task merge:check` decides merge (Gate 3): an independent reviewer run that wrote none of the deliverable, not weaker than every implementer run by model tier, then effort band (ADR 0002), a different agent only when the policy sets `cross_provider`, and a record naming a PR commit with only clean `main` merges or docs-only commits after it (on Gitea, no later commit at all). An equal tier and band passes, so the live gate has no ceiling and no exception. Reasoning effort is ranked separately from model strength, and a different provider never proves strength. A policy declaring only tiers and bands, as this repository's does, configures no routes: `ateam review route` exits non-zero by design, and no route needs confirming.

Follow the [developer](contracts/developer.md) and [reviewer](contracts/reviewer.md) contracts. The reviewer remains read-only and the author fixes findings. Review the complete diff and affected callers; instructions, contracts, policy, executable Markdown and rename paths require substantive review. Record all runs, findings, verdict and actual/reported/unavailable verification against the reviewed commit. A later commit the reviewer contract's [Record](contracts/reviewer.md#record) section does not exempt needs a new record. Expect one red `Merge check` between marking the PR ready and the record: after an approving record the reviewer adds its `reviewed-by:` line to the description and reruns the failed job (review-pr § 5), and after a rejecting one the check stays red; do not update or rebase the branch to clear it.

**Configured route path (optional; dormant here).** Only when the policy also declares models and routes: validate the route before implementation with `ateam review route`, and after any escalation revalidate against every actual implementation model. `ateam review check` then requires a reviewer that outranks every implementer, binds evidence to the repository, base/head commits and governing policy digest (any new head or policy revision invalidates it), and at the model ceiling blocks approval until an authorized owner explicitly grants a head/reviewer/policy-bound exception with a reason. It waives superiority only, never independence, provider constraints, current evidence, required checks or the verdict. `ateam review check` reads trusted baseline policy and authenticated GitHub/Gitea comment authors; candidate policy cannot authorize its own waiver. A configured route and a forge login remain attestations, not proof of model runtime identity or a separate human owner. See [model review](../docs/model-review.md) for the format and limitations.

**Gate files (#140).** The review policy, the merge-check and gate-files scripts, the workflows (`.github/workflows/` and `.gitea/workflows/`), and `internal/review/` decide whether a pull request may merge, and CI runs them from the pull request's own tree. So a pull request touching any of them fails the `Gate files` check until the owner adds the label `gate-change-approved`. That check runs from the base branch on `pull_request_target` and reads the pull request through the API only, so the pull request cannot edit it; the `Merge check` job likewise checks out and builds from the base commit, so the script, policy and `ateam` binary judging a pull request never come from it. Only the owner applies the label. An agent blocked by it hands off; it never applies the label itself. Under a shared forge login the label is an attestation, not proof of who applied it, and a token with admin rights can still edit the ruleset itself: give unattended agents a token without it.

### Unattended and fan-out runs

This subsection explains how the state model behaves; it adds no gate condition. Skipping tool permissions changes nothing about the gate: hooks still run, the ruleset still requires `Required` and `Gate files` with no bypass actors, and auto-merge still waits for both. An unattended run finishes by satisfying the gate or by handing off.

- **Review.** The gate needs a `reviewed-by:` line and a record at the PR head from a run not weaker than every implementer (ADR 0002). Name the model as `.agents/review-policy.json` lists it (for example `opus`) and the effort: a placeholder effort (`unknown`, `n/a`) is a missing effort, and a model the policy does not map passes only against another unmapped run (#125). The gate cannot tell a fresh-context review from one written in the implementing context under a shared login (ADR 0001). A fresh context (a reviewer subagent, a separate session, or a configured reviewer flow) is what makes the record worth anything; the per-model finding rate is the long-run check.
- **Gate files.** A run whose change touches a gate file waits for the owner's label (above). It hands off; it never applies the label.
- **Fan-out.** Give each run its own worktree and branch. Pickup from `main` shows the handoff recorded inside each worktree (#136). The handoff slot is keyed `agent@branch`, so on a shared branch a deliberate handoff over another session's fresh one is refused (CLI exit 2, MCP error, HTTP 409) unless forced (#135). Do not force over another session's handoff; record yours from your own worktree. The handoff command also posts it as a comment on the branch's open PR.
- **Lease.** The writer lease is per worktree, so runs in separate worktrees never see each other's. "Another writer may be active" means a second live session shares this worktree and its handoff slot: move to your own worktree rather than writing here. Never clear another session's lease.
- **Stop boundaries.** Unchanged. Destructive actions and scope changes end the run with a handoff.

For hosts launching unattended agents, the token the agent inherits is what stands between it and the ruleset: give it one without admin rights, as the gate-files paragraph says.
