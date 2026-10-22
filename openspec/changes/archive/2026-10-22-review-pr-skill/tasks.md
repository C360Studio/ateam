## 1. Implementation

- [x] 1.1 Write `.agents/skills/review-pr/SKILL.md`, the Claude adapter, the Gemini command, and the README bullet (#113).
- [x] 1.2 Add `review-pr` to `canonicalSkills`, `requiredSkillMarkers`, and `canonicalSkillBodies` so `ateam init` ships it (#113).
- [x] 1.3 Extend `TestInitRepoAndParitySensitivity`: the shipped body equals the repository file, the three files are written, and a canonical skill missing a record marker fails parity (#113).
- [x] 1.4 Add the `Shipped review-pr skill` scenario to the harness spec (#113).
