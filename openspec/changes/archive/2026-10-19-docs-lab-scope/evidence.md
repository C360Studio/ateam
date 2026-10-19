# Documentation evidence

## Scope

Only prose, generated instruction strings, and this OpenSpec change are modified. No runtime branches or algorithms changed; mutation/PBT additions are not applicable. Existing bootstrap parity and full repository verification cover template integration. The author reviewed the diff; independent approval is not claimed.

## Non-Go adoption probe

Built the changed CLI with `go build -o .enjoy-logs/docs-probe/ateam ./cmd/ateam`. Used a disposable Git repository under `.enjoy-logs/docs-probe/python-project` with no Go module, a Python file containing `# shortcut: python limit, revisit later`, and a TypeScript file containing `// shortcut: typescript limit, revisit later`.

The probe asserted these outcomes (all passed):

| Command/check | Observed result |
| --- | --- |
| `ateam init` | Exit 0; generated instructions include the adoption scope; no Taskfile created. |
| `ateam doctor --check` | Exit 0 despite absence of project verification tasks and a Go module. |
| `ateam state debt` | Exit 0; exactly one result, `app.ts`; Python marker silently omitted. |
| `ateam task list` without Taskfile | Exit 1, error names missing `Taskfile.yml`. |
| `ateam mutate -pkg . -test TestExample -file app.py -mutant <external-copy> -expect-text wrong` | Exit 2, non-test `.go` refusal before test execution. |
| Add a Taskfile `verify` target running `python3 -c 'print(42)'`; run `ateam task verify` | Exit 0, output includes 42. |

These checks establish specific boundaries, not comprehensive Python support or integration with live agent CLIs. The extension allowlist, mode behavior, benchmark behavior, and dependency claims were also traced to their linked source implementations in the adoption guide.

## Verification record

Exact tested commit, commands, exit statuses, and final preflight results are recorded in PR #5. Documentation links were checked locally and the author reviewed the diff for scope and consistency. Concurrent UI changes in the original checkout were neither copied nor edited; integration must retain those changes.
