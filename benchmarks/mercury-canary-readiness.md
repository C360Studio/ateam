# Mercury canary readiness — October 21, 2026

Historical preparation snapshot. The owner later authorized the canary and A/A separately; both completed with retained replayable evidence. See [calibration results and the proposed instruction A/B](experiments/mercury-instructions-v1/README.md). The authorization and credential statements below describe preparation time, not permission for another run.

Addresses #58 after PR #77 merged at `e88be0555091420ca103770597b28efa0cc5e4e1`. This records preparation of the [reviewed protocol](mercury-canary-plan.md); it does not authorize paid execution. The existing OpenSpec spending hold remains open. No runtime code, benchmark controls, native runtime defaults or frozen Gemini evidence changed.

## Frozen local inputs

Fresh branch `codex/mercury-canary-preparation` starts at the fetched merge commit. Private preparation artifacts live in `.ateam/mercury-canary-preparation/` in that worktree. `canary.json` and `aa.json` copy the checked-in examples with only executable paths resolved explicitly:

- Claude: `/Users/coby/.local/share/claude/versions/2.1.292`, reporting `2.1.292 (Claude Code)`.
- Bridge: `/Users/coby/Code/c360/ateam/.worktrees/codex-mercury-cancellation-qualification/.ateam/teardown-env/litellm-go`, reporting `LiteLLM: Current Version = 1.99.0`.
- Existing isolated bridge dependencies: Python 3.14.5, HTTPX 0.28.1, OpenAI 2.54.0. The wrapper and its original environment must remain available; no installation or upgrade was performed. Executable hashes do not authenticate every transitive dependency.
- Runner: built from clean documentation commit `54e9fa1aaa72d2ac402cd028417c1cc3eb55276f`; `cmd/`, `internal/`, `go.mod` and `go.sum` are unchanged from the merge commit. Go 1.26.4, darwin/arm64. Both embedded `vcs.revision` and explicit `BuildCommit` identify `54e9fa1`; `vcs.modified=false`. Keep this binary with the frozen inputs rather than using an ambient installed `ateam`.

| Input | SHA-256 |
| --- | --- |
| Claude executable | `97a01e5bc74a199e67189435d0331ea3a24eac2e07db4b76d9148c5b0386138f` |
| Bridge wrapper | `12a6220c313db3beb493a32079b43cba1008f7e1a7dd9d44a74200dc69d325e9` |
| Runner executable | `ef61c27ccb28fbfd3b478bf6ae9fbf1009cf8a461af0f20d5c74c0005e1b4de5` |
| Canary resolved config digest | `f6aa66510ed12975d1b9ffb1af2c44bc1eaaea1724ca49380b2be2cc7f090ac4` |
| A/A resolved config digest | `f3f3cebaedda842d46c737329b096c047a1336e3d8af56d186c219a4fa864c3c` |

The config digests are the runner's canonical `config_sha256`, not file-byte hashes. Full settings, schedule, instruction text, grader/evaluator digests and command hashes are retained in `canary-dry-run.json` and `aa-dry-run.json`. Both arms contain exactly `Return a correct, minimal Go implementation. Follow the requested JSON schema.` No native hook/skill activation or full ATeam instruction treatment is claimed.

Independent review caught the initial build's automatic VCS stamp naming the shared root's older `d9ac179` commit. That runner was replaced, both dry-runs refreshed, and their earlier output retained under `superseded/`. The corrected build scoped Git explicitly to this worktree; `runner-build.txt` retains `go version -m` output:

```sh
GIT_DIR="$(git rev-parse --absolute-git-dir)" GIT_WORK_TREE="$PWD" \
  GOCACHE="$PWD/.ateam/go-cache" go build \
  -ldflags '-X github.com/c360studio/ateam/internal/state.BuildCommit=54e9fa1aaa72d2ac402cd028417c1cc3eb55276f' \
  -o .ateam/mercury-canary-preparation/ateam ./cmd/ateam
```

That build command was run at clean `54e9fa1`; replay it only from that source revision to reproduce the provenance claim. Do not propagate the scoped Git variables into tests that create their own repositories.

## Preparation checks

`task bench:selftest` passed (exit 0) on the merge commit. Its output is retained at `.enjoy-logs/mercury-selftest.log`. Both credential-free commands below passed (exit 0); they probe versions and hashes but launch no bridge or inference:

```sh
env -i PATH="$PATH" .ateam/mercury-canary-preparation/ateam bench run .ateam/mercury-canary-preparation/canary.json --dry-run
env -i PATH="$PATH" .ateam/mercury-canary-preparation/ateam bench run .ateam/mercury-canary-preparation/aa.json --dry-run
```

Dry-run confirms two safe-path attempts for canary and six attempts across the three example tasks for A/A. Both use low effort, 8192 output tokens/request, at most four provider requests/attempt, 30 seconds/attempt, identical arms, read-only tools and isolated native settings. No retries, fallback, replacement attempts or automatic advancement are proposed. Canary stops at 180 seconds or 50,000 accounted tokens; A/A has its own 360-second and 50,000-token limits. Token limits are checked between attempts and can overshoot. Missing usage or execution failure stops the run and requires reconciliation.

The local `ateam state pickup` forge refresh failed; its cached PR list is not authoritative. Git fetch succeeded, and the GitHub connector independently confirmed PR #77's merge and the latest #58 handoff. Main in the shared root was left untouched because another writer was active. The fresh worktree owns this preparation only.

## Pricing and authorization boundary

[Inception's model page](https://docs.inceptionlabs.ai/get-started/models), checked October 21, 2026, lists Mercury 3 at $0.20/M input, $0.02/M cached input and $0.70/M output, with launch prices of $0.10/$0.01/$0.35 respectively. The [announcement](https://www.inceptionlabs.ai/blog/introducing-mercury-3) corroborates those rates. Budget using regular prices: 50,000 accounted tokens would cost at most $0.035 if all were output; the two stages' 100,000-token nominal total would be $0.07. These calculations exclude overshoot and unknown charges. The combined $1 target remains a target, not a hard cap.

Account-specific pricing, available credit and dashboard charges were not inspected. `INCEPTION_API_KEY` was absent from the preparation process; credentials must be supplied through the authorized runner environment, never committed or pasted into evidence. Only the Go runner receives the provider key; children receive local tokens. No subscription authentication is proposed.

After explicit owner authorization and credential provisioning, the concrete canary command from this worktree is:

```sh
.ateam/mercury-canary-preparation/ateam bench run .ateam/mercury-canary-preparation/canary.json --output benchmarks/runs/mercury-canary
```

Immediately before execution, recheck the frozen config/executable hashes and preserve every outcome and provider-usage artifact. Review both attempts for complete usage, expected model/controls, structured source, replayability and grading; reconcile dashboard charges before any A/A. A/A needs a separate decision after that review. The earlier actual-stack offline qualification is historical evidence from PR #77, not a fresh live-provider test. Remote cancellation, invoice finality and model quality remain unproved.
