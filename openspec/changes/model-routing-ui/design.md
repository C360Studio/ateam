# Design

Read the served checkout's `.agents/review-policy.json` on each request; never interpret it as trusted PR approval policy. Return a compact summary rather than owners, review authors or full policy. Missing files mean unconfigured; malformed or oversized policy means invalid; I/O and non-regular/escaping paths mean unavailable. A structurally valid but ineligible route is blocked and retains both configured routes. Only valid policies receive a digest. All responses are non-cacheable.

Reuse Go review parsing and eligibility. Open a confined `os.Root`, reject non-regular policy paths, use a nonblocking open and recheck the descriptor to reject concurrent FIFO replacement, and cap input at the existing JSON limit plus one byte. No write, subprocess, network, or model launch occurs in the routing endpoint.

The dashboard receives the same summary through its existing refresh path. A pure Svelte component renders read-only model/provider/effort rows and textual status. Missing data from an older server shows unavailable. Explicit copy distinguishes configured intent, actual runtime identity, PR approval, effort and Ponytail mode.

Testing oracle: a literal approved developer/reviewer relationship must yield eligible; equal models with higher effort must remain blocked. Explicit fixtures cover the finite read/parse/eligibility states, same/different provider constraints, exactly-at/over size limits, and eligible -> blocked -> invalid -> deleted -> eligible histories. Existing parser tests cover grammar exploration; no new parser or ranking algorithm is introduced, so deterministic examples suffice. SSR component assertions cover every state and escaped untrusted text. Mutation checks must detect bypassed eligibility and false UI status.
