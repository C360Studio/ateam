# Change Proposal: `strict-openspec-127`

## Why

`openspec validate --all --strict` fails on `main` because three requirement statements are longer than 500 characters (#127). CI never noticed: `spec:check` runs the validator only when the root `node_modules/.bin/openspec` exists, and the CI job never installed the root packages.

## What Changes

- Three requirement statements keep their headers and their normative content, but detail moves into scenarios so each statement is 500 characters or shorter: `benchmark-experiments` *Owned Mercury bridge startup*, `benchmark-reports` *Deterministic non-overwriting output and read-only checks*, `model-review` *Model tiers and effort bands policy*. No behavior changes.
- CI installs the root packages, so `spec:check` runs the strict validator.
- `spec:check` prints a notice when the validator is not installed, so a local green says what it did not check.
