## ADDED Requirements

### Requirement: Merge check on pull_request_target

On GitHub Actions, `ateam review gate` SHALL treat `pull_request_target` as a pull-request run and apply the same rules as `pull_request`, including the awaiting probe. On Gitea Actions it SHALL stop with exit 1 naming why, as any other unhandled event does.

#### Scenario: Base-branch trigger on GitHub

- **WHEN** `GITHUB_ACTIONS=true` and `GITHUB_EVENT_NAME=pull_request_target` on a GitHub repository
- **THEN** the gate reads the pull request and applies the review and known-flake checks as for `pull_request`

#### Scenario: Base-branch trigger on Gitea

- **WHEN** the same event is set on a Gitea repository
- **THEN** the gate exits 1 and says `pull_request_target` is not supported on Gitea
