# Design outline

## Reuse the existing pipeline

The current Experiment carries baseline/candidate strings; ResolveExperiment creates retained provenance and schedule; RunExperiment re-resolves before invoking an adapter; AdapterRequest.Instructions is the shared native/generic transport; validateRun and rescore check retained evidence. Extend these boundaries instead of introducing another runner or provider interface.

Initial scope includes internal/bench/experiment.go, experiment_run.go and associated parser/run/replay tests, the bench CLI's existing configuration loading, benchmarks/examples and benchmarks/README.md. Native adapters should continue receiving the same resolved instruction field; provider-specific changes require evidence of an actual need.

## Explicit resolution

An optional `bundles: {baseline: "relative-manifest.json", candidate: "relative-manifest.json"}` selects both arms. Existing inline strings must be empty/omitted when selected; inline configurations and artifacts retain their existing format. Each manifest is `{version: 1, name: "label", files: ["ordered.txt", "inputs.md"]}`. Its directory is the bundle root. Config-relative manifest paths and manifest-relative file paths must be canonical local slash-separated paths without parent traversal or symlinks. There is no additional root selector or discovery.

The independent plan review approved this shape with strict unique-key JSON, bounded regular UTF-8 files, anchored `os.Root` reads, pure snapshot validation and known artifact-expansion checks. Limits: 64 KiB manifest, 1–32 files, 1 MiB effective instructions per arm including literal `\n\n` separators. Composition preserves file bytes; no metadata enters the prompt. Resolved experiments retain the original configuration and optional immutable bundle evidence; their effective Experiment has the composed inline strings and no unresolved bundle references. All non-treatment controls must match the original configuration.

Read only explicit inputs beneath the declared bundle root, with bounded aggregate content and clear errors for missing files, escaping paths and unsupported symlinks. Do not discover files, fetch URLs, execute text or automatically discover credential files or inherit their content. Resolve relative paths from the declaring configuration/manifest, not the caller's incidental cwd. Retain exact bytes, ordered component hashes and the effective instruction content/hash in the artifact. Metadata belongs in provenance; arm labels and filesystem names should not alter model prompts.

## Treatment semantics and replay

Equality is based on effective instruction bytes and fixed execution controls. Different labels, paths or version strings do not make an effective A/B treatment difference; identical content may be a valid A/A control. Dry run must expose this distinction. Reordering content that changes effective instructions is a real treatment change.

Before new inference, revalidate resolved inputs so changed local files cannot silently produce an experiment different from the reviewed snapshot. Reject known snapshot plus scheduled-request JSON larger than the existing 64 MiB artifact limit before inference; unknown model output remains subject to the existing final artifact limit. Comparison and offline rescore instead consume the immutable saved snapshot and must succeed after the original files are moved or removed. Old inline artifacts must remain unchanged and replayable.

Model/provider/reasoning, task set, ordering seed, repetitions and budget controls stay shared across arms. Do not add a competing router or allow init defaults to overwrite explicit experiment settings.

## Verification and risk

Use literal text/hashes and a small test-owned ordering/reference model as independent oracles. Cover equivalent metadata/different content, ordering, missing or changed files, symlink/path escape, aggregate size boundaries, stale/tampered snapshots, cross-arm isolation and replay after file removal. Decide generated-case coverage before implementation, attach test-to-spec references and prove critical guards through non-destructive mutations.

Risks are artifact compatibility, time-of-check/file-change handling, prompt normalization changing the treatment, accidental private-file capture and conflating instruction delivery with native skill execution. Do not claim instruction text caused a behavior or performance improvement merely because it was delivered. Any new native execution path needs calibration evidence before performance claims; this claim authorizes no paid calls.
