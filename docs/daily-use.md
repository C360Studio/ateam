# Daily developer guide

This guide covers day-to-day work in a repository that uses default `ateam` settings. Before you begin, run `ateam init`, add a `Taskfile.yml` with a `verify` target, and pass `ateam doctor --check`. `ateam init` writes Go-first instruction templates; adapt `AGENTS.md` to your stack or map the `ateam` commands below in `Taskfile.yml`. In this workflow, you set scope, approve proposals, trigger independent reviews, and merge pull requests. The agent writes code, runs checks, and records state. If you have not initialized your repository yet, read [the adoption guide](adoption.md) or [the brownfield guide](brownfield.md) first.

## Roles at a glance

| Phase | What you do | What `ateam` or the agent does |
| --- | --- | --- |
| Orient | Start your agent CLI or open `ateam serve`. | `ateam hook` injects branch, `HEAD`, `Hold:`, shortcut, and `full` mode state. |
| Clarify | Confirm scope, pick `quick` or `design`, and sign `Approved-by:`. | The agent inspects the tree, drafts `proposal.md` + `tasks.md` (plus read-only `design.md` on `design`), and pauses on Task 1's `OPEN-Q` hold. |
| Claim and build | Steer `/ponytail` mode; run parallel tasks in separate worktrees. | The agent claims `.worktrees/<agent>-<topic>`, opens a draft PR, writes code and tests, and tracks holds/shortcuts. |
| Pause or switch | Run `/handoff` (or `ateam state handoff --goal "..."`) before stopping. | `ateam` saves `.ateam/state.json`, clears the writer lease, and comments on the open PR. |
| Ready and review | Open a fresh session and run `/review-pr <pr>`. | The implementer runs `/preflight`, archives specs, and marks ready; the reviewer verifies and posts `reviewed-by:`. |
| Merge and clean up | Squash-merge the approved PR (or arm auto-merge). | After merge, the claiming agent deletes its worktree and remote branch (`ateam state prune`). |

## 1. Start or resume a session

1. Start Claude Code, Codex CLI, Gemini CLI, or Antigravity (`agy`) in your repository.
2. On startup, `ateam hook` injects your branch, `HEAD`, `Hold:` count, shortcuts, and Ponytail `full` mode ([hook flow](hook-flow.md)).
3. When you resume earlier work or switch agent CLIs, run `/pickup` ([pickup skill](../.agents/skills/pickup/SKILL.md)) or `ateam state pickup`.
4. `ateam state pickup` reconciles live Git worktrees and GitHub or Gitea state against the last checkpoint.
5. When you want a browser board for issues, holds, and shortcuts, run `ateam serve` and open `http://127.0.0.1:8420`.
6. When you want to refresh cached GitHub or Gitea issues and PRs manually, run `ateam state sync`.

## 2. Clarify scope, pick a track, and approve the proposal

1. Give the agent an issue number or a task description.
2. When a request leaves choices open, the agent MUST stop before editing and confirm:
   - **Deliverable:** What output the agent produces and which role drafts it ([ADR 0003](adr/0003-roles-and-routes.md)).
   - **Scope and non-goals:** Which packages or files change, and which files stay untouched.
   - **Definition of Done:** Which `Taskfile.yml` or `ateam` commands prove the change works.
   - **Stop boundary:** Whether the agent pauses after the proposal/design (default for non-trivial work) or runs through verification.
   - **Track (non-trivial work):** `quick` or `design` (trivial fixes and docs-only edits skip `openspec/changes/` altogether).
