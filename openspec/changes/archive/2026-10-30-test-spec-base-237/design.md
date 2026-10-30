# Design: `test-spec-base-237`

## Context

Opened in this session at `main` d7bb142 (v0.3.4).

- `internal/testspec/check.go:48` `Check(ctx, root)` walks the whole tree (`:59-104`), scans Vitest through `node` (`:105-123`), then judges every declaration (`:126-180`). `missing spec citation` is appended at `:178`; malformed, unresolved, and parse errors come from the same loop and from `declaration.Error`.
- `check.go:194-235` `scanGo` records `fset.Position(fn.Pos()).Line` and the attached doc comments; it keeps no end line. `internal/testspec/vitest.mjs:96` pushes `{file, line, name, comments}`; no end line either.
- Nothing in `internal/testspec` runs `git` (`grep exec.Command`); the only subprocess is `node`. The package is internal, so the adopter surface is the CLI, not the Go signature.
- `cmd/ateam/main.go:667-683` `runTestSpec` refuses any argument (exit 2), exits 1 on diagnostics; help line at `:94`.
- `Taskfile.yml:74-78` `test:spec` runs `go run ./cmd/ateam test-spec`; `.github/workflows/ci.yml:32` checks out with the default depth 1 and `:59` runs `task verify`.
- Spec: `openspec/specs/test-attribution/spec.md:55-76` "Deterministic discovery and verification" fixes exit codes 0/1/2 and offline discovery; it does not mention arguments. `openspec/specs/harness/spec.md:370` requires the gate in `task verify` only for this repository.
- Docs: `.agents/protocol.md:67-78` § Test-to-spec attribution; `docs/adoption.md:13`; `docs/brownfield.md` (118 lines) has no `test-spec` section.
- Adopter data (#237): SemTeams `059c2ae7` has 4,732 uncited tests (843 Go), 11 capabilities, 3 citations in an older form.

## Goals / Non-Goals

Goal: an adopter gates every new or edited test from its first PR without backfilling legacy ones. Non-goals: `proposal.md`.

## Decisions

### D1. Mechanism

- **A (recommended). `--base <ref>` diff scope only.** No repo state; the scope is what the PR changed. Cost: one flag, a `git` subprocess, end lines from both scanners. Forecloses nothing: a baseline can be added later.
- **B. Committed baseline only.** Works on full-repo runs and shallow clones, but adds a 4,732-line file that must only shrink and its own write/prune rules.
- **C. Both now.** Twice the surface before one adopter has tried either.

Adopter seam: nothing changes until a repository passes `--base`.

### D2. What "added or changed" means

- **A. File scope:** every test in a changed file. Simple, but one edit in a 300-test file forces 300 citations; the backfill wall returns file by file.
- **B (recommended). Declaration span meets a changed line.** Span = first attached comment line through the declaration's last line (`fn.End()`, `node.getEnd()`). A deletion-only hunk counts at its new-side position, so stripping a citation from a legacy test is a change. Cost: two end-line fields and a `-U0` hunk parser (a parser, so it gets fixture tests).
- **C. Names absent at base.** Needs a scan of the base tree; misses edits to legacy tests.

Adopter seam: touching a legacy test obliges citing it; untouched ones never fail.

### D3. Which diff

- **A. `git diff <merge-base>...HEAD`.** Committed changes only; a local run before commit misses new tests.
- **B (recommended). Merge base to the working tree, plus untracked files.** `git diff -U0 --no-ext-diff <merge-base>` and `git ls-files --others --exclude-standard` (every line of an untracked supported test is new). Same result as A on CI's clean tree, and a local run matches what the PR will show. Ignored files are out of scope, as the full walk's excluded directories already are.

### D4. Which diagnostics the scope filters

- **A (recommended). Only `missing spec citation`.** Malformed, dangling, unsafe citations and parse errors still fail anywhere: a spec rename that orphans a legacy citation is drift this PR caused. SemTeams' 3 old-form citations would fail on the first run and need fixing once.
- **B. Every diagnostic.** Fully quiet legacy, but a renamed requirement goes unnoticed until someone edits the citing test.

### D5. Go signature

- **A (recommended).** `Check(ctx, root, base string)`; `""` is today's full scope. Two callers (`cmd/ateam/main.go:672`, `check_test.go`).
- **B.** An `Options` struct. Nicer if the baseline lands; premature now.

### D6. Unresolvable base

Exit 2, no diagnostics, message naming the ref and `git fetch origin <branch>` / `fetch-depth: 0`. Never fall back to full scope (fails every legacy test) or empty scope (passes everything): a silently empty scope is a gate that always passes.

### D7. Adopter path

`docs/brownfield.md` gains "Gate new tests first": PR CI runs `ateam test-spec --base origin/${{ github.base_ref }}` after `actions/checkout` with `fetch-depth: 0`; a full `ateam test-spec` joins `task verify` once the backfill reaches zero. This repository keeps its full run.

## Risks / Trade-offs

- `--base` trusts the ref: `--base HEAD` scopes to uncommitted work only. Documented; the adopter's CI chooses the ref.
- A test moved between files with rename detection shows no hunks and stays out of scope; a copy into a new file is in scope. Acceptable: moves are not new tests.
- `git` becomes a runtime dependency only when `--base` is passed; the full run stays offline and git-free.

## Migration Plan

None for this repository. Adopters opt in per D7; without the flag output is byte-identical.

## Open Questions

1. **Owner ruling:** D4 A — keep failing malformed and dangling citations on untouched legacy tests (SemTeams fixes 3 on first run)?
2. **Owner ruling:** D3 B — include uncommitted and untracked changes, or committed-only (A)?
3. **Owner ruling:** D1 A — confirm the baseline file stays deferred until SemTeams asks after trying `--base`.
