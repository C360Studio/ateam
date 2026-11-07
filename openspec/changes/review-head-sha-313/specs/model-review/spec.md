## MODIFIED Requirements

### Requirement: Review request spec
`ateam review request <pr>` SHALL print one JSON object naming the PR, its head sha, the prompt `/review-pr <n>`, the implementer runs, the eligible reviewer runs, the detected host, `return_to`, and a `launch` field of `manual`, `printed`, or `executed`. It SHALL execute a host manifest `launch` command only when `review_launch` is `agent`, as an argument array without a shell.

#### Scenario: Eligible runs
- **WHEN** the policy ranks the implementers
- **THEN** `eligible` lists each model whose tier is not below every implementer's, from another provider when `cross_provider` is set, at the highest implementer band

#### Scenario: Frontier implementer under cross-provider
- **WHEN** the PR body says `implemented-by: claude (opus, high)` and the policy sets `cross_provider: true`
- **THEN** `eligible` lists only frontier models of codex and gemini at the `high` band

#### Scenario: Manual never executes
- **WHEN** `review_launch` is `manual` and a manifest with `launch` exists
- **THEN** the object is printed with `launch: manual` and no command runs

#### Scenario: Agent executes the manifest
- **WHEN** `review_launch` is `agent` and the detected host's manifest has `launch`
- **THEN** the expanded argument array runs without a shell and the object says `launch: executed`

#### Scenario: Offline without implementers
- **WHEN** no `--implementer` flag is given and the PR body cannot be fetched
- **THEN** the command exits 2 naming the flag

#### Scenario: Reviewer the gate would reject
- **WHEN** a host manifest exists and the `--reviewer` run, or the first eligible run when none is given, shares an implementer's provider under `cross_provider`, fails the comparison against an implementer, or does not exist
- **THEN** the command exits 2 with the reason, runs no command and prints no tool text

#### Scenario: Launch without a way back
- **WHEN** a host manifest's `launch` tool or command names `{return_to}` and neither `--return-to` nor the host's environment supplies one
- **THEN** the command exits 2 naming `--return-to`, runs no command and prints no tool text
- **AND** with `--dry-run` it prints the same line as a warning and exits 0

#### Scenario: Head from the PR
- **WHEN** `--implementer` is given and the command runs from a checkout whose `HEAD` is not the PR's head
- **THEN** `head_sha` is the PR's head read from the forge
- **AND** when the PR cannot be read the command exits 2 naming its head and prints nothing
