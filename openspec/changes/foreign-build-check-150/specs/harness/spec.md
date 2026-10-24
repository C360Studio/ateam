## MODIFIED Requirements

### Requirement: Agent initialization and configuration parity

Initialization SHALL write usable canonical instructions, skill adapters, hook settings, and OpenSpec scaffolding. Parity checks SHALL reject missing AGENTS imports, instruction size overflow, mismatched rule row counts, copied adapter checklists, missing native hook configuration, placeholder workflow content, and documented hook event drift.

#### Scenario: Generated configuration

- **WHEN** a repository is initialized with default templates
- **THEN** its agent configuration passes parity checks

#### Scenario: Shipped review-pr skill

- **WHEN** a repository is initialized with default templates
- **THEN** `.agents/skills/review-pr/SKILL.md` is written with the same body this repository uses, with thin adapters at `.claude/skills/review-pr/SKILL.md` and `.gemini/commands/review-pr.toml`, and parity checks reject a canonical `review-pr` skill that loses its numbered steps or the record grammar markers `reviewed-by:` and `Reviewed at`

#### Scenario: Build provenance and duplicate server detection in doctor

- **WHEN** `ateam version`, `session-start`, or `ateam doctor` runs
- **THEN** build provenance (`vcs.revision` or `-ldflags` commit) is reported in `ateam version` and the `[ateam]` session-start line (or `unknown` when built without VCS metadata), and `ateam doctor` lists running `ateam serve` and `ateam mcp` processes (with their build commits) and warns when the binary has no build provenance (`binary has no build provenance; build with task build`), when the binary commit is behind checkout `HEAD` (`binary <build> is behind HEAD <head>; run task build`) or not an ancestor of checkout `HEAD`, when hook latency exceeds `5000 us` (`<5ms` budget), when a running `ateam mcp` process is stale, or when multiple `ateam serve` processes are running concurrently

#### Scenario: Stale or missing binary build provenance warning at session start

- **WHEN** `ateam hook --event session-start` runs in a git checkout with a resolved `HEAD` commit
- **THEN** it compares the embedded build commit (`state.BuildRevision()`) against `ReadFastGitHead` without spawning a git subprocess, emits `binary <build> is behind HEAD <head>; run task build` when the binary commit differs from `HEAD`, emits `binary has no build provenance; build with task build` when build provenance is `unknown` or `dev`, and emits neither line when the binary commit matches `HEAD`

#### Scenario: Build provenance checks only the ateam checkout

- **WHEN** the session-start hook or `ateam doctor` runs in a checkout whose root `go.mod` does not declare `module github.com/C360Studio/ateam` (or that has no `go.mod`)
- **THEN** neither compares the binary's build commit with the checkout's `HEAD`: the hook prints no `binary <build> is behind HEAD` line, doctor prints no not-an-ancestor warning and reports `build_provenance: not applicable (binary is not built from this checkout)`, and the module path is read from `go.mod` with a line scan, without spawning a subprocess
- **AND** in the ateam checkout both checks behave as before

