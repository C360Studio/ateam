# Session-only trust for generated benchmark workspaces

## Why
The first approved native pilot stopped with Gemini CLI exit 55: fresh generated workspaces have no persisted trust. The user explicitly approved session-only trust for these workspaces.

## What Changes
Set GEMINI_CLI_TRUST_WORKSPACE=true in the isolated Gemini child environment and disclose it in dry-run policy and documentation. Preserve API-only auth, managed-policy refusal, fixed model/settings, restricted tools and fresh homes. Do not change global trust files or the parent environment.

## Capabilities
### Modified Capabilities
- `benchmark-experiments`: scoped native workspace trust.

## Impact
Native adapter environment, fake-child isolation regression, dry-run policy and benchmark documentation. No new dependencies or routing changes.

implemented-by: https://github.com/C360Studio/ateam/pull/13
