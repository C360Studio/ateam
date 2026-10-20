# Proposed Mercury canary and A/A — $1 spending target

**Not authorized for paid execution.** After PR #77 review/merge, obtain explicit approval for this two-stage protocol. Use the existing `ateam bench` CLI, no new run controller.

| Stage | Configuration | Maximum attempts | Accounted-token stop | Wall time |
| --- | --- | ---: | ---: | ---: |
| Canary | `examples/mercury-canary.json`: safe-path, identical arms | 2 | 50,000 | 180 seconds |
| A/A, only after canary review | `examples/mercury-aa.json`: all three single-file tasks, identical arms | 6 | 50,000 | 360 seconds |

Both stages use native Claude2.1.292, LiteLLM1.99.0, `mercury-3`, low effort, 30 seconds per attempt, requested8192 output tokens/request and at most four captured model requests/attempt. No retries/fallback, cap increases or replacement attempts. Keep instructions, permissions and controls identical across arms. Freeze the resolved config/executable/hook hashes before each stage. Run the offline self-test and inspect dry-run output first.

The combined spending **target is $1**, not an enforced billing cap. At the provider's published list prices checked October20,2026 ($0.20/M input and $0.70/M output),100,000 accounted tokens would cost at most about$0.07 if all were output. Do not rely on launch discounts. [Inception's Mercury3 announcement](https://www.inceptionlabs.ai/blog/introducing-mercury-3) is the pricing source; verify current account pricing before authorization.

The token stop is checked between attempts and can overshoot within the last attempt. Missing/failed usage can leave unknown charges. Timeouts close the local bridge connection but do not establish remote computation cancellation or invoice finality. An account-enforced spending limit, if available and independently configured, is separate from this runner; none is claimed here.

Advance from canary only when both attempts retain complete provider usage, expected model/controls, valid structured source and replayable artifacts, with no unexpected model requests or runner failures. Inspect grading results separately from transport correctness. Stop on any execution/accounting failure and reconcile provider dashboard charges before proceeding. A/A is a pipeline/calibration check, not an instruction-effectiveness result. Retain every failure and compare only completed pairs; no A/B or cross-runtime claim follows automatically.
