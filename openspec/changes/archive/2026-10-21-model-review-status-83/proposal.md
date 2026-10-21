# Change Proposal: `model-review-status-83`

## Why

`docs/model-review.md` read as a description of the operating gate, but in this repository the system it describes is switched off (ADR 0001). Two sentences were false for the gate that runs: "no docs-only relaxation or filename classifier", and an exact-head binding rule that the up-to-date ruleset would turn into a re-review on every PR (#83, ADR 0001 step 4).

## What Changes

- A status block at the top of `docs/model-review.md` covers:
  - the operating gate: `scripts/merge-check.sh` with model-named lines and commit-bound records;
  - the planned never-weaker strength check (#81 ruling, #82);
  - what turns on with a policy (#81), and what stays off (JSON records, reviewer-executed checks, exceptions);
  - a link to ADR 0001.
- The binding sentence is scoped to `ateam review check`. The missing merge-of-`main` / docs-only exemption in `validateBinding` is recorded as a limitation to fix before enabling that path; the code is not changed.
- The classifier sentence is scoped to `ateam review check` and now describes merge-check's classification by extension and its exemptions.
