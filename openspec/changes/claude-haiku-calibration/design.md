# Design

Use Claude Code 2.1.292 in bare, restricted print mode with API-key authentication, one fixed `claude-haiku-5-5` model and explicit low effort for the pilot. Fresh homes/config directories and no session persistence prevent session reuse. Bare mode excludes OAuth/keychain and discovered customizations; read-only built-in tools and empty strict MCP limit the tool surface. Local managed configuration is refused and remote policy remains enforced.

Normalize complete per-model input + cache creation + cache read and output exactly once. Claude top-level usage describes the main loop, whereas modelUsage includes helpers; retain both and do not add them together. Reported cost is a CLI estimate, not reconciled billing. Any extra model rejects the fixed-model contract. Failures, malformed/incomplete output, truncated streams and unknown usage remain visible and stop as appropriate.

Keep the existing Gemini artifact shape compatible; add optional Claude provenance/evidence without rewriting originals. Share native runner checks through small provider dispatch helpers, not a new plugin framework.

Acceptance: fake CLI process integration proves isolated environments, exact arguments, refusal/timeout/redaction; independent parser oracles and generated counter cases prove accounting; non-destructive mutations activate key assertions; existing Gemini and generic replay pass; full clean-tree task verify and independent review before push.

Live pilot: same 30 attempts/100,000 reported tokens/120 seconds per attempt/one hour/$2 target policy, not a hard monetary guarantee. Credentials are absent at planning time. Before launch, obtain ANTHROPIC_API_KEY through the local ignored environment file and record isolated claude doctor remote-policy outcome. Never disable remote policy to force a run. No paid inference until that prerequisite is satisfied.
