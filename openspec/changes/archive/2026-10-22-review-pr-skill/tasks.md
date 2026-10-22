## 1. Implementation

- [x] 1.1 Write `.agents/skills/review-pr/SKILL.md`, the Claude adapter, the Gemini command, and the README bullet (#113).
- [x] 1.2 Add `review-pr` to `canonicalSkills`, `requiredSkillMarkers`, and `canonicalSkillBodies` so `ateam init` ships it (#113).
- [x] 1.3 Extend `TestInitRepoAndParitySensitivity`: the shipped body equals the repository file, the three files are written, and a canonical skill missing a record marker fails parity (#113).
- [x] 1.4 Add the `Shipped review-pr skill` scenario to the harness spec (#113).
- [x] 1.5 Raise the `ReadDiskBinaryBuild` exec timeout from 2 s to 10 s so the provenance test does not fail under full race-suite load (#116, found by the Codex review of `aaf5a38`).
- [x] 1.6 Add `TestReadDiskBinaryBuildOutlivesTheOldTwoSecondBudget`, a 3 s probe that must be read, and record `task mutate:check` detection when the 2 s budget is restored (#116, from the Codex review of `9ca032e`).
