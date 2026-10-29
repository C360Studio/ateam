## ADDED Requirements

### Requirement: Merge check runs on pull-request state events

`Merge check` SHALL run in its own workflow (`.github/workflows/merge-check.yml` and its Gitea twin), separate from `CI`, on the pull-request events that change its inputs, and SHALL be a required context of its own beside `Required` and `Gate files`. `Required` SHALL need only `Verify`.

#### Scenario: The reviewer's description edit re-runs the merge check

- **WHEN** a ready code pull request's head carries a review record comment and the reviewer then edits the description to add `reviewed-by:`
- **THEN** the `edited` event runs `Merge check` against the head with no manual rerun, and it passes

#### Scenario: An open flake issue does not fail Required

- **WHEN** an issue labelled `class:flake` is open and a pull request that does not close it gets a green `Verify`
- **THEN** `Required` passes and `Merge check` fails, naming the open issue

#### Scenario: Pinned merge-check workflow

- **WHEN** `.github/workflows/merge-check.yml` is evaluated against the merge-gate contract
- **THEN** it triggers only on `pull_request` (`opened`, `synchronize`, `reopened`, `ready_for_review`, `edited`), runs one queued run per pull request (`concurrency.group` per pull request number, `cancel-in-progress: false`, so no cancelled `Merge check` sits beside a pass at one head), has the one job `merge-check` on `ubuntu-24.04` with only `contents: read`, `issues: read`, and `pull-requests: read`, and runs `scripts/merge-check.sh "$PR_NUMBER"` with `ATEAM_REQUIRED_CHECKS: Required,Merge check`

#### Scenario: A draft pull request carries no Merge check

- **WHEN** a draft pull request is opened, pushed, or edited
- **THEN** the merge-check job is skipped under a name that is not a required context, as for `Verify` on a draft

## MODIFIED Requirements

### Requirement: Merge gate and CI workflow enforcement

CI SHALL enforce the pinned verification workflow and aggregate `Required` check, and `merge-check.yml` the `Merge check`. The merge check SHALL require an active strict `main` ruleset (on Gitea, branch protection requiring `CI / Required (pull_request)` with `block_on_outdated_branch`) requiring each check in `ATEAM_REQUIRED_CHECKS` when set, review by a not-weaker reviewer for non-draft code PRs, and no open known flakes unless the PR closes them, on GitHub and on Gitea.

#### Scenario: Complete workflow and merge-check contract

- **WHEN** the workflow and merge check are evaluated
- **THEN** `.github/workflows/ci.yml` SHALL trigger on `push` to `main` and `pull_request` (`opened`, `synchronize`, `reopened`, `ready_for_review`), pin every job to `ubuntu-24.04` with read-only permissions, run `Verify` (`15` minute timeout, `ateam doctor --check` and `task verify`), and gate merges on the aggregate `Required` job beside `Merge check` (`scripts/merge-check.sh`, in `.github/workflows/merge-check.yml`) (`if: always()`, reading `join(needs.*.result, ' ')`). `scripts/merge-check.sh` (`task merge:check`) SHALL verify the active `main` ruleset requiring `Required`, or each check `ATEAM_REQUIRED_CHECKS` lists (`strict_required_status_checks_policy=true`), classify a changed file (and any `previous_filename` on rename) as a document only when its path ends in `.md` (outside `.claude/agents/` and `.agents/contracts/`) or starts with `openspec/` and ends in `.md`, `.yml`, `.yaml`, or `.json`, enforce review (`implemented-by:` and `reviewed-by:` each naming one of `claude`, `codex`, `gemini`, the reviewer not weaker by tier then band through `ateam review compare`, and different agents only when the policy sets `cross_provider`) on non-draft code pull requests, and block merges while any open `class:flake` issue is not closed by the pull request.

#### Scenario: Pinned CI workflow and Required aggregate

