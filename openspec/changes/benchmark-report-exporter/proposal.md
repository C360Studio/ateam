# Offline benchmark report exporter

## Why

Issue [#89](https://github.com/C360Studio/ateam/issues/89), under roadmap #16, closes the gap between replayable benchmark evidence and a human-readable worked example. Mercury's completed A/B is saved in [PR #85](https://github.com/C360Studio/ateam/pull/85) at `fc854aebc24ec753233916555c888e50c570ca66`, but its report required manual assembly. Existing `compare` and `rescore` already provide trustworthy seams; a new statistics engine or model-generated verdict is unnecessary.

## What Changes

Implement a credential-free `ateam bench export` command that validates retained runs and emits deterministic JSON, Markdown and an evidence manifest from an explicit field projection. Add a read-only check mode for generated-file drift, optional retained-replay verification, and documented repository closeout. Reuse current artifact validation/comparison; preserve private original evidence and human interpretation.

## Scope and stop boundary

This change delivers the offline exporter and check command, synthetic worked examples, tests, and usage documentation on `codex/benchmark-report-exporter`. It began as a reviewed design claim at `fb144947f530ab4a4b131d38c981940647c89515`; implementation integrates main at `5b59d9d854c4aa8465b3e7e09ba2dee5cedd7755`. Delivery stops at review readiness, without merging. The dynamic Ponytail epic #91 and its later evaluation slice #95 remain separate; version 1 cannot infer classifier activation or isolated classifier latency from existing artifacts.

Implementation scope is CLI + Go bench reporting + offline fixtures/tests + documentation/verification entrypoint. No new model calls, installation, runtime/grader/task changes, dashboard viewer, cloud storage, automatic PR publishing or automatic next-experiment launch. PR #85 owns the Mercury worked example; do not duplicate its raw artifacts or merge it as part of this claim. Review/test against that example after it is available, retaining synthetic fixtures for standalone CI.
