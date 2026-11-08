# Design: `codex-hook-trust-311`

## Context

These references were opened in this session on `claude/codex-hook-trust-311` at `main` a53369f. Evidence for the cause is in the #311 body and the owner's comment of 2026-11-07.

- `internal/bootstrap/bootstrap.go:739-740`: the Codex row checks `.codex/hooks.json`, falling back to `~/.codex/hooks.json` under `ATEAM_HOOK_SCOPE=user`, and uses the native events in `claudeHookEvents` (`:751`): `SessionStart`, `UserPromptSubmit`, `PreCompact`, and `Stop`. `HookSources` (`:755`) names the file that satisfied each client's check. Doctor checks the file and never asks whether Codex loads it.
- `internal/bootstrap/mcp.go:11` already imports `github.com/pelletier/go-toml/v2` to read `.codex/config.toml`, so the change needs no new dependency.
- `~/.codex/config.toml` on this machine, read in this session, has one table per trusted hook: `[hooks.state."/Users/coby/Code/c360/ateam/.codex/hooks.json:session_start:0:0"]` with `trusted_hash = "sha256:…"`, plus the same for `user_prompt_submit`, `pre_compact`, and `stop`. `grep -c .worktrees` returns 0, so none of the linked worktrees are trusted. That includes this one, where Codex would run no hooks.
- `cmd/ateam/main.go:600-620`: doctor's warnings are built up and reported. Warnings never fail `--check`; violations do (`:622`). The `pin` key (`:641`) is the precedent for a new report key.
- `cmd/ateam/main.go:556`: `ateam init` prints a single `next_steps` string.
- `docs/adoption.md:36` already ends with "Codex loads project hooks only from a trusted `.codex/` layer." The T3 row (`:165`) says that T3 doesn't bypass hook trust.
- `internal/bench/codex.go:91` sets `CODEX_HOME` for benchmark runs, so Codex honors that variable as its config home.

## Goals / Non-Goals

Goal: a checkout whose Codex project hooks aren't trusted is flagged visibly by `ateam doctor`, and `init` and the docs name the one-time step. Non-goals are in `proposal.md`.

## Decisions

**D1. How untrusted hooks are detected.**
- A) **Recommended:** read the Codex config's `hooks.state` keys and match the prefix `<abs root>/.codex/hooks.json:<event>:`, where `<event>` is the snake_case form of each native event. This costs about 50 lines and a table test. The blast radius is doctor only. It forecloses nothing. It depends on an undocumented Codex key format, but the format is stable across the four entries on this machine, and if Codex changes it the check fails loud (`untrusted`) with a warning, never silently green.
- B) A liveness signal: warn when no `--agent codex` hook has ever written `.ateam/state.json`. It works without reading Codex's config, but state keeps only the latest writer, so a checkout used by Claude would read as "Codex never fired". It's also wrong on every machine that never runs Codex. Rejected.
- C) Only the `init` note and docs. It costs almost nothing but detects nothing, and #311's Definition of Done asks for a flag. This is kept as D5, alongside A.
- Seam: an adopter who never runs Codex sees `not checked` and nothing else. A Codex user sees one warning per untrusted checkout until they approve it.

**D2. Severity.**
- A) **Recommended:** a warning. CI has no Codex config, and a warning cannot block a merge.
- B) A violation when a Codex config exists. A developer couldn't pass `--check` in a fresh worktree until they had opened Codex there, which is the Claude-only user's case. See Open Question 1.

**D3. Hash.**
- A) **Recommended:** check that the key is present only. The bytes `trusted_hash` covers are unknown, so a hooks edit that revoked trust still reads `trusted`. A `// shortcut:` marks this limit.
- B) Recompute the hash. That needs Codex's canonical form, which isn't documented. Rejected until it is.

**D4. Path form.** Use `filepath.Abs(root)` without `EvalSymlinks`. Codex wrote `/Users/…` here, and on macOS `/Users` isn't a symlink. A checkout reached through a symlink would read as `untrusted`, which errs toward the warning. See Open Question 2.

**D5. Init and docs.** Append one sentence to `next_steps`: "Codex runs project hooks only after you run `codex` in each checkout or worktree and approve them; `ateam doctor` warns until you do." `docs/adoption.md:36` and `:165` say the same.

## Risks / Trade-offs

- The check depends on Codex's undocumented `hooks.state` key layout. If the layout changes, doctor warns `untrusted` on trusted checkouts until the check is updated.
- Every new `.worktrees/` checkout warns on a Codex user's machine. That's the real state, and it's the main finding of #311's follow-up.

## Migration Plan

None. The change only adds a report key and at most one warning.

## Open Questions

1. **Owner ruling:** should an untrusted checkout be a warning (D2 A, recommended) or a violation when a Codex config exists (D2 B)?
2. **Owner ruling:** is it acceptable for a checkout reached through a symlink to read `untrusted` (D4), or should doctor also try the `EvalSymlinks` path? Recommended: accept it, and add the second lookup only if someone reports the case.
