# Design: artifact to reviewed report

## Reuse and ownership

The path is retained run → validation → existing comparison → projected facts → deterministic files → human interpretation → reviewed repository publication. Export neither runs the adapter nor runs the grader. `rescore` remains the separate explicit command that executes retained generated code.

Implementation touches `cmd/ateam/bench.go` dispatch/help; `internal/bench/experiment_run.go` only if a shared bounded parse-from-bytes seam is needed; new report projection/render/write helpers and tests in `internal/bench`; CLI tests; `benchmarks/README.md`; and a narrowly scoped offline report-check Taskfile entrypoint. Reuse `LoadExperimentRun`/`validateRun`, `CompareExperiment`, `ArmSummary`, existing stop/usage semantics and original/rescore identities. Do not alter grading, experiment schemas or historical comparison results to accommodate export.

## Proposed CLI

```text
ateam bench export <run.json> --output <new-directory> [--replay <rescore-run.json>]
ateam bench export <run.json> --check <existing-directory> [--replay <rescore-run.json>]
```

Require exactly one of `--output` and `--check`; no implicit destination, overwrite or publishing flag. Flags follow the artifact path, as in existing commands. Return zero only when the requested operation succeeds; invalid input, incompatible replay identity, write failure, cancellation or generated-file drift returns nonzero with a useful error. A valid report about a failed/partial experiment may export successfully: command success is not experiment success.

Load bounded original/replay bytes once, compute their file digests from those same bytes, and reuse canonical parsing/validation. Avoid a hash/read race or a second parser. Reject unsupported versions, malformed JSON, digest mismatch, impossible schedules and evaluator/native provenance failures through existing validation. No runtime executable probe, credential lookup, native process, network call or automatic regrading belongs in export.

## Output contract v1

Emit exactly three generated files: `results.json`, `results.md`, `evidence.json`. Every JSON document has an explicit kind discriminator and schema version so a summary cannot be mistaken for an original run artifact. The curated Mercury example's schema is an input to design, not a compatibility promise for this new contract.

`results.json` contains an explicit projection:

- Experiment mode, task/repetition schedule, requested model/runtime version identities, fixed limits and instruction/configuration/runner/evaluator/Go-driver digests. Omit full instructions, resolved settings, command lines and executable paths.
- Existing `Comparison` semantics, plus attempt rows and matched pair references required to explain them: task, arm, repetition, retained status, quality outcome, separate execution/grading duration, nullable primary/cache/reasoning usage and source digest. No raw source text or diagnostic text.
- Scheduled, retained, graded, failed, unevaluable and unstarted counts kept distinct. Include the stop reason and unpaired histories; no invented rows for unstarted attempts. Use the existing pair eligibility rules for time and token deltas and show each denominator.
- Accounting basis explicitly distinguished: captured provider-wire usage for validated Mercury; native CLI-reported observations for other native adapters; adapter-reported for command adapters. Missing usage remains null/unknown, a reported zero remains zero, and cached/reasoning counts remain subsets. Known subtotal includes known retained attempts, including failures, and remains null when none are known. Costs stay unknown in v1; add no pricing API or inferred invoice.
- Replay state described below. No generated winner, significance, equivalence, ROI, skill activation or overall-security verdict.

`results.md` renders those same projected facts, including question/treatment identity fields that can be expressed without exposing instruction text, every attempt/pair, failures and denominators, evidence availability and limitations. All free text is data: escape Markdown/HTML and control characters so model labels or other metadata cannot inject links, HTML, table rows or apparent conclusions. Source LOC or a dollar ROI score is not added as a new metric.

`evidence.json` binds the summary/report file digests to the original byte digest and internal run digest, optional replay digests, exporter schema and source identities. The exporter digest covers `report.go`, `report_markdown.go`, and `report_files.go`; it is not a whole-binary or transitive dependency attestation. It says original/replay evidence is not included; hashes do not prove authorship or independent execution. Do not copy original artifacts or emit absolute paths, native session IDs, credentials, prompts, responses, diagnostics, environment variables or account details. Evidence availability is scoped to the export: do not promise a private original will remain accessible elsewhere. Default omission is not a universal secret scrubber for arbitrary user-supplied metadata; human review is still required before publication.

