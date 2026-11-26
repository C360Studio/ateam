---
name: coordinate
description: >-
  Coordinate milestone work in default mode: take the next issue, start developer and reviewer runs,
  relay review results, and stop at every owner gate with one batched question. Never implements or reviews.
---

# Coordinate Milestone Work

Read [the coordinator contract](../../contracts/coordinator.md) first; it names your write scope, the owner gates and the protocol sections you need. If you are not in `.worktrees/<agent>-coordinator` on `<agent>/coordinator`, create it from `main` and move there before anything else.

## 1. Measure the Queue

Run `ateam state sync`, `ateam state next` and `ateam state queue-check`, list the open pull requests, and read `main`'s latest CI run. Stop the run with a handoff when `main` is red or `next` names nothing; keep its reason for the owner. Queue-check findings go into the owner message in step 4, unchanged.

## 2. Start the Developer Run

For the issue `next` names, start a developer run with your host's own launch tool, the one its `.agents/hosts/<host>.toml` `[launch]` line uses: Enjoy `start_conversation`, herdr `agent start` then `agent prompt`, T3 Code `t3_thread_launch`. Use the developer's default route (ADR 0003: `frontier`, `high`) and this prompt: `Read .agents/contracts/developer.md fully. Take issue #<n>: pick up, claim a worktree on <agent>/<topic> with a draft PR saying Closes #<n>, work to the change's stop boundary, and hand off with ateam state handoff.` When the host has no launch tool, or `ATEAM_HOST` is unset, put the prompt in the owner message instead. Once the draft PR claims the issue, `next` moves on and you may start the next unit in its own worktree.

## 3. Request and Relay Review

When a developer hands off at ready-for-review, run `ateam review request <pr> --return-to <your id>` and follow the tool text it prints. On the reviewer's notify, run `ateam state sync` and read the verdict. On `Verdict: APPROVE`, confirm the reviewer added its `reviewed-by:` line to the PR description, then start or message the developer run to close per protocol § 2 step 5. On `Verdict: REQUEST CHANGES`, pass the `ateam review request --reviewer-session` line to the developer run and post a PR comment linking the record; on the third one for that PR, stop the unit for the owner.

## 4. Ask the Owner Once

When any unit waits on an owner gate, send one message that lists every waiting gate: the PR or issue, the question in plain words, and what you need (for a proposal, the digest `task spec:queue` prints). Then add 1 to the `owner-prompts: <n>` line of each PR that message covers, creating the line at `owner-prompts: 1`, with `gh pr edit <pr> --body-file <file>` (Gitea: `PATCH /repos/{owner}/{repo}/pulls/{index}`). Continue other units while you wait.

## 5. Hand Off

From your coordinator worktree, run `ateam state handoff`: Goal names the milestone you are coordinating, Addresses lists each unit in flight with its PR and the gate it waits on, and Constraints names your provider, model and effort. End with 1–2 lines naming what you skipped or did not check and any residual risk.
