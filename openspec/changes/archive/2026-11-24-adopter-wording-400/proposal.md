# Proposal: adopter-wording-400
Approved-by: Coby Leuschke (2026-11-24) sha256:c91a35a21303

## Track

`quick`: wording in three generated documents and their repo copies; no spec Requirement, capability, gate file, CLI flag or boot path changes. The generated text reaches adopters through `ateam init`, which is why each change lands in both the repo copy and its template.

## Why

An adopter's whole-repo review (#400) found four classes that passed every gate because the wording allowed a good-faith misreading: mechanisms with no caller, defensive machinery justified by "never cut", "independent oracle" tests that mirror the code under test, and measured-sounding claims with no artifact.

## What Changes

1. **Ponytail step 1 (caller test):** name the caller or spec scenario that needs it today; with none, do not build it.
2. **Ponytail guardrail (defensive bound):** defensive code names the concrete failure it prevents and how that failure occurs; a defence that turns a transient fault into a safety action is itself a hazard and needs a feature's justification.
3. **Protocol § 3 (oracle definition):** expected values are literals, a hand derivation shown in the test, or named external data; a helper mirroring the code under test's control flow is not an oracle. The reviewer contract asks, per oracle, where its expected values came from.
4. **Reviewer contract (evidence words):** a claim cites its artifact, using `measured` (with the artifact), `modelled` or `compile-checked`; the reviewer rejects `verified` or `proves` without an artifact. No word-list lint (#400 rules it out).

## Scope & Non-Goals

- **In-bounds:** `.agents/skills/ponytail/SKILL.md` and its `canonicalSkillBodies` entry in `internal/bootstrap/templates.go`; `.agents/protocol.md` § 3 and `defaultProtocolMD`; `.agents/contracts/reviewer.md` and `reviewerContractFor` in `internal/bootstrap/reviewrecord.go`; the bootstrap tests that pin that text.
- **Non-goals:** the `AGENTS.md` ladder and its template (byte budget; the skill carries the detail), the hook-injected ladder in `internal/hook`, any lint or `spec:check` rule, the developer contract.

## Definition of Done

- Each of the four rules reads the same in the repo copy and the generated file; a bootstrap test asserts each new sentence in the generated output, and `task mutate:check` shows the test fails when a sentence is removed.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived before ready.
