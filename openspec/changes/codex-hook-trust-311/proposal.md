# Change Proposal: `codex-hook-trust-311`

Approved-by: Coby Leuschke (2026-11-08)

## Why

`ateam doctor` passes the Codex hook row when `.codex/hooks.json` has the right shape (`internal/bootstrap/bootstrap.go:739`), but Codex runs project hooks only after the user has trusted them once (#311). In a checkout that hasn't been trusted, every Codex session (CLI, Enjoy, T3) runs no `ateam hook`, and nothing says so: there's no `[ateam]` line, no `.ateam/state.json` write, and doctor stays green. The owner confirmed the cause on 2026-11-07. Trust is stored in `~/.codex/config.toml` as one `[hooks.state."<absolute hooks.json path>:<event>:<i>:<j>"]` table per hook, so each git worktree needs its own approval. On this machine today, `~/.codex/config.toml` trusts the main checkout's four hooks and none of its `.worktrees/`.

## What Changes

- When `.codex/hooks.json` satisfies doctor's Codex hook check, doctor reads `$CODEX_HOME/config.toml` (or `~/.codex/config.toml` when `CODEX_HOME` is unset). It then reports `codex_hook_trust` as `trusted`, `untrusted`, or `not checked`, with a reason.
- An untrusted checkout gets one **warning**, not a violation. The warning names the hook events missing a trust entry and this checkout's absolute path, and says to run `codex` here and approve the project hooks. If the Codex config is missing or unreadable, the report says `not checked` and no warning is added, so CI and machines without Codex stay quiet.
- `ateam init`'s `next_steps` names the one-time Codex trust step.
- `docs/adoption.md` names that step and says that each worktree needs its own approval.
- `openspec/specs/harness/spec.md` gains one Requirement, `Doctor reports Codex project hook trust`.

## Track

`design`: adds a spec Requirement and a doctor report key that adopters read. See `design.md`.

## Non-goals

- No bypass of Codex hook trust, and no write to `~/.codex/config.toml`.
- No check of `trusted_hash`. Which bytes the hash covers is unknown, so a hooks edit that revokes trust still reads as trusted (design D3).
- No change to the Claude, Gemini, or Antigravity rows. No trust check for user-scope `~/.codex/hooks.json`.
- No check on the hook path. The session-start hook stays within its 10 ms budget.

## Definition of Done

- Doctor tests with a temporary `CODEX_HOME` cover these cases: all four events trusted for this checkout (`trusted`, no warning); no entries (`untrusted`, one warning naming the four events); a partial set (the warning names only the missing events); entries for another path only, such as the main checkout seen from a worktree (`untrusted`); no config file (`not checked`, no warning); invalid TOML (`not checked`, no warning); and Codex hooks satisfied only at user scope or failing (`not checked`). `task mutate:check` reports `detection` for the path match, the per-event check, and the skip conditions.
- `task verify`, `ateam doctor --check`, and `ateam bench --selftest` are green. The change is archived before it's marked ready.
