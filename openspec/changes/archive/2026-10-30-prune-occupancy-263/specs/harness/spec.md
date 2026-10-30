## MODIFIED Requirements

### Requirement: Prune finished worktrees the caller owns

`ateam state prune [--dry-run] [--agent <name>] [--force]` SHALL remove a linked worktree, its local branch, and its remote branch only when the caller's agent owns the worktree, its tree is clean, the refreshed state labels it finished, and, unless `--force` is passed, no other session occupies it. It SHALL list every other linked worktree with the reason it was kept. Pickup SHALL mark each worktree the same check would remove.

#### Scenario: A finished, clean, owned worktree is removed

- **WHEN** `claude` runs `ateam state prune` and a linked worktree on `claude/<topic>` is clean and labelled `merged via #<n>` or `merged (ancestor of main)`
- **THEN** the worktree is removed only if its live branch and head still match the refreshed state, the local branch is deleted only while it points at that head, the remote branch is deleted only under a lease on that head (`--force-with-lease`), and one `removed <branch>: <label>` line is printed
- **AND** a worktree whose head or branch moved since the refresh is kept as `moved since refresh`, a local or remote branch that moved before its deletion is kept and named, and a branch deletion that fails without the branch having moved is reported as `failed` with git's error

#### Scenario: Unsafe or foreign worktrees are kept

- **WHEN** a linked worktree is dirty, has an unknown dirty state, is labelled `ahead of merged`, `open PR`, `no PR`, or `no commits yet`, is on another agent's prefix or no agent prefix, is the current checkout, or is on the default branch
- **THEN** it is listed as `kept <branch>: <reason>` and nothing about it changes

#### Scenario: A detached review checkout of a closed PR is removed

- **WHEN** a detached linked worktree's directory is named `<agent>-review-pr-<n>[-<suffix>]` for the caller's agent, no cached PR `<n>` on any forge is open, and its head equals the head of a closed or merged cached PR `<n>`
- **THEN** the checkout is pinned to a temporary branch, removed, and the pin deleted only while it still points at that head, so a commit landing during removal stays on the pin
- **AND** if any cached PR `<n>` is open, or no closed PR `<n>` has that head, the worktree is kept

#### Scenario: Dry run and pickup hint

- **WHEN** `--dry-run` is passed
- **THEN** the same lines are printed with `would remove` in place of `removed`, and no worktree, branch, or remote ref changes
- **AND WHEN** pickup renders a worktree the same check would remove for the current agent, its line ends with `, prune candidate)`

#### Scenario: A finished worktree another session occupies is kept

- **WHEN** a finished, clean, owned branch worktree's own `.ateam/writer.json` names a session other than the caller's whose pid is alive or whose `last_seen` is within the writer-lease TTL, or its own `.ateam/state.json` holds a handoff for its branch from a session other than the caller's written less than 30 minutes ago
- **THEN** it is kept as `session <id> holds the lease (<age> ago)` or `session <id> handed off <age> ago; prune after 30m or pass --force`, and pickup does not mark it a prune candidate
- **AND** a lease or handoff from the caller's own session does not keep it, while with no resolvable caller session every lease and handoff does
- **AND WHEN** `--force` is passed, the worktree is removed as if neither rule applied (with `--dry-run`, listed as `would remove`), and every other keep reason still holds
