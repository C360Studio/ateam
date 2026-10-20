# Design

Gemini CLI 0.43.0 passes generateContentConfig to the native SDK. Use responseMimeType application/json and responseJsonSchema requiring exactly one string source property. An explicit output_format json-schema config field distinguishes new execution from legacy artifacts; absence must regenerate identical old settings and remain replayable. Strict source parsing and failed-request accounting rules remain unchanged.

Outcome-derived stop reasons precede missing telemetry for genuine timeout, cancellation and adapter failures. Runner-generated identity refusal retains the model-specific reason. Loading old artifacts must still accept their original evidence-backed reasons without rewriting them. A duplicate session ID is detectable before grading and must not execute the generated candidate.

Risks: schema with tools is a preview capability and needs a native canary; provider retries can still make usage incomplete; diagnostic wording must not accidentally classify operational errors as identity changes. Existing failed Gemini runs stay immutable and separate. Shared helper renaming/provider dispatch consolidation and version-policy tables remain nonblocking review follow-ups before a fourth provider or contract refresh; this correction avoids expanding the already large PR into a structural refactor.

References: https://geminicli.com/docs/cli/generation-settings/ and https://ai.google.dev/gemini-api/docs/generate-content/structured-output . Installed CLI and SDK call paths were inspected independently of the live result.
