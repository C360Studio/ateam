## 1. Implementation

- [x] 1.1 Label worktrees without a PR card as `no PR` and qualify forge-derived worktree labels with `(cache <age>)` + `labels from forge cache synced <age> ago; run ateam state pickup --sync` when the forge cache is `>= 10m` old (#73).
- [x] 1.2 Add unit, hook, and non-destructive mutation checks covering fresh vs stale forge caches and `no PR` worktree labels (#73).
