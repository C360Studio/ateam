# Change Proposal: `forge-token-roles-160`

## Why

ATeam resolves one forge credential for every operation: `GITEA_TOKEN`/`ATEAM_GITEA_TOKEN` on Gitea, `GH_TOKEN`/`GITHUB_TOKEN` or `gh auth` on GitHub (`cmd/ateam/review.go` `reviewToken`, `internal/state/forge.go`). A bot that only syncs the board therefore holds the token that can comment and create labels. Adopters asked for one token per role, named by operation, and a doctor nag when one token serves both (#160).

## What Changes

- `ATEAM_TOKEN_READ` authenticates forge reads: board sync and pickup (`SyncForgeBoard`, Gitea REST and `gh` lists) and review evidence (`ateam review check`, the merge check's Gitea reads).
- `ATEAM_TOKEN_WRITE` authenticates forge writes: PR handoff comments and label bootstrap, on Gitea REST and `gh`.
- Either one alone serves both roles. When neither is set, the existing resolution is unchanged.
- On GitHub the role token reaches `gh` as `GH_TOKEN` for that one command.
- `ateam doctor` warns, never fails `--check`, when reads and writes resolve to the same token, naming the variables and never a token value.
- `docs/adoption.md`, `docs/model-review.md` and `README.md` name the two variables.

## Non-goals

- Listing token scopes in doctor. Only GitHub classic tokens expose them (`X-OAuth-Scopes`), and only through a network call doctor does not otherwise make; Gitea has no cheap equivalent.
- The `tea` CLI fallback, which authenticates through its own login config.
- The GitHub merge check's `gh` calls under `internal/review/` (a gate file); they keep `gh`'s own resolution.
- GitHub fine-grained tokens versus a GitHub App installation token, and how either fits the merge gate: the open question on #160 stays open.
