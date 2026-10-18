# ATeam Agent & Platform Profiles

`.agents/` is the tracked, platform-neutral behavioral authority for Claude Code, Codex CLI, and Gemini CLI. Platform adapters under `.claude/`, `.codex/`, and `.gemini/` are thin pointers generated and verified by `ateam init` and `ateam doctor` (`TestAgentSetupParity`).

## Universal Matrix (Claude Code, Codex CLI, Gemini CLI)

| Capability | Claude Code | Codex CLI | Gemini CLI |
| --- | --- | --- | --- |
| Instruction File | `CLAUDE.md` (`@AGENTS.md`) | `AGENTS.md` | `.gemini/settings.json` (`contextFileName: "AGENTS.md"`) |
| Canonical Skills | `.agents/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` |
| Skill Adapters | `.claude/skills/<name>/SKILL.md` | Native `.agents/skills/` | `.gemini/commands/<name>.toml` |
| Hook Config | `.claude/settings.json` | `.codex/hooks.json` | `.gemini/settings.json` |
| Session Start | `SessionStart` -> `ateam hook --event session-start` | `SessionStart` -> `ateam hook --event session-start` | `SessionStart` -> `ateam hook --event session-start` |
| Before Prompt | `UserPromptSubmit` -> `ateam hook --event before-prompt` | `UserPromptSubmit` -> `ateam hook --event before-prompt` | `BeforeAgent` -> `ateam hook --event before-prompt` |
| Pre-Compaction | `PreCompact` -> `ateam hook --event pre-compact` | `PreCompact` -> `ateam hook --event pre-compact` | `PreCompress` -> `ateam hook --event pre-compact` |
| Turn End | `Stop` -> `ateam hook --event turn-end` | `Stop` -> `ateam hook --event turn-end` | `AfterAgent` -> `ateam hook --event turn-end` |

## Shared Skills

Canonical skills live in `.agents/skills/<name>/SKILL.md`:

- [pickup](skills/pickup/SKILL.md): reconcile live git/worktree, GitHub issues/PRs, `Hold:` queue, and cached `.ateam/state.json` checkpoint.
- [handoff](skills/handoff/SKILL.md): record a structured 5-field checkpoint (`Goal`, `Addresses`, `Constraints`, `Done when`, `Checks to run`) via `ateam_state` / `ateam state handoff`.
- [preflight](skills/preflight/SKILL.md): select and execute `Taskfile.yml` verification gates and record evidence.
- [ponytail](skills/ponytail/SKILL.md): apply the Ponytail 6-step decision ladder (`lite`, `full`, `ultra`, `off`), diff review, repo audit, and `// shortcut:` debt tracking.

## Parity Verification (`ateam doctor` / `ateam init --check`)

1. Confirm `AGENTS.md`, `CLAUDE.md`, `.agents/README.md`, `.agents/protocol.md`, and `.agents/rules-index.md` exist and are non-empty.
2. Confirm `CLAUDE.md` imports `@AGENTS.md`, stays within 10 lines, and carries no section headings or rule tables of its own.
3. Confirm `AGENTS.md` stays strictly under 24,000 bytes and matches `.agents/rules-index.md` in numbered rule sections and row counts.
4. Confirm every canonical skill in `.agents/skills/<name>/SKILL.md` has a matching thin Claude adapter in `.claude/skills/<name>/SKILL.md` and Gemini command in `.gemini/commands/<name>.toml`.
5. Confirm `.claude/settings.json`, `.codex/hooks.json`, and `.gemini/settings.json` wire all four normalized lifecycle hook events (`session-start`, `before-prompt`, `pre-compact`, `turn-end`) and the `ateam mcp` server.
