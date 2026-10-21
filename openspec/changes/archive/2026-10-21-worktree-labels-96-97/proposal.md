# Change Proposal: `worktree-labels-96-97`

## Why

Two worktree labels misled pickup readers. A detached checkout was labeled `detached` by the hook but `HEAD … no PR` after a pickup. `ReadFastGitHead` names a detached checkout `HEAD`, and `reconcileLiveBranchHeads` stored that as the branch (#96). A branch created from `main` with no commits was labeled `merged (ancestor of main)`, so a claim with nothing pushed read as finished work (#97).

## What Changes

- `reconcileLiveBranchHeads` stores a live `HEAD` read as `""`, the form `git worktree list` parsing already uses, so every surface prints `detached`.
- `worktreePRStatusAndAge` labels a branch with no PR card, `ahead == 0`, and a head equal to the `main` head `no commits yet`. Branches at an older ancestor of `main` keep `merged (ancestor of main)`. Limit (tagged `// shortcut:`): a fast-forward-merged branch reads `no commits yet` until `main` moves, and a branch started from an older `main` still reads merged.
- The harness spec's worktree-label scenario names both labels.
- Not changed: the UI board's separate `Branch ancestor of main` evidence (`ui/src/lib/board.ts`).
