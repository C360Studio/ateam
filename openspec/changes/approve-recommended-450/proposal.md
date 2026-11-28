# Proposal: approve-recommended-450
Approved-by:

## Track

`quick`: it adds one scenario under an existing harness Requirement, touches one capability, adds no command, flag or exported surface, and is no boot path, durability or gate file. The edits are Markdown instructions and their Go template copies: protocol § 2 Clarify (`.agents/protocol.md` step 1, `internal/bootstrap/templates.go:294`), the design skill (`.agents/skills/design/SKILL.md` § 2, `templates.go:454`) and the `proposal:` and `design:` rules in `openspec/config.yaml` (`templates.go:341`).

## Decisions you are making by approving

Approving accepts the recommended option of each decision unless you name another.

- **Every question put to you comes with a recommended answer, and approving accepts those answers.** Each open question in a proposal or its design names the option the author recommends, and the decisions section opens with the sentence above, as #421, #441 and #451's proposal already did. One approval at one digest then carries every ruling into the archived change, instead of answers left in pull request comments.
- **When you pick a different option, you approve again.** The developer rewrites the proposal or design to your choice and asks you to approve its new digest; your first approval does not carry over. This is the order hold-my-beer mode already follows (a design question first, the approval after).
- **The sentence is guidance, not a check (recommended).** It goes into the shipped protocol, design skill and OpenSpec rules; `task spec:check` does not fail or warn on a proposal without it. The other option, failing a proposal that lacks it, would add a spec Requirement and code and move this change to the design track; say so if you want that instead.

## Why

#450, from the v0.4.7 coordinator dogfood (notes 4 and 11): in default mode the owner got design questions and the approval request together, and answering a question in `design.md` changed the digest the approval named. A ruling given without changing the digest could live only in a PR comment, so it never reached the archived design.

## What Changes

1. Protocol § 2 step 1 (and the shipped protocol in `templates.go`): one sentence in the Stop boundary bullet.
2. Design skill § 2 (and its template): Decisions and Open Questions mark the recommended option; the decisions section opens with the sentence.
3. `openspec/config.yaml` `proposal:` and `design:` rules (and `openspecConfig`): the same, kept identical by the existing init test.
4. Harness spec: one scenario under *Opt-in design track*, with a shipped-text test.

## Scope & Non-Goals

- **In-bounds:** the files above, a test in `internal/bootstrap/adopter_rules_test.go`, the harness spec delta.
- **Non-goals:** any `spec:check` code; the coordinate skill and coordinator contract (#449 / PR #488 owns the owner-message wording); the architect and judge contracts, which draft with the design skill or rule on what it drafted; `AGENTS.md` and the rules index; gate files.

## Definition of Done

- A test asserts the new sentences in the shipped protocol and design skill, shown able to fail with a non-destructive mutation of each template; the existing init test keeps the shipped `openspec/config.yaml` rules equal to this repository's.
- `task verify`, `ateam doctor --check`, `ateam bench --selftest` green; change archived per the Close rule.
