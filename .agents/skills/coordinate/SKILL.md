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

For the issue `next` names, start a developer run with your host's own launch tool, the one its `.agents/hosts/<host>.toml` `[launch]` line uses: Enjoy `start_conversation`, herdr `agent start` then `agent prompt`, T3 Code `t3_thread_launch`. Use the developer's default route (ADR 0003: `frontier`, `high`) and this prompt: `Read .agents/contracts/developer.md fully. Take issue #<n>: pick up, claim a worktree on <agent>/<topic> with a draft PR saying Closes #<n>, work to the change's stop boundary, and hand off with ateam state handoff. After every handoff, send 'developer handoff for #<n> (<reason>); run ateam state sync, then ateam state pickup' to <your id> with the [notify] in .agents/hosts/<host>.toml, where <reason> is ready at ready-for-review, blocked at a blocker or owner gate, or question when you posted a question on the PR.` Replace `<your id>` and `<host>` in it with your own session id and host before you send it. When the host has no launch tool, or `ATEAM_HOST` is unset, put the prompt in the owner message instead. Do not queue fallback wake-ups: the developer's message and the reviewer's notify are the signals you wait on. Once the draft PR claims the issue, `next` moves on and you may start the next unit in its own worktree.

## 3. Request and Relay Review

When a developer hands off at ready-for-review, first read the required checks at the PR head (`gh pr checks <pr> --required`; on Gitea, the head commit's statuses for the contexts branch protection requires). When one other than `Merge check`, which stays absent until the reviewer's `reviewed-by:` line, or, on a PR touching a gate file, `Gate files`, which waits for the owner's approval at the end, failed, is still running or has not reported yet (the command lists only checks that have reported), message the developer run with that check and its state instead of requesting review. Otherwise run `ateam review request --return-to <your id> <pr>` and follow the tool text it prints. On the reviewer's notify, run `ateam state sync` and read the verdict. On `Verdict: APPROVE`, confirm the reviewer added its `reviewed-by:` line to the PR description, then start or message the developer run to close per protocol § 2 step 5. On `Verdict: REQUEST CHANGES`, pass the `ateam review request --reviewer-session` line to the developer run and post a PR comment linking the record; on the third one for that PR, stop the unit for the owner: `ateam review request` refuses the next round until the owner posts a `review-rounds-continue: <n>` line naming that count, so put the summary it prints in your owner message and never post the line yourself.

## 4. Ask the Owner Once

A developer handoff with the reason `question` is an owner gate: read the question from the PR comment its handoff names. When any unit waits on an owner gate, send one message that lists every waiting gate: the PR or issue, the question in plain words, and what you need (for a proposal, the digest `task spec:queue` prints; for a gate-file change, the PR's live head sha and a link to the developer run that claimed it, where the owner's yes goes, since you never post the approval). When the owner answers a developer's question, message that developer run to resume from the answer. Then add 1 to the `owner-prompts: <n>` line of each PR that message covers, creating the line at `owner-prompts: 1`, with `gh pr edit <pr> --body-file <file>` (Gitea: `PATCH /repos/{owner}/{repo}/pulls/{index}`). Continue other units while you wait.

## Hold-my-beer Mode

Only while `.agents/review-policy.json` sets `"hold_my_beer": true`, and only for the gates the coordinator contract lists for a judge: collect the evidence (the proposal and design paths, the review records, the developer's question comment, the milestone and issue numbers, or a status-labelled issue's number with its `ateam state queue-check` finding), run `ateam review compare` for the judge run against every run it rules on, and start the judge with your host's launch tool at ADR 0003's `frontier`, `high` default and this prompt: `Read .agents/contracts/judge.md fully. Question: <one question>. Evidence: <paths and links>. Compare: <ateam review compare output>. When your comment is posted, send 'judge ruling for #<n> posted; run ateam state sync, then read it' to <your id> with the [notify] in .agents/hosts/<host>.toml.`, filling in `<your id>` and `<host>` as in step 2. On a third `Verdict: REQUEST CHANGES`, ask the judge whether each open finding blocks: the developer run fixes the findings it rules blocking and files the rest as follow-up issues, then request a fresh review as in step 3, which the judge's `review-rounds-continue: <n>` line allows. On a ruling on a proposal or a developer's question, message the developer run to resume from it. After a judge removes a status label, run `ateam state sync`, so `ateam state next` can name the issue. A design question comes before its proposal's approval: after the developer run updates `design.md`, start a fresh judge for the approval. When the judge answers "cannot decide", put its question in the owner message in step 4 and end the run.

## 5. Hand Off

From your coordinator worktree, run `ateam state handoff`: Goal names the milestone you are coordinating, Addresses lists each unit in flight with its PR and the gate it waits on, and Constraints names your provider, model and effort. End with 1–2 lines naming what you skipped or did not check and any residual risk.
