# Synthetic report examples

These redistributable run artifacts are test-owned evidence models. Their adapter,
Go-driver and runner identities are deliberately synthetic; no model or grader
executed to produce them. They must not be used as performance evidence.

The complete A/B schedules four attempts: baseline uses 30 primary tokens and
100 ns each; candidate uses 40 primary tokens and 130/1000 ns, with its second
attempt failing quality. The independent expectations are 3 passes/1 failure,
140 known tokens, one both-passed execution delta of +30 ns, and two known-usage
pair deltas totaling +20 tokens. Cache/reasoning counters are subsets. Every
retained source is 43 bytes, so the one both-passed pair reports 43 bytes per arm
in both examples.

The partial example retains three of four scheduled attempts before cancellation:
two passes followed by an adapter error; all three have unknown normalized usage.
The known subtotal is null, not zero. There is one pair and one unpaired attempt.

`TestReportWorkedExamples` regenerates facts from these inputs and compares all
three committed files byte for byte. Projection tests separately assert the
hand-calculated expectations above. Tests never refresh these golden files.
When behavior or the identified exporter sources change, export each original
into a fresh directory, inspect the changed facts and source digest, and copy only
the reviewed generated files here. Original input hashes and retained outcomes
must not be casually rewritten to make a failing check pass.

The inputs' `evaluator_sha256` and run `sha256` were rewritten once, in #111, when
`decision.go` joined the grader sources and `bench.go` gained the decision tier.
Only those two fields changed; outcomes, timings, usage and every other hash are
the original values. The outputs were re-exported and reviewed: the new evaluator
digest, the both-passed source-bytes line, and the dependent evidence digests.

They were rewritten the same way a second time in #111, after review of c8e0398
changed the grader (`propagate-limit` gate redesign and the `discipline_gate`
tag rename). Again only `evaluator_sha256` and the run `sha256` changed. The
re-exported outputs differ only in the evaluator digest and the dependent
evidence digests; these safe-path examples carry no discipline fields, so the
tag rename does not appear in them.
