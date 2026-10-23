## 1. Implementation

- [x] 1.1 Read other worktrees' state in `writePickupWorktrees` and print their handoffs, skipping the current checkout and slots the current state already holds; missing or unreadable state is silent (#136).
- [x] 1.2 `TestPickupShowsHandoffsRecordedInsideWorktrees` (fresh, HEAD-drifted, unreadable, current-checkout file). `task mutate:check` mutants, all `detection` at `state.go` sha256 `2a84e92e`: worktree read removed, HEAD-drift stale mark removed, current-checkout skip removed (#136).
- [x] 1.3 Cost: `BuildPickupSummary` with ten worktree states measured at about 0.2 ms per call (1.9 ms under `-race`), inside the hook budget (#136).