- **WHEN** `.github/workflows/ci.yml` is evaluated against the merge-gate contract
- **THEN** `pull_request` lists `opened`, `synchronize`, `reopened`, and `ready_for_review`, every job runs on `ubuntu-24.04` without write permissions, there is no `merge-check` job, `verify` runs under `if: github.event_name != 'pull_request' || !github.event.pull_request.draft`, `required` needs `verify` under `if: always() && (github.event_name != 'pull_request' || !github.event.pull_request.draft)` and fails unless every needed job succeeded, each of the two is named `${{ github.event.pull_request.draft && '<name> (draft, skipped)' || '<name>' }}` for its check name `Verify` or `Required`, and `concurrency.cancel-in-progress` is `${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}` so the run that marking a draft ready starts never cancels the in-flight run for the same head and leaves a failed `Required` on it

#### Scenario: Review strength and known-flake merge check

- **WHEN** `scripts/merge-check.sh` runs for a pull request or a `main` push
- **THEN** it checks the active `main` ruleset for `Required` (`strict_required_status_checks_policy=true`), treats any path under `openspec/` not ending in `.md`, `.yml`, `.yaml`, or `.json` (such as `.py`, `.sh`, `.go`, `.lock`, or executables) as code, requires non-draft code PRs to carry one `implemented-by:` and one `reviewed-by:` line each naming an agent (`claude`, `codex`, `gemini`) and its model in parentheses after the agent (several comma-separated on `implemented-by:`; a missing or blank model is a finding); once `.agents/review-policy.json` exists, each model is followed by its native effort (`claude (sonnet, medium, opus, high)`), the runs pass to `$ATEAM review compare` (default `ateam`; the CI job and `task merge:check` build it) whose findings, including a missing effort, are review findings and whose exit other than 0 or 1 is unavailable (exit 2), and the two lines must name different agents only when the policy sets `cross_provider: true`; verifies PR commit list completeness against the PR's `commits` count and `head.sha`, requires a PR comment review record (`reviewed-by: <agent>` or a `### Independent Cross-Agent Review` heading) designating a single full or short commit SHA in the PR's commit list, paginates commit file lists (failing closed on truncated pages or the 3,000-file ceiling) to allow later commits after the latest recorded commit only when each is a merge of `main` touching none of the PR's changed files or a documents-only commit (failing any other later commit with `review record at <sha> is older than content commit <sha>; request re-review`), and blocks merges when any open `class:flake` issue is not closed by the pull request

#### Scenario: Gate-file changes wait for the owner

- **WHEN** a pull request's changed files, or a rename's previous path, include `.agents/review-policy.json`, `scripts/merge-check.sh`, `scripts/gate-files.sh`, anything under `.github/workflows/` or `.gitea/workflows/`, or anything under `internal/review/`
- **THEN** the `Gate files` check (`.github/workflows/gate-files.yml` on `pull_request_target`, running `scripts/gate-files.sh` from the base branch with read-only permissions and never checking out or running pull request code) fails until the pull request carries the label `gate-change-approved`, and passes for a pull request that touches no gate file
- **AND** a failed read, a file list that is not a list of file objects, or one shorter than the pull request's `changed_files` (the listing stops at 3,000 files) exits 2 naming the read, never counted as no gate file
- **AND** the same holds on Gitea for `.gitea/workflows/gate-files.yml`, which runs `scripts/gate-files.sh` with `ATEAM_FORGE=gitea` to read Gitea's API with `curl`, every file page until an empty one, and the labels from the pull request
- **AND** the `Merge check` job checks out and builds from the base commit (`github.event.pull_request.base.sha`), so the merge-check script, the review policy it reads, and the `ateam` binary are never taken from the pull request they judge

#### Scenario: Merge check on either forge

- **WHEN** `scripts/merge-check.sh` runs
- **THEN** it runs `$ATEAM review gate` (default `ateam`), which picks the forge from `ATEAM_FORGE`, else under Actions from `GITHUB_SERVER_URL` (`https://github.com` is GitHub, any other root is Gitea) with `GITHUB_REPOSITORY`, else from `origin`, else GitHub through `gh`, and prints `merge-check: forge=<forge> repo=<owner>/<repo> (from <source>)` without the host
- **AND** GitHub is read through `gh` with the argv the shell script used, and Gitea through API v1 with `GITEA_TOKEN`, with the same review and known-flake lines on both
- **AND** `.gitea/workflows/merge-check.yml` is `.github/workflows/merge-check.yml` with only the merge-check step's credentials changed, to `GITEA_TOKEN: ${{ github.token }}` (no extra secret), and its `ATEAM_REQUIRED_CHECKS` to the Gitea contexts