Human interpretation, hypotheses and the next decision belong in a separate `interpretation.md` maintained by the owner/reviewer. Export/check never creates, overwrites, deletes, parses as instructions or verifies that file. Reports remain useful without it. Avoid timestamps or machine-local paths generated at export time; only recorded run time may be shown. Stable field/key/order/number formatting makes identical input bytes and options produce identical generated files.

## Retained replay evidence

Without `--replay`, say `not-provided`; artifact validation is not proof of regrading. With it, both artifacts must pass normal validation and bind to the same resolved experiment, original attempts/order, source/response/native envelopes, original execution timings and evaluator/Go-driver identity. No original inputs may be rewritten during verification.

Allow the changes that actual `rescore` makes: creation/digest metadata, grading durations/diagnostics and grading outcomes. If identities or execution evidence differ, refuse the claimed association before output creation. If validly associated grading outcomes differ, preserve original results and expose `different-results` with the affected attempt indexes; do not silently replace original grades or call the replay matched. `matched` means recorded outcomes agree, not cryptographic proof that another actor executed the grader. Check actual `RescoreExperiment` behavior while implementing; tests must exercise this distinction.

## Output and check safety

Render/validate all generated bytes before claiming a destination. The destination parent must be trusted and stable during initial directory creation/open: portable Go cannot atomically create a directory and return its handle, so replacement between `Mkdir` and the first identity capture is outside this guarantee. After identity capture, operations are anchored and subsequent replacement is detected. Exclusively create a previously absent destination directory; reject existing files/directories and symlink destinations, and anchor operations to the opened directory to prevent later path replacement from redirecting writes. Fixed filenames must never derive from task labels or arbitrary artifact paths. Write with exclusive creation and checked close errors; write the manifest last. A failure must not return success or overwrite/delete unrelated files. Retain an incomplete newly owned directory for diagnosis if safe rollback cannot be proved; incomplete/mismatched sets must fail check. Do not claim atomic publication or crash durability beyond the implemented guarantees.

Check mode only reads the three fixed generated files and recomputes expected bytes from the supplied original/replay snapshot. Refuse unsafe/symlinked expected files and report missing, changed, truncated or unsupported output. Compare both file contents and provenance, not merely a self-reported hash in the manifest. Ignore human interpretation and unrelated files. Check mode writes nothing and never repairs drift automatically.

## Repository workflow

1. Run and retain an explicitly authorized experiment under its frozen protocol.
2. Invoke existing `compare`/`rescore` offline as appropriate; export validated facts to a fresh local directory.
3. Review the projected fields, add human interpretation separately, and deliberately copy/commit the reviewed report into the experiment directory.
4. Use check mode in a repository gate wherever the original/replay artifacts are available. Public CI uses redistributable synthetic fixtures with known expected outcomes; it cannot rederive a real report whose raw evidence remains private. Such reports must state that verification availability rather than imply public reproducibility.
5. Review findings and decide on a new frozen protocol; export has no authority to schedule inference or modify guidance automatically.

Mercury's complete success case and Gemini's stopped/unknown-usage case motivate the acceptance fixtures. Never add their private raw traces to CI merely to make a check green. UI rendering and an experiment catalog can later consume this shared schema, without adding a second calculator in this slice.

## Verification plan

Test behavior before implementation with independent expected counts/pairs/totals and golden human-readable output. Cover complete A/A and A/B, quality failure, grading/adapter failure, zero retained attempts, stopped and unpaired histories, absent versus zero usage, partial provider observations excluded from totals, and cached/reasoning subsets. Show that every relevant assertion executes on boundary cases.

Replay tests distinguish absent, exact-match, changed grading outcome and mismatched source/config/evaluator/attempt identities. Projection/render tests put sentinel private strings in excluded fields, exercise hostile label/control/Markdown input and verify exported kinds cannot load as run artifacts. File tests cover existing destinations, symlinks/path replacement, partial write/close failure, cancellation, extra human files, concurrent creation with explicit barriers and read-only check behavior.

Generated checks are appropriate for rendering/label combinations and deterministic export/check laws. Keep a small test-owned reference model for directory histories (create, fail, retry to the same path, check, tamper); prove its transitions are reached. Use bounded waits and deterministic synchronization. Run meaningful overlays that collapse unknown usage to zero or remove an exclusion/escaping rule, replay identity validation or overwrite protection; record detection/survivor/invalid/inconclusive accurately. Attach test-to-spec references, run applicable Taskfile gates/full verify and obtain independent implementation review. No live model call or extra dependency is needed for these tests.
