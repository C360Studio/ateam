# Proposal: queue-checks-416
Approved-by: Coby Leuschke (2026-11-26) sha256:3258493157fd

## Track

`design`: adds a harness Requirement, a new `ateam state queue-check` command adopters run, a new pickup-line field, and a doctor advice line.

## Decisions you are making by approving

- Every session start shows how many queue problems ateam found, so labels that drifted from the truth become visible without anyone asking.
- An issue marked blocked whose dependency has closed, or that names no dependency ateam can read, is reported.
- An issue still waiting for your decision after a decision is recorded on it is reported.
- An open issue that has no milestone, or is in a milestone but missing from its order, is reported, however new it is. Placing it stays a person's job, or the judge's in hold-my-beer mode (#417).
- A milestone holding more than a set number of open issues is reported, and so is an open milestone with no open issues left. The number is a team setting.
- Nothing is ever blocked: no merge, check or pickup fails because of a finding, and `ateam doctor` reports the count without failing.

## Why

ateam trusts queue labels and never checks them, so drift stays invisible until a person notices (#416). One adopter measured on 2026-11-26 found:

- two `status:blocked` issues whose blocker had closed three days earlier;
- three stale `status:needs-decision` labels;
- 15 of 25 milestoned issues in one milestone;
- five unmilestoned issues.

This repository showed the inverse drift on the same day. #418 added the `Order:`/`Depends on:` conventions these checks read.

## What Changes

1. `ateam state sync` also reads open milestones (title, open and closed issue counts) in one extra call per forge.
2. `ateam state queue-check` lists each finding with its issue or milestone and the rule it breaks. It exits 0 with or without findings, and 1 only without synced data.
3. The pickup line gains `queue=<n>` (or `queue=unavailable`), computed from the cache. `ateam state status` JSON carries the findings.
4. `ateam doctor` prints `queue: <n> finding(s); run ateam state queue-check` from the cache, and `--check` never fails on it. The preflight skill names the command before a handoff.
5. `.agents/ateam.json` gains `queue.max_milestone_issues` (default 15).

## Scope & Non-Goals

- **In-bounds:**
  - `internal/state`: milestones read, checks, summary line, settings;
  - `cmd/ateam/main.go`: subcommand and doctor line;
  - `openspec/specs/harness/spec.md`: one ADDED Requirement;
  - `.agents/skills/preflight/SKILL.md` and its template, `docs/daily-use.md`, `docs/adoption.md` § Team settings.
- **Non-goals:**
  - fixing labels automatically;
  - age thresholds (issue creation dates are not synced);
  - a scheduler (a host workflow can run the command);
  - a `Blocked by` grammar;
  - widening the decision-line grammar;
  - judge placement (#417);
  - `internal/review/` (a gate file).

## Definition of Done

- Table tests over cached cards and milestones cover each rule's hit and miss, plus `queue=unavailable` without a sync. A Gitea and a GitHub fixture fill the milestone counts. `task mutate:check` detects the removal of each rule.
- The session-start hook still reads only the cache, and `ateam doctor --check` makes no forge call.
- `task verify`, `ateam doctor --check` and `ateam bench --selftest` are green, and the change is archived before ready.
