# Design

Reuse the existing runner, protected grader, counterbalanced schedule, immutable artifacts and offline rescore. Add an optional built-in Codex configuration; legacy command adapters remain compatible. Codex runs in fresh workspaces and homes with an explicitly inherited CODEX_API_KEY only, no user configs or resumed sessions. Record CLI identity, requested model/reasoning, observed native events, and unknown resolved settings. Redact the inherited credential before retaining streams; traces remain sensitive local artifacts.

Explicit `mode: aa` requires identical arm instructions; default/`ab` still requires different instructions. Compare all statuses and known/missing usage with descriptive dispersion; do not require identical stochastic output or infer equivalence.

Run budgets require a finite attempt count, per-attempt timeout, and token ceiling for native execution; reported usage is a between-attempt stop signal, not a guarantee against in-flight spend. Unknown usage stops subsequent native calls. Dollar cost remains unknown without a supplied price basis; live authorization must establish an external/provider cap if a hard monetary bound is required.

Risks: CLI event/config drift; incomplete failure usage; prompts/traces can contain sensitive data; local workspaces are not security boundaries; native read-only sandbox protects model tools only, generated Go grading remains local execution. Offline fixtures prove plumbing, not provider accounting. Acceptance requires full tests, mutation evidence, native fake process isolation, immutable rescore, independent change review, and a prepared pilot. Live calibration is a separate authorized execution boundary.