3. On the **`quick` track** (the default when no `design` trigger applies), the developer creates `openspec/changes/<id>/proposal.md` (at most 60 lines, with an empty `Approved-by:` header) and `openspec/changes/<id>/tasks.md` (plus any `specs/<capability>/spec.md` scenario edits), with **no `design.md`**.
4. A change MUST take the **`design` track** when it **(a)** adds a spec `Requirement`, **(b)** touches more than one capability under `openspec/specs/`, **(c)** adds exported surface reachable from an adopting repository, or **(d)** touches a boot path, durability/ack behavior, or a gate file.
5. On `design`, `/design` ([design skill](../.agents/skills/design/SKILL.md)) runs under the read-only [architect contract](../.agents/contracts/architect.md): it drafts `design.md` (at most 120 lines) in OpenSpec's `spec-driven` sections (`file:line` context inventory, at least two options per decision with cost/blast radius/foreclosures and the adopter seam, and marked owner-ruling `Open Questions`) and ties each task in `tasks.md` to its `file:line` premise and acceptance check. The architect never rules or writes code.
6. On both non-trivial tracks, Task 1 in `tasks.md` starts with `Hold: OPEN-Q proposal awaits owner approval` (surfaced under `OPEN-Q` by `ateam state spec-queue` and `pickup`), and the agent stops there and hands off.
7. One edit approves the change: fill in `Approved-by: <name> (<YYYY-MM-DD>)` in `proposal.md` (recording any Open Question rulings) and delete that `Hold:` line in `tasks.md`; on `design`, that single edit lifts the gate for both `proposal.md` and `design.md`.
8. Run `ateam state spec-check` (`task spec:check`) to validate OpenSpec structure and task holds; it runs `openspec validate --all --strict --no-interactive` when `openspec` is installed and warns, without failing, past 60 lines (`proposal.md`) or 120 lines (`design.md`). On code PRs, `ateam review gate` (`task merge:check`) prints a non-blocking note when `proposal.md` has no filled `Approved-by:`.

## 3. Claim a worktree and build

For each task, the agent creates `.worktrees/<agent>-<topic>` on branch `<agent>/<topic>` (`claude/…`, `codex/…`, or `gemini/…`) and opens a draft PR. Because `.ateam/writer.json` tracks one worktree at a time, you can run separate agents in parallel across separate worktrees.

- If a second live session enters the same worktree, `ateam` prints a competing-writer warning; move to a separate worktree.
- When you want lighter or stricter scope pushback, run `/ponytail lite`, `/ponytail full`, `/ponytail ultra`, or `/ponytail off` ([ponytail skill](../.agents/skills/ponytail/SKILL.md)).
- In Antigravity (`agy`), switch modes with `ateam state mode <level>` or `/ponytail` because `PreInvocation` stdin omits prompt text.
- When an external blocker stops a task, the agent adds `Hold: <reason>` under that task in `tasks.md` (`ateam state spec-queue`).
- When the agent defers an edge case in code, it adds `// shortcut: <limit>, <when to upgrade>` (`ateam state debt`).
- When you maintain `openspec/specs/`, the agent links Go and Vitest tests with `// spec:` comments (`ateam test-spec`).
- For new Go branches or parsers, `ateam mutate` tests a mutant copy in `/tmp` without modifying your worktree.

## 4. Pause, compact, or switch agents

1. Before you stop or switch agent CLIs, run `/handoff` ([handoff skill](../.agents/skills/handoff/SKILL.md)) or pass `--goal` to `ateam state handoff`:

   ```sh
   ateam state handoff \
     --goal "Add Gitea webhook check (#12)" \
     --addresses "PR #15, branch claude/gitea-hmac" \
     --constraints "None" \
     --done-when "task verify passes and PR #15 merges" \
     --checks "task verify"
   ```

2. The CLI requires `--goal` and accepts `--addresses` (defaults to branch and SHA), `--constraints`, `--done-when`, and `--checks` (defaults to `task verify`).
3. `ateam state handoff` writes those five fields to `.ateam/state.json`, clears `.ateam/writer.json`, and posts the checkpoint to the open PR.
4. When you have no open PR or want a local checkpoint only, add `--no-post` to `ateam state handoff`.
5. In Claude Code, Codex CLI, and Gemini CLI, `ateam init` sets ~500k auto-compaction so `pre-compact` saves local state.
6. Antigravity (`agy`) has no compaction threshold setting; run `/compact` or `/handoff` manually during long sessions.

## 5. Prepare the PR and run an independent review

