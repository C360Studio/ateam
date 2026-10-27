---
name: preflight
description: >-
  Select and execute Taskfile verification gates for the current diff and record reproducible evidence.
  Use before implementation pushes, handoffs, or concluding a task.
---

# Verify Changes (Preflight)

Read [the shared protocol](../../protocol.md) first. One `Taskfile.yml` entrypoint serves local verification, MCP (`ateam_task`), and CI.

## 1. Select Gates by Diff Scope

Inspect `git status --short` and `git diff --stat` before choosing gates:

- **Test or specification changes:** `task test:spec`; review whether each reference matches the assertions, since the gate proves syntax and target existence only.
- **Docs or skill definitions only:** `task fmt:check`, `ateam doctor` (`ateam init --check`), and `git diff --check`
- **Go packages (`cmd/`, `internal/`):** `task fmt:check`, `task vet`, `task lint`, `task test:unit`, `task bench:selftest`, and `task mutate:check` for non-trivial refusals/branches
- **SvelteKit UI (`ui/`):** `task ui:check`, `task ui:test`, `task ui:build`
- **Before any push or completion (Full 4-Gate DoD):** `task verify`

Execute gates via the `ateam_task` MCP tool (`{"target": "verify"}`) or `task verify`.

## 2. Record Evidence

Evidence counts only for the `HEAD` SHA (or reproducible dirty snapshot) it measured:

- Record the exact command, exit status, and tested SHA.
- For changes touching parsers, laws, histories, or refusal guards, record the 6-line testing block (`Decision`, `Rule`, `Inputs`, `Run` with seed/completed checks and seed replay vs. `-fuzz` exploration separated, `Mutation` verdict from `task mutate:check`, and `Not covered`).
- Never report historical green before later edits as current proof, and never report a `survivor` or `inconclusive` mutation run as a `detection`.
- End your conclusion with Ponytail's 1–2 line summary: **what was skipped or not checked** and **any residual risk**.
