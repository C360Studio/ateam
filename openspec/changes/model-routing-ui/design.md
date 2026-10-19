# Design

Read the selected repository's `.agents/review-policy.json` on each request; never interpret it as trusted PR approval policy. Return a compact summary rather than owners, review authors or full policy. Missing files mean unconfigured; malformed or oversized policy means invalid; I/O and non-regular/escaping paths mean unavailable. A structurally valid but ineligible route is blocked and retains both configured routes. Only valid policies receive a digest. All responses are non-cacheable.

Reuse Go review parsing and eligibility. Open a confined `os.Root`, reject non-regular policy paths, use a nonblocking open and recheck the descriptor to reject concurrent FIFO replacement, and cap input at the existing JSON limit plus one byte. No write, subprocess, network, or model launch occurs in the routing endpoint.

The dashboard receives the same summary through its existing refresh path. A pure Svelte component renders read-only model/provider/effort rows and textual status. Missing data from an older server shows unavailable. Explicit copy distinguishes configured intent, actual runtime identity, PR approval, effort and Ponytail mode.

Testing oracle: a literal approved developer/reviewer relationship must yield eligible; equal models with higher effort must remain blocked. Explicit fixtures cover the finite read/parse/eligibility states, same/different provider constraints, exactly-at/over size limits, and eligible -> blocked -> invalid -> deleted -> eligible histories. Existing parser tests cover grammar exploration; no new parser or ranking algorithm is introduced, so deterministic examples suffice. SSR component assertions cover every state and escaped untrusted text. Mutation checks must detect bypassed eligibility and false UI status.


Restore the unpushed UI history from `codex/ui-preserved-work` (`26d5abd`) alongside current main. Repository selection is global to the running server; mutations carry `X-ATeam-Repo` and refuse stale clients before loading or writing state. Single-repository API clients can omit the header for compatibility. Dashboard loads discard superseded responses, and failed transitions clear stale state. Registered repository paths persist atomically; active selection starts at the served checkout on restart. Webhooks remain bound to the startup repository.

The restored Current Ops view only labels the selected checkout clean/dirty; other worktrees are uninspected. Handoff checkpoints compare compatible SHA prefixes. Native dialogs provide focus containment and Escape dismissal, with errors inside the active dialog. Canonical OpenSpec parsing feeds requirement details; card associations remain inferred navigation hints. Benchmarks run only on explicit request and display evaluated and skipped counts.

Forge settings validate target components and persist explicit removal. Until explicitly configured, detection follows origin changes. GitHub sync passes the configured target on every request; custom GitHub API URLs are rejected. Explicit Gitea target failures cannot fall back to a different CLI repository.
