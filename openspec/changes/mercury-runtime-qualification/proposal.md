# Mercury runtime qualification

## Why

Issue #58 asks whether Mercury 3 can run inside an existing native agent while ATeam remains the bootstrap and measurement layer. A direct API probe does not establish native-runtime compatibility. Claim this investigation without changing the existing Claude, Codex or Gemini contracts.

## What Changes

- Claim #58 with a documentation-only compatibility audit and independently reviewed plan.
- Inspect the documented Claude Code/LiteLLM path and pinned public source without installing or executing the bridge.
- Record concrete forwarding, isolation and accounting blockers, then specify the smallest offline tests that could resolve them.
- Keep Codex compatibility unproved and leave native skills to #28 and cross-runtime experiments to #59.

## Impact

Addresses #58 and roadmap #16; informs #59 and #28. This change owns only `openspec/changes/mercury-runtime-qualification/`. No production source, dependency, runtime configuration, historical benchmark evidence or global settings change.

The deliverable is a scoped draft claim plus a reviewed compatibility plan identifying a candidate path or concrete blocker. Completion of this planning slice is not completion of #58. Stop before bridge installation, downloaded-code execution, adapter implementation or paid calls. A later request must authorize installation and implementation; a separately reviewed bounded protocol and explicit authorization are required for any canary/A/A. The earlier Gemini budget supplies no authorization.

Risks are silent parameter changes, provider/model misidentification, inherited credentials, extra requests and missing usage becoming zero. Acceptance is a source-linked audit, explicit unresolved controls, independent read-only review, and repository checks with failures/unavailable checks disclosed.
