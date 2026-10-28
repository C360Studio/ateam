# Change Proposal: `latency-ceiling-241`

Approved-by: Coby Leuschke (2026-10-28)

## Why

`ateam doctor --check` fails when the best of three warm `session-start` runs exceeds 25ms (`25000 us`). That makes wall-clock I/O jitter a hard failure. On shared 2-vCPU CI runners and in larger adopter repositories, the margin is thin: SemTeams measured 17.1ms at v0.3.2 (#207). #207 (`d292428`) halved latency here, but the shortcut walk still visits every file. The owner ruled on 2026-10-28 (#241) that 100ms is the right ceiling.

## What Changes

- The `--check` hard ceiling moves from 25ms (`25000 us`) to 100ms (`100000 us`). The 10ms target and its warning are unchanged.
- `TestHookLatencyBudget` fails above 100ms rather than 25ms, so the same jitter cannot fail `task test:unit` on CI either.
- The rule text changes everywhere it appears: `AGENTS.md` §2, `.agents/rules-index.md`, `.agents/protocol.md` Gate 2, the generated adopter templates in `internal/bootstrap/templates.go`, the harness spec scenario, README, `docs/adoption.md`, `docs/daily-use.md`, and `docs/hook-flow.md`.

## Track

`quick`. It modifies an existing scenario in one capability and adds no Requirement, exported surface, boot path, or gate file. Precedent: `archive/2026-10-26-hook-latency-153`.

## Non-goals

- Changing the 10ms target or the warning.
- A separate CI ceiling or an environment override. One number keeps the rule simple.
- Caching or narrowing the shortcut walk.

## Definition of Done

- `hookLatencyMaxUs = 100000`. `TestHookLatencyFindings` covers the new boundary (100000 passes; 100001 is a violation), and `TestHookLatencyBudget` fails only above 100ms.
- `task mutate:check` reverting the constant to `25000` reports `detection` against `TestHookLatencyFindings`.
- No `25ms` / `25000` ceiling text is left outside `openspec/changes/archive/`.
- `task verify` passes on a clean tree, and `ateam doctor --check` parity passes.
