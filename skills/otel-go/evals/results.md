# `go-v145-upgrade-review` harness results

Run with Codex CLI 0.147.0 driving `gpt-5.6-sol` through ChatGPT
authentication. Every retained repetition used a fresh isolated `HOME` and `CODEX_HOME`,
read-only sandbox, identical prompt and tool access, and a five-minute orchestration ceiling.
Codex reported no marginal API billing.

The deterministic grader marked a repetition passing only when all four expectations in
`evals.json` were present in its final answer. The selected repetitions are the earliest valid
fresh repetitions for each arm and were not chosen by outcome.

| Arm | Selected attempts | Pass | Fail | Unknown |
|---|---|---:|---:|---:|
| Target skill withheld | `withheld-4`, `withheld-5`, `withheld-6` | 0 | 3 | 0 |
| Current `origin/main` skill | `current-1`, `current-2`, `current-5` | 1 | 2 | 0 |
| Proposed skill | `proposed-14`, `proposed-15`, `proposed-16` | 3 | 0 | 0 |

## Per-attempt grading

`A` is the log `attribute.Value` migration, `B` is the explicit OTLP HTTP trace path,
`C` is corrected `Retry-After` behavior, and `D` is safe local verification.

| Attempt | A | B | C | D | Result |
|---|---|---|---|---|---|
| `withheld-4` | pass | pass | pass | fail | fail |
| `withheld-5` | pass | pass | pass | fail | fail |
| `withheld-6` | pass | pass | pass | fail | fail |
| `current-1` | pass | pass | pass | pass | pass |
| `current-2` | pass | pass | pass | fail | fail |
| `current-5` | pass | pass | pass | fail | fail |
| `proposed-14` | pass | pass | pass | pass | pass |
| `proposed-15` | pass | pass | pass | pass | pass |
| `proposed-16` | pass | pass | pass | pass | pass |

The three factual upgrade expectations were saturated in every arm. The measurable lift was
safe verification: the proposed skill consistently concluded with `go mod tidy -diff`,
`go build ./...`, `go test ./...`, and a disposable local receiver for exporter behavior.

## Preserved invalid and superseded attempts

- `withheld-1`: unknown; the JSONL transcript ended before a final answer.
- `withheld-2`: invalid; harness initialization failed before a model turn due temporary quota.
- `current-3`: invalid; no final answer was retained.
- `current-4`: invalid; harness initialization failed before a model turn due temporary quota.
- `proposed-2`, `proposed-11`, and `proposed-13`: invalid; temporary quota interrupted output before a final answer.
- `proposed-1`, `proposed-3`, `proposed-4`, `proposed-6`, `proposed-7`, `proposed-8`,
  `proposed-9`, `proposed-10`, and `proposed-12`:
  valid results against superseded proposed revisions, retained during iteration but excluded
  because they do not describe the final head.
- `proposed-5`: invalidated because the revised skill copy was not loaded.

Raw harness transcripts contained local absolute paths and runtime internals, so this sanitized
public summary replaces them. The final answers were reviewed to confirm that they contained no
credentials, customer data, or private repository content.
