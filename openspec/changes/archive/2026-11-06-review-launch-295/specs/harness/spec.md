## ADDED Requirements

### Requirement: Host launch manifest
When `.agents/hosts/<host>.toml` exists for the detected host, ateam SHALL read its `launch` and `notify` argument arrays and substitute the launch placeholders inside each argument. A missing manifest SHALL change no output other than `launch: printed`; a malformed one SHALL fail with fixed text that does not echo the file.

#### Scenario: Detection order
- **WHEN** no `ATEAM_HOST` is set
- **THEN** the host is `t3code` when `T3CODE_THREAD_ID` is set, else `herdr` when `HERDR_ENV=1`, else `enjoy` when `ENJOY_MCP_ENDPOINT` is set, else `cli`

#### Scenario: Placeholders
- **WHEN** an argument holds `{pr}`, `{prompt}`, `{sha}`, `{agent}`, `{model}`, `{effort}`, or `{return_to}`
- **THEN** each is replaced by the launch spec's value

#### Scenario: Override wins
- **WHEN** `ATEAM_HOST=herdr` and `T3CODE_THREAD_ID` are both set
- **THEN** the detected host is `herdr`

#### Scenario: Placeholder inside an argument
- **WHEN** a `launch` argument is `--prompt={prompt}`
- **THEN** the argument becomes `--prompt=/review-pr <n>` as one argument, with no shell splitting

#### Scenario: Malformed manifest
- **WHEN** the manifest is not valid TOML or `launch` is not an array of strings
- **THEN** `ateam review request` exits non-zero with a fixed-text reason

### Requirement: Review record in the forge cache and pickup
`ateam state sync` SHALL fetch the comments of open pull requests whose head branch has a local worktree, inside the existing comment budget, and keep the newest review record's sha, verdict, agent, model and effort on the PR card. Pickup and session-start SHALL print that record on the worktree's line from the cache alone, with no network call.

#### Scenario: Record line
- **WHEN** the cache holds a record for the worktree's PR
- **THEN** the worktree line ends with `review record at <sha>, <verdict>, <agent> (<model>, <effort>)`

#### Scenario: Record present
- **WHEN** the cache holds a record whose sha equals the worktree head
- **THEN** the worktree line ends with the record and no `stale`

#### Scenario: Record stale
- **WHEN** the worktree head is not the record's sha
- **THEN** the worktree line ends with the record and `stale`

#### Scenario: No record or budget exhausted
- **WHEN** the PR has no record comment, or the comment fetch exceeded the budget
- **THEN** the worktree line is byte-for-byte today's and sync does not fail