1. Before marking a PR ready, the agent MUST pass `/preflight` ([preflight skill](../.agents/skills/preflight/SKILL.md)).
2. Preflight runs your `task verify` target, plus `ateam test-spec` or `ateam mutate` when those checks apply.
3. When a PR edits `AGENTS.md`, `CLAUDE.md`, or `.agents/`, run `ateam doctor --check` to verify setup parity.
4. Before marking the PR ready, the agent MUST commit moving completed `openspec/changes/<id>/` work to `openspec/changes/archive/<YYYY-MM-DD>-<id>/`.
5. On any code PR, the agent adds `implemented-by: <agent> (<model>, <effort>)` to the PR description and runs `gh pr ready`.
6. Open a fresh agent session with an equal or stronger model that wrote none of the PR.
7. By default, any provider may review; add `.agents/review-policy.json` only when you want tier or cross-provider rules ([model review](model-review.md)).
8. Run `/review-pr <pr-number>` ([review-pr skill](../.agents/skills/review-pr/SKILL.md)) in that fresh session.
9. The reviewer checks a detached worktree, runs `task verify`, posts `reviewed-by:` with `Reviewed at <sha>` ([reviewer contract](../.agents/contracts/reviewer.md)), and cleans up.
10. When the verdict is `APPROVE`, the reviewer adds `reviewed-by: <agent> (<model>, <effort>)` to the PR description.
11. When the verdict is `REQUEST CHANGES`, the author fixes the findings and requests a fresh review on the new commit.

## 6. Merge and clean up

1. Once your checks pass and the review approves, squash-merge the PR (`gh pr merge --squash`) or merge on Gitea.
2. When your repository enables branch protection and auto-merge, arm `gh pr merge --squash --auto` (or Gitea `merge_when_checks_succeed`) instead.
3. To enforce review records, `main` rulesets, and `class:flake` blocks in CI, wire in `ateam review gate <pr-number>` ([merge gate setup](adoption.md#the-merge-gate-on-github-and-gitea)).
4. After the PR merges, the claiming agent removes `.worktrees/<agent>-<topic>` and deletes the remote branch; `ateam state prune` does both for every merged, clean worktree on its prefix (`--dry-run` lists them first).
5. An agent MUST NOT delete a worktree that belongs to another agent or contains uncommitted or unpushed work.

## What to expect day to day

- **Fast hooks and small schemas:** `ateam hook` runs in under 10ms (100ms `--check` limit), and `ateam mcp` uses about 190 schema tokens ([MCP guide](mcp.md)).
- **Explicit blind-spot lines:** At the end of each task conclusion and each handoff, the agent states what it skipped and what risk remains in one or two lines. Acknowledgements and progress updates skip this, but still name blockers, failed or unavailable checks, and changed risks.
- **Advisory hooks versus hard CI gates:** `ateam hook` and Ponytail modes inject prompt guidance; they do not block edits, enforce obedience, or run tests automatically.
- **OpenSpec track versus Ponytail mode:** `quick` versus `design` sets pre-code planning depth and OpenSpec artifacts; `/ponytail` (`lite`, `full`, `ultra`, `off`) adjusts injected code-scope reminders during implementation without changing OpenSpec requirements.
- **Local checks versus optional CI gates:** `task verify` and `ateam doctor --check` run locally; add `ateam review gate` in CI to block unreviewed code PRs.
- **Commit-bound reviews:** Pushing a code commit after review invalidates `Reviewed at <sha>` and requires a fresh review comment.
- **`Merge check` waits for the review record (when wired in CI):** ateam's `merge-check.yml` skips it as `Merge check (awaiting review record, skipped)` until the description has a `reviewed-by:` line, so it is absent, not red, before `/review-pr` comments; a gate run without that skip fails when the PR is marked ready. The reviewer's description edit runs it; after a rejection that follows an approval at the same head, the reviewer reruns it and confirms it fails.
- **Shared token warning in `ateam doctor`:** Solo setups see a non-failing warning until `ATEAM_TOKEN_READ` and `ATEAM_TOKEN_WRITE` use separate tokens ([forge tokens](adoption.md#forge-tokens)).
- **MCP reload reminder after upgrades:** After you upgrade the `ateam` binary, `session-start` warns if a running `ateam mcp` process is older; restart your agent CLI.
- **Go-first tool limits:** Coordination works in any language, while `ateam mutate` accepts only non-test `.go` files and `debt`/`test-spec` scan fixed file extensions ([adoption limits](adoption.md#feature-and-language-boundaries)).
