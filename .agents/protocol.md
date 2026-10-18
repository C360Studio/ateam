# Shared Work Protocol (Claude, Codex, and Gemini)

Canonical protocol for `ateam` and repositories bootstrapped with `ateam init`. `AGENTS.md` and `CLAUDE.md` point here and inline the four workflow gates (**Clarify, Claim, Merge, Close**).

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
   - Add a focused test for every new branch, loop, parser, or boundary condition.
   - Tag any deliberate limitation with `// shortcut: <the limit>, <when to upgrade>`.
4. **Verify (4-Gate Definition of Done):**
   - **Gate 1 (`task verify`):** `fmt:check`, `spec:check`, `tidy:check`, `vet`, `lint` (`revive`), `test:unit` (`-race`), `ui:check`, `ui:test`, and clean git tree check.
   - **Gate 2 (`ateam doctor` / Parity):** All canonical skills, `.claude/`, `.codex/`, `.gemini/` adapters/hooks, `<24 KB` `AGENTS.md` cap, `<300` token MCP tool schema budget, and `<5ms` hook latency pass.
   - **Gate 3 (`ateam bench --selftest`):** Good/bad reference fixtures and mutation checks pass offline.
   - **Gate 4 (Ponytail Exit Summary):** Reply concludes with 1–2 lines naming what was skipped/not checked and any residual risk.
5. **Handoff or Land:** At a session boundary or before context compaction, run `ateam state handoff` (or `/handoff`) to record the 5-field pickup prompt.
