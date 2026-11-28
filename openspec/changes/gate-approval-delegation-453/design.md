# Design: gate-approval-delegation-453

Drafted by claude (opus, high) in the developer run, under the design skill. Read-only on the deliverable until approval; the owner rules. Q1 and Q2 were ruled by the owner on 2026-11-28 (#453); D2, D3 and D5 are the details those rulings leave open.

## Context

- `.agents/protocol.md:94` (*Gate files*): `Gate files` fails "until the owner adds the label `gate-change-approved` and posts a comment starting `gate-change-approved: <sha>`" at the live head; "Only the owner applies the label and writes the comment. An agent blocked by it hands off; it never applies the label or writes the comment itself." Under a shared login both are "an attestation, not proof of who applied it". Owner-only since #140 (`e437cae`).
- `.agents/protocol.md:101` (*Unattended and fan-out runs*): "It hands off; it never applies the label."
- `.agents/contracts/developer.md:7` (`internal/bootstrap/review.go:23`): a coordinated developer never asks the owner in its own session; it posts the question on the PR and hands off. Nothing covers the gate-file approval.
- `.agents/contracts/coordinator.md:5` (`templates.go:368`): "never apply `gate-change-approved`"; `:7`: a change touching a gate file is an owner gate; `:9`: in hold-my-beer mode it still stops for the owner and ends the run. `internal/bootstrap/coordinator_test.go:43` pins the "never apply" literal.
- `.agents/skills/coordinate/SKILL.md:26` (`templates.go:421`): the owner message names "the PR or issue, the question in plain words, and what you need (for a proposal, the digest `task spec:queue` prints)"; nothing for a gate file.
- `scripts/gate-files.sh:113-115`: fails without the label; `:126`: the newest comment whose body starts `gate-change-approved: <7-40 hex>` wins, with no author check (the whole-review finding, out of scope); `:131-140`: fails unless that sha prefixes the live head.
- `.github/workflows/gate-files.yml:12-13`: runs on `labeled`/`unlabeled` and pushes, not on comments; `.gitea/workflows/gate-files.yml:13`, `:26-30` likewise on `label_updated`. So a comment alone reruns nothing; the label event after it does.
- `AGENTS.md:91`, `.agents/rules-index.md:42`: "only the owner applies them, and a blocked agent hands off". `docs/adoption.md:176`: *Gate-change approval*, decided by the owner. `docs/adr/0003-roles-and-routes.md:26`: the developer's *May write* cell.
- `openspec/specs/harness/spec.md:524-533` (*Gate-file changes wait for the owner*) specifies the check, not who posts; `:1438` keeps the coordinator from applying the label. PRs #483 and #484 edit that file.
- `internal/bootstrap/review_test.go:151-158` and `coordinator_test.go:30-39` fail when a shipped contract or skill differs from the repository's.

## Goals / Non-Goals

**Goals:** the owner approves a gate-file change in one word to the right run; the approval still names one exact commit; the coordinator stays write-free.
**Non-Goals:** any gate-file edit, a later-head tolerance (Q2 = B), an author check on the comment, coordinator or judge writes.

## Decisions

### D1. Who posts the approval (ruled: Q1 = A)

- **A (ruled): the developer run that claimed the PR**, on the owner's yes naming the PR and head sha. The coordinator never posts; its owner message links that developer's session. Blast radius: one developer write, two forge calls, on its own PR.
- **B: the owner only, as written; the message gives the exact command.** No new write, but the owner leaves the conversation for a terminal, the friction #453 reports.
- *Adopter seam:* adopters' developer contracts gain the write on the next `ateam init` or contract reconcile. A team that wants owner-only keeps its contract text; the check is unchanged either way.

### D2. What counts as the owner's yes

- **A (recommended): the owner's own words in the developer's session, naming the PR number and a sha the developer confirms is still the live head.** A relayed yes (coordinator, judge, another PR's comment) or one that names no sha is not an approval: the developer asks again. Follows from the ruling's link to "that developer conversation".
- **B: a yes the coordinator relays counts.** Fewer hops, but the coordinator's text becomes the approval, which reopens what Q1 closed and lets a router's paraphrase move a gate.
- *Adopter seam:* a host without conversations (owner at a terminal) is the same session. If nobody follows D2, the check still demands the exact head; only the provenance of the yes is weaker.

### D3. Order of the writes and a moved head

- **A (recommended): comment first, then the label; when the label is already on from an older head, remove and re-add it.** The label event is what reruns `Gate files` (`gate-files.yml:13`). If the head moved between the yes and the post, post nothing and hand off `blocked` naming the new head.
- **B: label first.** The check runs, finds no comment, fails, and nothing reruns it until another label event or push.

### D4. When to ask (ruled: Q2 = A)

- **A (ruled): keep the exact-head rule; ask last.** The developer asks only after the approving review, the archive commit and an update from `main`, when this PR is the next to merge (#451's merge plan puts gate-file PRs last). One sentence in the protocol and the developer contract; no script change.
- **B: accept a later head that differs only by clean `main` merges.** Would loosen `scripts/gate-files.sh`, a gate file. Not taken.

### D5. Spec

- **A (recommended): no delta (`skip_specs: true`).** The check is unchanged and the scenario at `spec.md:524` describes it; the new route is instruction text pinned by the init tests. Keeps clear of #483 and #484.
- **B: reword the scenario's "until the owner posts".** More literal, but rewrites a long Requirement block (strict-length risk after archive) for no behaviour change.

## Risks / Trade-offs

- Under a shared login the forge cannot tell an owner's comment from an agent's, so D2 is an instruction, not a control; that was already true (`protocol.md:94`). An author filter by role (the out-of-scope finding) would not separate them either; filtering by the label actor's login would, and stays a gate-file decision.
- `scripts/gate-files.sh:114`, `:130` still print "waits for the owner" and "the owner posts"; still accurate (the approval is the owner's), and editing them is a gate-file change.

## Migration Plan

None. Adopters get the text on the next `ateam init` or contract reconcile.

## Open Questions

None for the owner beyond approving D2, D3 and D5; D1 and D4 record the rulings.
