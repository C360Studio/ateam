# Stable Gemini with explicit best-effort accounting

## Why

The owner revised #38 from complete fatal billing evidence to a bounded stable-CLI upgrade with honest partial accounting. Stable0.64.0 still omits stats on fatal JSON exits, so a large workaround would not provide a trustworthy ledger.

## What changes

Require explicit Gemini cli_version0.64.0 for new execution, preserve omitted/0.43.0 historical replay, revalidate native isolation/output contracts, and expose known normalized token subtotals with attempt coverage. Update examples and docs. Claim #38 under #16; PR32/43 remain separate.

## Stop boundary

Verified implementation and independent review in a PR, no merge or paid calls. No telemetry export, global installation/config changes, turn/model/tool expansion, invented counters or weakened stop rules. Native authentication, failure billing and backend settings are not proven by offline fixtures.
