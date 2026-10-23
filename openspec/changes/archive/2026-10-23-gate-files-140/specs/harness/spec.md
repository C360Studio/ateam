## MODIFIED Requirements

### Requirement: Merge gate and CI workflow enforcement

CI SHALL enforce the pinned verification workflow and aggregate `Required` check. The merge check SHALL require an active strict `main` ruleset, review by a not-weaker reviewer for non-draft code PRs, and no open known flakes unless the PR closes them.

#### Scenario: Complete workflow and merge-check contract

- **WHEN** the workflow and merge check are evaluated
- **THEN** `.github/workflows/ci.yml` SHALL trigger on `push` to `main` and `pull_request` (`opened`, `synchronize`, `reopened`, `ready_for_review`), pin every job to `ubuntu-24.04` with read-only permissions, run `Verify` (`15` minute timeout, `ateam doctor --check` and `task verify`) and `Merge check` (`scripts/merge-check.sh`), and gate merges on the aggregate `Required` job (`if: always()`, reading `join(needs.*.result, ' ')`). `scripts/merge-check.sh` (`task merge:check`) SHALL verify the active `main` ruleset requiring `Required` (`strict_required_status_checks_policy=true`), classify a changed file (and any `previous_filename` on rename) as a document only when its path ends in `.md` (outside `.claude/agents/` and `.agents/contracts/`) or starts with `openspec/` and ends in `.md`, `.yml`, `.yaml`, or `.json`, enforce review (`implemented-by:` and `reviewed-by:` each naming one of `claude`, `codex`, `gemini`, the reviewer not weaker by tier then band through `ateam review compare`, and different agents only when the policy sets `cross_provider`) on non-draft code pull requests, and block merges while any open `class:flake` issue is not closed by the pull request.

#### Scenario: Pinned CI workflow and Required aggregate

- **WHEN** `.github/workflows/ci.yml` is evaluated against the merge-gate contract
- **THEN** `pull_request` lists `opened`, `synchronize`, `reopened`, and `ready_for_review`, every job runs on `ubuntu-24.04` without write permissions, `merge-check` has only `contents: read`, `issues: read`, and `pull-requests: read`, and `required` needs `verify` and `merge-check` under `if: always()` and fails unless every needed job succeeded, and `concurrency.cancel-in-progress` is `${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}` so the run that marking a draft ready starts never cancels the in-flight run for the same head and leaves a failed `Required` on it

#### Scenario: Review strength and known-flake merge check

- **WHEN** `scripts/merge-check.sh` runs for a pull request or a `main` push
- **THEN** it checks the active `main` ruleset for `Required` (`strict_required_status_checks_policy=true`), treats any path under `openspec/` not ending in `.md`, `.yml`, `.yaml`, or `.json` (such as `.py`, `.sh`, `.go`, `.lock`, or executables) as code, requires non-draft code PRs to carry one `implemented-by:` and one `reviewed-by:` line each naming an agent (`claude`, `codex`, `gemini`) and its model in parentheses after the agent (several comma-separated on `implemented-by:`; a missing or blank model is a finding); once `.agents/review-policy.json` exists, each model is followed by its native effort (`claude (sonnet, medium, opus, high)`), the runs pass to `$ATEAM review compare` (default `ateam`; the CI job and `task merge:check` build it) whose findings, including a missing effort, are review findings and whose exit other than 0 or 1 is unavailable (exit 2), and the two lines must name different agents only when the policy sets `cross_provider: true`; verifies PR commit list completeness against the PR's `commits` count and `head.sha`, requires a PR comment review record (`reviewed-by: <agent>` or a `### Independent Cross-Agent Review` heading) designating a single full or short commit SHA in the PR's commit list, paginates commit file lists (failing closed on truncated pages or the 3,000-file ceiling) to allow later commits after the latest recorded commit only when each is a merge of `main` touching none of the PR's changed files or a documents-only commit (failing any other later commit with `review record at <sha> is older than content commit <sha>; request re-review`), and blocks merges when any open `class:flake` issue is not closed by the pull request

#### Scenario: Gate-file changes wait for the owner

- **WHEN** a pull request's changed files, or a rename's previous path, include `.agents/review-policy.json`, `scripts/merge-check.sh`, `scripts/gate-files.sh`, anything under `.github/workflows/`, or anything under `internal/review/`
- **THEN** the `Gate files` check (`.github/workflows/gate-files.yml` on `pull_request_target`, running `scripts/gate-files.sh` from the base branch with read-only permissions and never checking out or running pull request code) fails until the pull request carries the label `gate-change-approved`, and passes for a pull request that touches no gate file
- **AND** a failed read, a file list that is not a list of file objects, or one shorter than the pull request's `changed_files` (the listing stops at 3,000 files) exits 2 naming the read, never counted as no gate file
- **AND** the `Merge check` job checks out and builds from the base commit (`github.event.pull_request.base.sha`, or `github.sha` on a push), so the merge-check script, the review policy it reads, and the `ateam` binary are never taken from the pull request they judge
