## ADDED Requirements

### Requirement: Forge token roles

`ateam` SHALL authenticate forge reads (board sync, pickup, review evidence) with `ATEAM_TOKEN_READ` and forge writes (PR comments, label bootstrap) with `ATEAM_TOKEN_WRITE`, each falling back to the other when only one is set, and SHALL keep the existing Gitea (`GITEA_TOKEN`, then `ATEAM_GITEA_TOKEN`) and GitHub (`GH_TOKEN`, `GITHUB_TOKEN`, then `gh auth`) resolution when neither is set. Values are trimmed; a whitespace-only value counts as unset.

#### Scenario: Separate read and write tokens

- **WHEN** `ATEAM_TOKEN_READ` and `ATEAM_TOKEN_WRITE` hold different tokens
- **THEN** Gitea board sync sends the read token and a Gitea PR handoff comment or label bootstrap sends the write token, `gh` list calls run with the read token as `GH_TOKEN` and `gh pr comment` / `gh label create` with the write token, and review evidence uses the read token

#### Scenario: One role token serves both roles

- **WHEN** only one of `ATEAM_TOKEN_READ` and `ATEAM_TOKEN_WRITE` is set
- **THEN** that token authenticates both reads and writes, ahead of `GITEA_TOKEN` or `GH_TOKEN`

#### Scenario: Doctor warns on a shared token

- **WHEN** `ateam doctor` runs and reads and writes resolve to the same token (neither role variable set, only one set, or both set to one value)
- **THEN** it warns `forge reads and writes share one token; set ATEAM_TOKEN_READ and ATEAM_TOKEN_WRITE to separate tokens scoped to each role`, prints no token value, and `--check` still passes on that warning alone
