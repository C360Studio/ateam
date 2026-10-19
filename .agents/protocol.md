# Shared Work Protocol (Claude, Codex, and Gemini)

Canonical protocol for `ateam` and repositories bootstrapped with `ateam init`. `AGENTS.md` and `CLAUDE.md` point here and inline the four workflow gates (**Clarify, Claim, Merge, Close**).

## Adoption scope

ATeam is an opinionated experimentation lab primarily grounded in Go development. Initialization uses the same templates for all project languages; it does not detect language support or create a Taskfile. Review and adapt these instructions before adoption. Go mutation checks require non-test .go source; verification targets belong to the adopting project. Shortcut scanning covers only .go, .ts, .js, .svelte, .sh, .yml, and .yaml files (excluding Go tests and .test.ts files). Other languages are not validated as complete workflows. Ponytail modes change injected guidance, not model settings or guaranteed compliance. Benchmark self-tests validate reference fixtures, not improved agent performance.

See [the adoption guide](../docs/adoption.md) for command dependencies, scanner limits, and verification setup. The Go/Svelte gates below describe this repository; adopting projects must define suitable equivalents.

## 1. Hybrid State Architecture (GitHub + Gitea + OpenSpec + Local Cache)

Shared truth lives in the repository's versioned code, `openspec/` specs and changes, git worktrees, and the repository's remote forge (**GitHub** or **Gitea**), backed by a fast gitignored local cache (`.ateam/state.json`) so CLI lifecycle hooks execute in `<5ms` without blocking on network calls:

- **What is wanted & decided (GitHub & Gitea):** Issues with labels `type:epic`, `status:needs-decision`, `status:blocked`, `class:flake`. `ateam` auto-detects **GitHub** (`gh` CLI or REST API) and **Gitea** (`GITEA_URL`/`GITEA_TOKEN` REST API v1 or `tea` CLI) from the git remote or environment, falling back to `.ateam/state.json` when offline.
- **Who has claimed what:** A draft PR on an agent-prefixed branch (`claude/<topic>`, `codex/<topic>`, `gemini/<topic>`) in its own worktree (`git worktree list`), with `Closes #n` or `Addresses #n`.
- **Target state, task truth & holds (`openspec/`):**
  - `openspec/specs/<capability>/spec.md` holds current truth; `openspec/changes/<id>/` (`proposal.md`, `design.md`, `tasks.md`) holds in-flight changes.
  - `task spec:queue` (`ateam state spec-queue`) scans `openspec/changes/*/tasks.md` and reports task progress (`done/total`) plus open-task caveats (`WONTDO` for `- [~]` or `deliberate`/`not done`, `HALT`, `RED`, `BLOCKED` for `Hold:`, and `OPEN-Q` for `still open`).
  - `task spec:check` (`ateam state spec-check`) validates OpenSpec structure (running `openspec validate --all --strict --no-interactive` when installed plus native Go checks) and fails if any `Hold:` appears below the first `## ` heading outside a task block.
- **Intentional shortcuts (`// shortcut:`):** Deferred complexity is marked inline in source code as `// shortcut: <the limit>, <when to upgrade>` and scanned live by `ateam state debt`. Never maintain a separate tracked `debt-ledger.md` file that can drift from code.
- **Session handoff & pre-compaction cache:** `ateam state handoff` (and the `ateam_state` MCP tool) snapshots live git branch/HEAD/dirty status, OpenSpec queue/holds, forge issues/PRs, and `// shortcut:` items alongside the 5 semantic handoff fields (`Goal`, `Addresses`, `Constraints`, `Done when`, `Checks to run`) into `.ateam/state.json` and formats the 5-field pickup block.

## 2. Work Lifecycle & Definition of Done

1. **Clarify:** Before mutating state on an underspecified request, inspect the tree first, then confirm:
   - **Deliverable** (answer/audit, issue, OpenSpec design, or code/UI change)
   - **Scope and non-goals** (in-bounds packages/files and what stays untouched)
   - **Definition of Done and verification** (`task verify` gates and observable checks)
   - **Stop boundary** (pause after design/plan vs. run through to verified completion)
2. **Claim:** Create or switch to an isolated worktree on an agent-prefixed branch (`claude/…`, `codex/…`, `gemini/…`; in Enjoy desktop, use `move_to_worktree`), backed by a draft PR on GitHub or Gitea.
3. **Execute (Ponytail Decision Ladder):**
   - Take the smallest complete change that solves the task.
   - Reuse existing code and stdlib before adding helpers or dependencies.
   - Add a focused test for every new branch, loop, parser, or boundary condition using an independent oracle.
   - Tag any deliberate limitation with `// shortcut: <the limit>, <when to upgrade>`.
4. **Verify (4-Gate Definition of Done):**
   - **Gate 1 (`task verify`):** `fmt:check`, `spec:check`, `tidy:check`, `vet`, `lint` (`revive`), `test:unit` (`-race`), `ui:check`, `ui:test`, and clean git tree check.
   - **Gate 2 (`ateam doctor` / Parity):** All canonical skills, `.claude/`, `.codex/`, `.gemini/` adapters/hooks, `<24 KB` `AGENTS.md` cap, `<300` token MCP tool schema budget, and `<5ms` hook latency pass.
   - **Gate 3 (`ateam bench --selftest` & `task mutate:check`):** Good/bad reference fixtures and mutation checks pass offline.
   - **Gate 4 (Ponytail Exit Summary):** Reply concludes with 1–2 lines naming what was skipped/not checked and any residual risk.
5. **Handoff or Land:** At a session boundary or before context compaction, run `ateam state handoff` (or `/handoff`) to record the 5-field pickup prompt.

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


## 4. Configured Model Routing & Independent Review

When `.agents/review-policy.json` is configured, validate the route before implementation with `ateam review route`. After any escalation, revalidate against every actual implementation model. Reasoning effort is separate from model strength; neither effort nor a different provider proves superiority. The reviewer must be an independent run that wrote none of the deliverable and outrank every implementer under explicit trusted-policy comparisons. Cross-provider review is optional and additional.

Follow the [developer](contracts/developer.md) and [reviewer](contracts/reviewer.md) contracts. The reviewer remains read-only and the author fixes findings. Review the complete diff and affected callers; instructions, contracts, policy, executable Markdown and rename paths require substantive review. Bind all runs, findings, verdict and actual/reported/unavailable verification to the repository, base/head commits and governing policy digest. New commits or policy revisions invalidate approval.

At the model ceiling, block approval until an authorized owner explicitly grants a head/reviewer/policy-bound exception with a reason. It waives superiority only, never independence, provider constraints, current evidence, required checks or the verdict. `ateam review check` reads trusted baseline policy and authenticated GitHub/Gitea comment authors; candidate policy cannot authorize its own waiver. A configured route and a forge login remain attestations, not proof of model runtime identity or a separate human owner. See [model review](../docs/model-review.md) for the format and limitations.
