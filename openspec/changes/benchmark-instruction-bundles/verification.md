# Claim checkpoint

implemented-by: codex (claim/design only; implementation and independent approval pending)

Parent: #16. Focused implementation issue: #26. Base: 76e6f63d8deae62c075923f91fb703afe2d6c5a5. Dedicated branch: codex/benchmark-treatments.

Reviewed the current Experiment/ResolvedExperiment/AdapterRequest boundaries, resolution-before-execution behavior and affected experiment tests. No runtime code, dependencies or generated client settings changed. No configured .agents/review-policy.json exists at this base; recheck policy and route before implementation if one is added.

This checkpoint makes the next slice concrete and reviewable. It does not claim completed implementation, independent design approval, live calibration or performance gains. Gemini capacity retry remains deferred; no credentials or previous raw benchmark traces are copied into this worktree.

Direct documentation checks: `openspec validate benchmark-instruction-bundles --strict --no-interactive` passed; `git diff --check` passed. Runtime tests and full implementation gates are deferred because this draft contains only the claim/design, not implementation.