#### Scenario: Gitea up-to-date rule

- **WHEN** the merge check reads Gitea
- **THEN** the up-to-date rule holds only when `main` is `protected`, the branch protection in force on it (`effective_branch_protection_name`) has `enable_status_check` and lists exactly `CI / Required (pull_request)`, or each context `ATEAM_REQUIRED_CHECKS` lists, in `status_check_contexts` (no glob), and, when the rule can be read, has `block_on_outdated_branch`; otherwise it fails like a GitHub ruleset without the strict setting
- **AND** when reading the rule answers 403 or 404 (the Gitea Actions job token, the only token the CI job uses), the branch's `protected`, `enable_status_check` and `status_check_contexts` decide and the run prints `merge-check: rule strict=unverified (branch protection needs a repo-admin token; the job token can see protected, status check and context only)`
- **AND** any other failed read of the branch or the rule (401, 5xx, a wrong shape) exits 2, never passes

#### Scenario: Gitea review evidence

- **WHEN** the merge check reads a Gitea pull request
- **THEN** its file and commit lists are read page by page and must match `X-Total-Count` (and the files `changed_files`), the commit list is taken oldest first and must end at `head.sha`, and every commit after the review record is a finding needing a new review until a live run shows Gitea's commit file lists are complete

#### Scenario: Gitea closing references

- **WHEN** an open `class:flake` issue exists on Gitea
- **THEN** the label is read from the repository's labels first (Gitea discards an unknown label in an issue filter, so a missing label never reads as an empty list), the open issues are filtered for the label client side, and a flake counts as closed only when its timeline holds a `ref_action` of `closes` from this pull request in the same repository; the description is not parsed

#### Scenario: Draft pull requests defer CI and ready revisions run every gate

- **WHEN** a `pull_request` or `pull_request_target` event arrives on GitHub for a draft pull request
- **THEN** `Verify`, `Merge check`, `Required`, and `Gate files` do not run (`if: github.event_name != 'pull_request' || !github.event.pull_request.draft`, with `Required` as `always() && (...)`, `Merge check`, whose event is always `pull_request`, as `if: ${{ !github.event.pull_request.draft }}`, and `Gate files`, whose event is always `pull_request_target`, as `if: ${{ !github.event.pull_request.draft || github.event.action == 'unlabeled' }}`, so removing a label on a draft still replaces an earlier `Gate files` pass at the same head), and each is renamed on a draft (`name: ${{ github.event.pull_request.draft && '<job> (draft, skipped)' || '<job>' }}`) so no check run named `Required`, `Merge check`, or `Gate files` exists at a draft head and the required contexts stay unreported; GitHub does not evaluate a skipped job's `name:`, so the draft's skipped check run carries the raw expression text, which is also not a required name
- **AND** a non-draft `opened` or `reopened`, a `ready_for_review`, and every `synchronize` while the pull request is not a draft run `Verify`, `Merge check`, `Required`, and `Gate files` against the head revision under their required names, and `push` to `main` still runs `Verify`
- **AND** whenever `Required` runs for a non-draft revision it fails on any needed result other than `success`, including `skipped`
- **AND** on Gitea, `.gitea/workflows/ci.yml` adds `edited` to its `pull_request` types and skips `Verify` on a WIP pull request under the same guard, as `.gitea/workflows/merge-check.yml` skips `Merge check`, with `required` under `if: always() && (github.event_name != 'pull_request' || !github.event.pull_request.draft)` and each skipped job renamed `<name> (draft, skipped)` as on GitHub, so a rerun of a WIP run, which re-evaluates its saved `draft=true` event, never posts under a required context; `.gitea/workflows/gate-files.yml` adds `edited`, runs under `if: ${{ !github.event.pull_request.draft || github.event.action == 'label_updated' || github.event.action == 'label_cleared' }}`, and keeps the name `Gate files` for those two actions; the existing `concurrency` rule cancels an in-flight WIP run when the pull request leaves WIP; and this needs Gitea 28.x, as `docs/adoption.md` states, because older Gitea evaluates neither `name:` nor the compound `always()`
