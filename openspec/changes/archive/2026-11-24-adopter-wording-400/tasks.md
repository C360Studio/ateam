# Tasks: adopter-wording-400

## 1. Approval

- [x] 1.1 Owner approves the proposal (`Approved-by: Coby Leuschke (2026-11-24) sha256:c91a35a21303`).

## 2. Implementation

- [x] 2.1 Ponytail step 1 caller test and defensive-code bound, in `.agents/skills/ponytail/SKILL.md` and the generated skill.
- [x] 2.2 Oracle definition in `.agents/protocol.md` § 3 and `defaultProtocolMD`.
- [x] 2.3 Oracle question and evidence words in `.agents/contracts/reviewer.md` and `reviewerContractFor`.
- [x] 2.4 Bootstrap test asserting each new sentence in generated output; mutation check per sentence.
- [x] 2.5 Run `task verify`, `ateam doctor --check`, `ateam bench --selftest`; archive this change.
