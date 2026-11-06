# Change Proposal: `t3-host-worktree-296`

Approved-by: Coby Leuschke (2026-11-06)

## Why

T3 Code (`pingdotgg/t3code`, main @ 6577c6b6, 2026-11-06) creates each thread's worktree under its own home directory on a `t3/<topic>` branch. Hooks and skills load unchanged through the Agent SDK, but two things do not hold (#296):

- The Claim rule's host exemption names only Enjoy's `move_to_worktree` and herdr (`AGENTS.md:98`, `.agents/protocol.md:32`, `internal/bootstrap/templates.go:639`).
- Ownership comes only from the branch prefix (`inferAgentFromBranch`, `internal/state/state.go:2634`), so a `t3/` worktree reads as nobody's in pickup, and prune keeps it as a bare `no agent prefix` (`internal/state/prune.go:47`). T3 hosts several providers per project, so no prefix can name the agent.

Pickup already lists the worktree (`git worktree list --porcelain`); placement outside `.worktrees/` is not the problem.

## What Changes

1. **Claim text.** Name T3 Code in the exemption in all three places, in the shape used for Enjoy and herdr: the host creates the worktree and the branch; the agent still opens the draft PR and the OpenSpec change. The bootstrap drift test keeps the template and protocol coupled.
2. **Owner fallback.** For a worktree whose branch has no agent prefix, pickup's worktree line names the agent from that worktree's handoff slot, else its writer lease, else `owner unknown`.
3. **Prune reason.** Prune still keeps such a worktree and says `no agent prefix (host-created branch)`.
4. **docs/adoption.md** gains a short T3 Code paragraph: worktree location, `t3/` prefix, `T3CODE_THREAD_ID`, hooks through the SDK, `t3.json` `runOnWorktreeCreate` as the place for `npm ci` or `ateam doctor`, and the open question of whether the prefix is configurable per project.
5. **Spec.** Scenarios on the existing Requirement "Prune finished worktrees the caller owns", which already owns pickup's prune hint; no new Requirement.

## Track

`quick`. One package (`internal/state`) plus prose and one bootstrap template constant. No new Requirement, one capability (`harness`), no exported surface reachable from an adopting repository beyond the existing Claim sentence, and no boot path, durability behavior, or gate file. The owner named the track on 2026-11-06.

## Non-goals

- No change to `inferAgentFromBranch`'s prefix list and no `t3` agent: T3 is a host.
- Host detection, launch, and notify (#295).
- Automatic prune of host-created worktrees; a `t3.json` from `ateam init`.

## Definition of Done

- Tests with attached spec references: pickup line for an unprefixed branch with a handoff slot, with a writer lease only, and with neither; the prune reason text for an unprefixed branch; the bootstrap drift test covering the new Claim sentence.
- `task mutate:check` reports `detection` when the owner-fallback order is swapped (lease before handoff).
- `task verify` green on a clean tree, `ateam doctor` parity green, `ateam bench --selftest` green.
