# Daily developer guide

This guide covers day-to-day work in a repository that uses default `ateam` settings. Before you begin, run `ateam init`, add a `Taskfile.yml` with a `verify` target, and pass `ateam doctor --check`. `ateam init` writes Go-first instruction templates; adapt `AGENTS.md` to your stack or map the `ateam` commands below in `Taskfile.yml`. In this workflow, you set scope, approve proposals, trigger independent reviews, and merge pull requests. The agent writes code, runs checks, and records state. If you have not initialized your repository yet, read [the adoption guide](adoption.md) or [the brownfield guide](brownfield.md) first.

## Roles at a glance

| Phase | What you do | What `ateam` or the agent does |
| --- | --- | --- |
| Orient | Start your agent CLI or open `ateam serve`. | `ateam hook` injects branch, `HEAD`, `Hold:`, shortcut, and Ponytail mode state. |
| Clarify | Confirm scope, pick `quick` or `design`, and sign `Approved-by:`. | The agent inspects the tree, drafts `proposal.md` + `tasks.md` (plus read-only `design.md` on `design`), and pauses on Task 1's `OPEN-Q` hold. |
| Claim and build | Steer `/ponytail` mode; run parallel tasks in separate worktrees. | The agent claims `.worktrees/<agent>-<topic>`, opens a draft PR, writes code and tests, and tracks holds/shortcuts. |
| Pause or switch | Run `/handoff` (or `ateam state handoff --goal "..."`) before stopping. | `ateam` saves `.ateam/state.json`, clears the writer lease, and comments on the open PR. |
| Ready and review | Open a fresh session and run `/review-pr <pr>`. | The implementer runs `/preflight` and marks ready; the reviewer verifies and posts `reviewed-by:`; after approval the implementer archives the OpenSpec change (on Gitea, before ready). |
| Merge and clean up | Squash-merge the approved PR (or arm auto-merge). | After merge, the claiming agent deletes its worktree and remote branch (`ateam state prune`). |
| Coordinate (optional) | Run `/coordinate` in a fresh session and answer its one batched message of owner decisions ([section 7](#7-coordinate-a-milestone-optional)). | The coordinator takes the issue `ateam state next` names and starts and relays the developer and reviewer runs. |

## 1. Start or resume a session

1. Start Claude Code, Codex CLI, Gemini CLI, or Antigravity (`agy`) in your repository.
2. On startup, `ateam hook` injects your branch, `HEAD`, `Hold:` count, shortcuts, and the Ponytail ladder when its mode is `on` ([hook flow](hook-flow.md)).
3. When you resume earlier work or switch agent CLIs, run `/pickup` ([pickup skill](../.agents/skills/pickup/SKILL.md)) or `ateam state pickup`.
4. `ateam state pickup` reconciles live Git worktrees and GitHub or Gitea state against the last checkpoint. Its `next=` field, and `ateam state next` in full, name the one next issue every session shares: the first startable issue in the earliest milestone, ordered by an `Order: #a -> #b` line in each milestone description and an `After: <milestone title>` line naming the milestone it follows. `Depends on: #n` lines in an issue's body or comments hold it back, and an open PR whose body says `Closes`, `Fixes`, `Resolves` or `Addresses #n` claims it. When the order cannot be read, `next=none (<reason>)` names nothing; issues no `Order:` line places are listed as unplaced. Its `queue=` field, and `ateam state queue-check` in full, count queue drift: a `status:blocked` issue whose `Depends on:` issues are all closed or that has none (Q1, Q2), a `status:needs-decision` issue with a recorded decision (Q3), an unplaced issue (Q4), a milestone with more open issues than `queue.max_milestone_issues` (Q5), and an open milestone with none left (Q6). `queue=unavailable` means nothing is synced or the milestones read did not succeed. The findings are advice: nothing fails on them, and `ateam doctor` prints the count.
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
7. One edit approves the change: fill in `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` in `proposal.md`, with the digest `ateam state spec-queue` prints, (recording any Open Question rulings) and delete that `Hold:` line in `tasks.md`; on `design`, that single edit lifts the gate for both `proposal.md` and `design.md`.
8. Run `ateam state spec-check` (`task spec:check`) to validate OpenSpec structure and task holds; it runs `openspec validate --all --strict --no-interactive` when `openspec` is installed and warns, without failing, past 60 lines (`proposal.md`), 120 lines (`design.md`), 500 characters (a delta Requirement) or 60 words (a scenario step). It fails a change with a `design.md` whose `proposal.md` has no bullet under `## Decisions you are making by approving`, and warns on a quick change without one. It fails when a filled `Approved-by:` digest no longer matches `proposal.md` and `design.md`, naming the edited file, its last commit or `uncommitted`, and the digest to re-approve with, and warns when the line carries no digest. On code PRs, `ateam review gate` (`task merge:check`) prints a non-blocking note when `proposal.md` has no filled `Approved-by:`.

## 3. Claim a worktree and build

For each task, the agent creates `.worktrees/<agent>-<topic>` on branch `<agent>/<topic>` (`claude/…`, `codex/…`, or `gemini/…`) and opens a draft PR. Because `.ateam/writer.json` tracks one worktree at a time, you can run separate agents in parallel across separate worktrees.

- If a second live session enters the same worktree, `ateam` prints a competing-writer warning; move to a separate worktree.
- To turn the Ponytail ladder reminder on or off, run `/ponytail on` or `/ponytail off` ([ponytail skill](../.agents/skills/ponytail/SKILL.md)). Effort is set by the OpenSpec quick/design track, not by Ponytail.
- In Antigravity (`agy`), switch modes with `ateam state mode <level>` or `/ponytail` because `PreInvocation` stdin omits prompt text.
- When an external blocker stops a task, the agent adds `Hold: <reason>` under that task in `tasks.md` (`ateam state spec-queue`).
- When the agent defers an edge case in code, it adds `// shortcut: <limit>, <when to upgrade>` (`ateam state debt`).
- When you maintain `openspec/specs/`, the agent links Go, Vitest, and Rust tests with `// spec:` comments (`ateam test-spec`).
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
3. `ateam state handoff` writes those five fields to `.ateam/state.json`, clears `.ateam/writer.json`, posts the checkpoint to the open PR, and sets the informational `ateam/proposal`, `ateam/handoff`, and `ateam/review-record` commit statuses on the PR head ([table](adoption.md#informational-commit-statuses)). `ateam state sync` refreshes them for every open PR with a worktree here. None is a required check; `ATEAM_COMMIT_STATUSES=0` turns them off.
4. When you have no open PR or want a local checkpoint only, add `--no-post` to `ateam state handoff`; it skips the comment and the statuses.
5. In Claude Code, Codex CLI, and Gemini CLI, `ateam init` sets ~500k auto-compaction so `pre-compact` saves local state.
6. Antigravity (`agy`) has no compaction threshold setting; run `/compact` or `/handoff` manually during long sessions.

## 5. Prepare the PR and run an independent review

1. Before marking a PR ready, the agent MUST pass `/preflight` ([preflight skill](../.agents/skills/preflight/SKILL.md)).
2. Preflight runs your `task verify` target, plus `ateam test-spec` or `ateam mutate` when those checks apply.
3. When a PR edits `AGENTS.md`, `CLAUDE.md`, or `.agents/`, run `ateam doctor --check` to verify setup parity.
4. The agent marks the PR ready with its OpenSpec change still under `openspec/changes/<id>/` and the change's last task, the archive, open; a `REQUEST CHANGES` round appends its tasks there. On Gitea, the agent instead moves the change to `openspec/changes/archive/<YYYY-MM-DD>-<id>/` before marking the PR ready.
5. On any code PR, the agent adds `implemented-by: <agent> (<model>, <effort>)` to the PR description and runs `gh pr ready`.
6. Open a fresh agent session with an equal or stronger model that wrote none of the PR.
7. By default, any provider may review; add `.agents/review-policy.json` only when you want tier or cross-provider rules ([model review](model-review.md)).
8. Run `/review-pr <pr-number>` ([review-pr skill](../.agents/skills/review-pr/SKILL.md)) in that fresh session.
9. The reviewer checks a detached worktree, runs `task verify`, posts `reviewed-by:` with `Reviewed at <sha>` ([reviewer contract](../.agents/contracts/reviewer.md)), and cleans up.
10. When the verdict is `APPROVE`, the reviewer adds `reviewed-by: <agent> (<model>, <effort>)` to the PR description.
11. When the verdict is `REQUEST CHANGES`, the author fixes the findings and runs the `ateam review request --reviewer-session <id> --return-to <your id> <pr>` line from the reviewer's notify message. It sends the new commit back to the same reviewer when the reviewer can still pass the merge gate and the host can reach it, and launches a fresh reviewer otherwise (`--fresh` forces one; the JSON's `rereview` field says which happened and why). The same line works after `APPROVE`: when a catch-up merge of `main` touches the PR's files and `task merge:check` says the record is older than a content commit, it asks the same reviewer to review only the commits since its record. After a third `Verdict: REQUEST CHANGES` on one PR, `ateam review request` exits 2 and prints the three records: the next round waits until you (or, in hold-my-beer mode, a judge) post a `review-rounds-continue: <n>` line in a PR comment, `<n>` being the number of rejections. It reads the last `ateam state sync`.

## 6. Merge and clean up

1. Once the review approves a PR that lands an OpenSpec change, the agent moves the change to `openspec/changes/archive/<YYYY-MM-DD>-<id>/` in one document-only commit, which needs no new review on GitHub. Then, once your checks pass, merge the PR with the configured method (`squash` by default, so `gh pr merge --squash`; `merge_method` in [`.agents/ateam.json`](adoption.md#team-settings) selects `--merge` or `--rebase`) or merge on Gitea with the matching `do` value.
2. When your repository enables branch protection and auto-merge, arm `gh pr merge --<method> --auto` (or Gitea `merge_when_checks_succeed`) instead.
3. To enforce review records, `main` rulesets, and `class:flake` blocks in CI, wire in `ateam review gate <pr-number>` ([merge gate setup](adoption.md#the-merge-gate-on-github-and-gitea)).
4. After the PR merges, the claiming agent removes `.worktrees/<agent>-<topic>` and deletes the remote branch; `ateam state prune` does both for every merged, clean worktree on its prefix that no other session holds or handed off in the last 30 minutes (`--dry-run` lists them first).
5. An agent MUST NOT delete a worktree that belongs to another agent or contains uncommitted or unpushed work.
6. When an epic or milestone closes, read the repository as one system and record it with `ateam review whole --at <sha>` (an annotated `review/whole/<date>-<short sha>` tag, pushed unless `--no-push`). Per-PR review sees one diff at a time; the pickup line's `since-review=<n> commits, +<a>/-<d> lines (<date>)`, or `since-review=never`, shows how much has landed since the last whole read. It is advice, not a gate.

## 7. Coordinate a milestone (optional)

Run `/coordinate` in a fresh session to have one agent move a milestone along for you. It is a skill, a slash command in your agent CLI, not an `ateam` subcommand. `ateam init` installs it, and Codex and Antigravity read `.agents/skills/coordinate/SKILL.md` directly.

1. **Prerequisites.** The milestone needs an `Order: #a -> #b` line, so `ateam state next` has an order (section 1). To start developer and reviewer runs the coordinator uses your host's launch tool, found in `.agents/hosts/<host>.toml`. `ateam init` does not write that directory: copy the manifest for your host from this repository (`enjoy.toml`, `herdr.toml` or `t3code.toml`; see [review launch hosts](adoption.md#optional-review-launch-hosts)). With no manifest or launch tool, the coordinator puts each prompt in its message to you instead.
2. **What it does.** It works from its own `<agent>/coordinator` worktree with no PR, takes the issue `ateam state next` names, starts a developer run, requests and relays reviews, and never implements, reviews, or rules ([coordinator contract](../.agents/contracts/coordinator.md)).
3. **What stops it.** Every decision that is yours: a proposal awaiting `Approved-by:`, a developer's question on its PR, a change touching a gate file, an issue labelled `status:needs-decision` or `status:blocked`, a third rejected review, any question of milestone scope or order (an unplaced issue included), and a release. It asks you once for all of them and records `owner-prompts: <n>`, the count of those messages, in the body of each PR it moves. It ends the whole run when `main` CI is red or `ateam state next` names nothing.
4. **Hold-my-beer mode.** To hand some of those decisions to a judge instead, set `"hold_my_beer": true` in `.agents/review-policy.json` (a gate-file change you approve). The coordinator then starts a judge ([judge contract](../.agents/contracts/judge.md)) for a proposal and its design, a developer's or design question, a third rejected review, milestone scope or order, `status:needs-decision`, and a `status:blocked` issue whose `Depends on:` issues are all closed. See [hold-my-beer mode](model-review.md#hold-my-beer-mode-hold_my_beer). Gate-file approval and releases stay yours.

## What to expect day to day

- **Fast hooks and small schemas:** `ateam hook` runs in under 10ms (100ms `--check` limit), and `ateam mcp` stays under 300 schema tokens (`ateam doctor` prints `mcp_schema_tokens`; [MCP guide](mcp.md)).
- **Explicit blind-spot lines:** At the end of each task conclusion and each handoff, the agent states what it skipped and what risk remains in one or two lines. Acknowledgements and progress updates skip this, but still name blockers, failed or unavailable checks, and changed risks.
- **Advisory hooks versus hard CI gates:** `ateam hook` and Ponytail modes inject prompt guidance; they do not block edits, enforce obedience, or run tests automatically.
- **OpenSpec track versus Ponytail mode:** `quick` versus `design` sets pre-code planning depth and OpenSpec artifacts; `/ponytail` (`on`, `off`) toggles the injected code-scope reminder during implementation without changing OpenSpec requirements.
- **Local checks versus optional CI gates:** `task verify` and `ateam doctor --check` run locally; add `ateam review gate` in CI to block unreviewed code PRs.
- **Commit-bound reviews:** Pushing a code commit after review invalidates `Reviewed at <sha>` and requires a fresh review comment.
- **`Merge check` waits for the review record (when wired in CI):** ateam's `merge-check.yml` skips it while a code PR has no `reviewed-by:` description line and no passing `Merge check` already stands at its head, so a record posted before the line is not a red run. GitHub shows a skipped job under its raw `name:` expression, never as `Merge check`, so the required check is absent, not red, until the line is written (documents-only PRs are judged at once); a gate run without that skip fails when the PR is marked ready. On GitHub the probe job needs `checks: read` to see that earlier pass; a probe job written before v0.4.0 lacks it and shows a red `Merge check` instead ([merge gate](adoption.md#the-merge-gate-on-github-and-gitea)). The reviewer's description edit runs it; after a rejection that follows an approval at the same head, the reviewer reruns it and confirms it fails.
- **Shared token warning in `ateam doctor`:** Solo setups see a non-failing warning until `ATEAM_TOKEN_READ` and `ATEAM_TOKEN_WRITE` use separate tokens ([forge tokens](adoption.md#forge-tokens)).
- **MCP reload reminder after upgrades:** After you upgrade the `ateam` binary, `session-start` warns if a running `ateam mcp` process is older; restart your agent CLI.
- **Go-first tool limits:** Coordination works in any language, while `ateam mutate` accepts only non-test `.go` files and `debt`/`test-spec` scan fixed file extensions ([adoption limits](adoption.md#feature-and-language-boundaries)). A Cargo repository gets the `.rs` debt scan and Rust test attribution, and runs `cargo-mutants` by hand ([Rust repositories](adoption.md#rust-repositories)).
