# Daily developer guide

This guide covers day-to-day work in an initialized repository that uses the default `pr-comment` review mode and passes `ateam doctor --check`. If `.agents/review-policy.json` sets `"review_record": "aip"` (no PR comments or squash merges), follow your generated [`.agents/protocol.md`](../.agents/protocol.md) and [review record mode](model-review.md#review-record-mode-review_record) instead. In this workflow, you set scope, approve proposals, trigger reviews, and approve gate changes; the agent writes code, runs checks, and records state. If you have not initialized the repository yet, read [the adoption guide](adoption.md) or [the brownfield guide](brownfield.md) first.

## Roles at a glance

| Phase | What you do | What `ateam` or the agent does |
| --- | --- | --- |
| Orient | Start your agent CLI or open `ateam serve`. | `ateam hook` injects branch, worktree, `Hold:`, shortcut, and mode state. |
| Clarify | Confirm scope and sign `Approved-by:` in `proposal.md`. | The agent inspects the tree, drafts `proposal.md`, and pauses on Task 1. |
| Claim and build | Steer `/ponytail` mode; run parallel tasks in separate worktrees. | The agent works in `.worktrees/<agent>-<topic>`, tracks holds/shortcuts, and mutates Go tests. |
| Pause or switch | Run `/handoff` (or `ateam state handoff --goal "..."`) before stopping. | `ateam state handoff --goal` saves `.ateam/state.json`, clears the writer lease, and comments on the PR. |
| Ready and review | Open a fresh session and run `/review-pr <pr>`. | The implementer archives specs and marks ready; the reviewer verifies and posts `reviewed-by:`. |
| Merge and clean up | Apply `gate-change-approved` only when a PR edits gate files. | Auto-merge lands the green PR, and the claiming agent deletes its worktree and branch. |

## 1. Start or resume a session

1. Start Claude Code, Codex CLI, Gemini CLI, or Antigravity (`agy`) in the repository.
2. On startup, `ateam hook` reads `.ateam/state.json` and injects your branch, `HEAD`, worktrees, `Hold:` count, shortcuts, and Ponytail mode ([hook flow](hook-flow.md)).
3. When you want a browser view of issues, holds, and shortcuts, run `ateam serve` and open `http://127.0.0.1:8420`.
4. When you want to refresh cached GitHub or Gitea issues and PRs manually, run `ateam state sync`.
5. When you resume earlier work or switch agent CLIs, run `/pickup` ([pickup skill](../.agents/skills/pickup/SKILL.md)) or `ateam state pickup`.
6. `ateam state pickup` compares live Git and forge state against the last checkpoint before the agent edits files.

## 2. Clarify scope and approve the proposal

1. Give the agent an issue number or a task description.
2. When a request leaves choices open, the agent MUST stop before editing and confirm four items with you:
   - **Deliverable:** What output the agent produces (an answer, an issue, an OpenSpec proposal, or a code change).
   - **Scope and non-goals:** Which packages or files change, and which files stay untouched.
   - **Definition of Done:** Which `Taskfile.yml` commands prove the change works.
   - **Stop boundary:** Whether the agent pauses after the proposal or runs through verification.
3. For non-trivial work, the agent creates `openspec/changes/<id>/proposal.md` (at most 60 lines) and `openspec/changes/<id>/tasks.md`.
4. The agent adds `Hold: OPEN-Q proposal awaits owner approval` under Task 1 and hands off to you.
5. Before implementation begins, read `proposal.md`, fill in `Approved-by: <name> (<YYYY-MM-DD>)`, and delete that `Hold:` line in `tasks.md`.

## 3. Claim a worktree and build

For each task, the agent creates `.worktrees/<agent>-<topic>` on branch `<agent>/<topic>` (`claude/…`, `codex/…`, or `gemini/…`) and opens a draft PR. In Enjoy desktop, the agent calls `move_to_worktree` instead of `git worktree add`. Because `.ateam/writer.json` tracks one worktree at a time, you can run separate agents in parallel across separate worktrees.

- If a second live session enters the same worktree, `ateam` prints a competing-writer warning; move to a separate worktree.
- When you want lighter or stricter scope pushback, run `/ponytail lite`, `/ponytail full`, `/ponytail ultra`, or `/ponytail off` ([ponytail skill](../.agents/skills/ponytail/SKILL.md)).
- In Antigravity (`agy`), switch modes with `ateam state mode <level>` or `/ponytail` because `PreInvocation` stdin omits prompt text.
- When an external blocker stops a task, the agent adds `Hold: <reason>` under that task in `tasks.md` (`task spec:queue`).
- When the agent defers an edge case in code, it adds `// shortcut: <limit>, <when to upgrade>` (`ateam state debt`).
- The agent attaches `// spec:` or `//aip:verifies` comments to Go and Vitest tests (`task test:spec`).
- For new Go branches or parsers, `task mutate:check` tests a mutant copy in `/tmp` without modifying your worktree.

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
4. When you want a local checkpoint without a PR comment, add `--no-post` to that `ateam state handoff --goal "..."` command.
5. In Claude Code, Codex CLI, and Gemini CLI, `ateam init` sets auto-compaction near 500k tokens so `pre-compact` saves local state early.
6. Antigravity (`agy`) has no compaction threshold setting; run `/compact` or `/handoff` manually during long sessions.

## 5. Prepare the PR and run an independent review

1. Before marking a PR ready, the agent MUST pass `/preflight` ([preflight skill](../.agents/skills/preflight/SKILL.md)).
2. In `ateam`, preflight runs `task verify`, `ateam doctor --check`, and `ateam bench --selftest`; adopting repositories supply their own `task verify` target.
3. Before marking the PR ready, the agent MUST move any completed `openspec/changes/<id>/` directory to `openspec/changes/archive/<YYYY-MM-DD>-<id>/` in a branch commit.
4. On any code PR, the agent adds `implemented-by: <agent> (<model>, <effort>)` to the PR description and marks the PR ready (`gh pr ready`).
5. Open a fresh agent session whose model and effort meet or exceed every implementer in `.agents/review-policy.json` ([model review](model-review.md)).
6. When `cross_provider` is `true` in `.agents/review-policy.json`, you MUST choose a different agent provider for the review.
7. Run `/review-pr <pr-number>` ([review-pr skill](../.agents/skills/review-pr/SKILL.md)) in that fresh session.
8. The reviewer checks a detached worktree, runs `task verify`, posts `reviewed-by:` with `Reviewed at <sha>` ([reviewer contract](../.agents/contracts/reviewer.md)), and deletes its review worktree.
9. When the verdict is `APPROVE`, the reviewer adds `reviewed-by:` to the PR description and reruns the failed `Merge check` job.
10. When the verdict is `REQUEST CHANGES`, the reviewer stops so `Merge check` stays red while the author fixes findings.

## 6. Merge and clean up

1. Once `task merge:check -- <pr-number>` passes, arm squash auto-merge (`gh pr merge --squash --auto`), or post `merge_when_checks_succeed` on Gitea.
2. When required checks pass on an up-to-date branch, GitHub or Gitea squashes and merges the PR.
3. After the PR merges, the claiming agent removes `.worktrees/<agent>-<topic>` and deletes the remote branch.
4. An agent MUST NOT delete a worktree that belongs to another agent or contains uncommitted or unpushed work.

## What to expect day to day

- **Fast hooks and small schemas:** `ateam hook` runs in under 10ms (25ms `--check` limit), and `ateam mcp` uses about 190 schema tokens ([MCP guide](mcp.md)).
- **Explicit blind-spot lines:** At the end of every reply, the agent states what it skipped and what risk remains in one or two lines.
- **Advisory hooks versus hard CI gates:** `ateam hook` and Ponytail modes inject prompt guidance; they do not block edits, enforce obedience, or run tests automatically.
- **Hard merge checks:** `task verify`, `ateam doctor --check`, and `ateam review gate` (`task merge:check -- <pr>`) reject broken builds, parity drift, or unreviewed code PRs.
- **Owner label for gate files:** If a PR edits CI workflows, `.agents/review-policy.json`, merge scripts, or `internal/review/`, the agent hands off until you add `gate-change-approved`.
- **One expected red `Merge check`:** Marking a code PR ready triggers CI before `/review-pr` posts its comment; the reviewer reruns that failed job after approving.
- **Commit-bound reviews and flake blocks:** Pushing a code commit after review invalidates `Reviewed at <sha>`, and any open `class:flake` issue fails `task merge:check`.
- **Stale binary reminders:** In the `ateam` repository, `session-start` warns when `HEAD` passes your local build; run `task build` and restart `ateam mcp`.
- **Go-first tool limits:** Coordination works in any language, while `ateam mutate` accepts only non-test `.go` files and `debt`/`test-spec` scan fixed file extensions ([adoption limits](adoption.md#feature-and-language-boundaries)).
