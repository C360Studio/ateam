# Export and review retained experiment reports

`ateam bench export` turns one validated paired experiment into three deterministic generated files: `results.json` (facts), `results.md` (readable facts), and `evidence.json` (source and generated-file digests). It makes no model calls, starts no adapter or grader, and does not publish anything. A valid report can describe a failed or stopped experiment; a successful export is not a successful experiment.

## Local workflow

Retain the original authorized run under its frozen protocol. Build the CLI from the repository root, then export to a directory that does not exist:

```sh
go build -o bin/ateam ./cmd/ateam
bin/ateam bench export benchmarks/runs/experiment-<id>/run.json --output /tmp/ateam-report-new
bin/ateam bench export benchmarks/runs/experiment-<id>/run.json --check /tmp/ateam-report-new
```

Flags follow the artifact path. Exactly one nonempty `--output` or `--check` is required. Use a trusted destination parent that remains stable during initial directory creation/open. Portable Go cannot prove directory ownership if another actor replaces it between creation and the first identity capture; use a private workspace, not a concurrently manipulated shared parent. After capture, anchored operations detect directory replacement. Export refuses existing destinations, including symlinks, and does not overwrite a report. Failed writes can leave an incomplete newly created directory for diagnosis; check rejects an incomplete or altered generated set. Check writes nothing and never repairs drift.

If the owner separately ran `bench rescore` and retained its artifact, add `--replay <rescore-run.json>` to both export and check. Export validates the association without regrading. `not-provided` means no replay was supplied; `matched` means recorded grading outcomes agree; `different-results` retains the original outcomes and lists disagreements. An incompatible execution identity is refused. Agreement and hashes do not prove independent execution or authorship.

Review all three files before copying them into the intended repository experiment directory. Keep hypotheses, interpretation, limitations and the next decision in a separate human-authored `interpretation.md`. Export and check never create, overwrite, delete or verify that file. Record the exact question and treatment in the experiment's reviewed protocol without adding private prompt text to the generated report. Commit reviewed files through the normal repository review process.

## Evidence and interpretation boundaries

The generated report is a versioned summary, not a replayable run artifact. The evidence manifest also identifies the exporter library sources (`report.go`, `report_markdown.go`, `report_files.go`); this digest is not a whole-binary or transitive dependency attestation. Original and replay artifacts are not included. Default projection excludes raw prompts/instructions, source, traces, diagnostics, environment, native session IDs and absolute paths; included metadata is escaped for display. This is not a universal secret scrubber: inspect remaining labels before publication. Keep private originals in their existing restricted location, not in public fixtures or CI uploads.

Known/unknown usage and the contributing denominators remain explicit; cache and reasoning counters are subsets. Execution and grading time stay separate. Costs remain unknown. Reports do not infer a winner, significance, equivalence, skill activation or ROI. A task set with saturated correctness does not establish that a treatment has no benefit elsewhere.

Version 1 supports the existing paired experiment schema. Dynamic Ponytail settings and an opt-in JEV-like classifier are follow-up work tracked by [#91](https://github.com/C360Studio/ateam/issues/91) and [#95](https://github.com/C360Studio/ateam/issues/95). This exporter does not implement their runtime behavior. The current producer schema does not record classifier latency as a distinct measurement: do not label it zero, infer activation decisions, or attribute an aggregate duration delta to classification. A future experiment protocol and producer schema must record those decisions and timing boundaries before reports can compare their overhead.

## Reproducible checks

```sh
task bench:report-check
```

This offline gate checks redistributable synthetic inputs and their expected report bytes, with no API calls. `task verify` also runs the worked-example test as part of its full unit suite. The gate validates the exporter and public examples; it cannot rederive a real report from hashes alone when the original artifacts remain private. To verify a real report, use `--check` with its actual retained original and the same optional replay. Missing private originals remain a stated verification limit; the workflow never fetches them automatically.
