## MODIFIED Requirements

### Requirement: Host launch manifest
When `.agents/hosts/<host>.toml` exists for the detected host, ateam SHALL read its `launch` and `notify` argument arrays and substitute the launch placeholders inside each argument; an action may instead hold `tool` text, expanded the same way and printed, never run, for a host with no one-command form. A missing manifest SHALL change no output other than `launch: printed`; a malformed one SHALL fail with fixed text that does not echo the file.

#### Scenario: Detection order
- **WHEN** no `ATEAM_HOST` is set
- **THEN** the host is `t3code` when `T3CODE_THREAD_ID` is set, else `herdr` when `HERDR_ENV=1`, else `enjoy` when `ENJOY_MCP_ENDPOINT` is set, else `cli`

#### Scenario: Placeholders
- **WHEN** an argument holds `{pr}`, `{prompt}`, `{sha}`, `{agent}`, `{model}`, `{effort}`, or `{return_to}`
- **THEN** each is replaced by the launch spec's value

#### Scenario: Override wins
- **WHEN** `ATEAM_HOST=herdr` and `T3CODE_THREAD_ID` are both set
- **THEN** the detected host is `herdr`

#### Scenario: Placeholder inside an argument
- **WHEN** a `launch` argument is `--prompt={prompt}`
- **THEN** the argument becomes `--prompt=/review-pr <n>` as one argument, with no shell splitting

#### Scenario: Malformed manifest
- **WHEN** the manifest is not valid TOML or `launch` is not an array of strings
- **THEN** `ateam review request` exits non-zero with a fixed-text reason

#### Scenario: Shipped host manifests name the selected reviewer run
- **WHEN** `.agents/hosts/enjoy.toml`, `.agents/hosts/t3code.toml`, or `.agents/hosts/herdr.toml` expands `[launch] tool`
- **THEN** the reviewer prompt names `{agent}`, `{model}`, and `{effort}` and tells the reviewer to use them in `reviewed-by:`
