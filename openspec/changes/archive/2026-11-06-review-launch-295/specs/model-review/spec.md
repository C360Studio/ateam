## ADDED Requirements

### Requirement: Review launch policy
`.agents/review-policy.json` SHALL accept an optional `review_launch` of `manual` (the default when absent) or `agent` and reject other values. Under `manual` the implementing session SHALL stop at ready-for-review and SHALL NOT start the reviewer. Under `agent` the implementing session MAY start the reviewer through its host. The key SHALL NOT change the meaning of `cross_provider`.

#### Scenario: Default is manual
- **WHEN** the policy has no `review_launch`
- **THEN** it parses as `manual`, `ateam doctor --check` reports no review violation, and `ateam review request` never executes a command

#### Scenario: Unknown launch value
- **WHEN** `review_launch` is any value other than `manual` or `agent`
- **THEN** policy validation rejects it naming the two accepted values

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

### Requirement: Review notify text
`ateam review notify` SHALL send only the fixed text `review record for PR #<n> landed at <sha>; run ateam state pickup` to the `return_to` of the launch spec through the host manifest's `notify` command, SHALL print the host tool to call when the host has no `notify`, and SHALL send nothing on the `cli` host. The record body SHALL NOT enter the message.

#### Scenario: Hostile record body
- **WHEN** the review record comment carries shell text and the different-writer warning line
- **THEN** the message sent is the fixed text and nothing from the comment
