# ATeam (`ateam`)

Experimental universal agentic harness, CLI, low-tax MCP server, Ponytail benchmark suite, and embedded SvelteKit Project Management board for **Claude Code**, **Codex CLI**, and **Gemini CLI**, with first-class **OpenSpec** (`openspec/`), **GitHub**, and **Gitea** support.

## Highlights

- **Universal Agent Governance (`AGENTS.md` + `.agents/`):** Single canonical workflow protocol (`pickup`, `handoff`, `preflight`, `ponytail`) with thin platform adapters in `.claude/`, `.codex/`, and `.gemini/` verified by `ateam doctor`.
- **Ponytail Decision Ladder:** Enforces minimum-complete-change discipline, tracks intentional deferrals via `// shortcut: <limit>, <when to upgrade>`, and requires a 2-line skipped/residual-risk exit summary.
- **Zero-Dependency Go Binary (`ateam`):**
  - `ateam hook <event>` — `<5ms` normalized lifecycle hook (`session-start`, `before-prompt`, `pre-compact`, `turn-end`)
  - `ateam mcp` — 2-tool stdio MCP server (`ateam_state`, `ateam_task`) under `<300` schema tokens
  - `ateam state` — Hybrid state management (`status`, `pickup`, `handoff`, `holds`, `debt`, `spec-queue`, `spec-check`, `sync`, `mode`)
  - `ateam init` / `ateam doctor` — Repo bootstrap (`--github`, `--gitea` label setup) and multi-agent parity checks
  - `ateam bench` — Offline self-test (`--selftest`) and artifact rescoring (`--rescore`) across Coding Discipline + Handoff/Pickup Continuity tiers
  - `ateam serve` — Embedded Svelte 5 + SvelteKit (`adapter-static`) Project Management UI (Kanban Board + Table view + OpenSpec queue + Shortcut Debt + Benchmark drawer)

## Quick Start

```bash
task verify         # Run all 4 Definition of Done gates
task build          # Build static SvelteKit UI + ./bin/ateam binary
./bin/ateam doctor  # Check multi-agent parity, MCP budget, and hook latency
./bin/ateam serve   # Launch embedded PM Board at http://localhost:8420
```
