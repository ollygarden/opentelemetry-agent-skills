# `filter` processor

| | |
|-|-|
| Kind | processor |
| Type | `filter` |
| Signals | traces (Alpha), metrics (Alpha), logs (Alpha), profiles (Development) |
| Distributions | core, contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/filterprocessor> |

## Description

Drops telemetry — spans, span events, metrics, datapoints, log records (and, in development, profiles) — from a pipeline when it matches an OTTL condition or a legacy metric-name/attribute matcher. Conditions are evaluated per item; when any condition for a given context evaluates to `true`, that item is dropped. Multiple conditions in the same context are ORed together. Filtering is hierarchical: if a higher-level item is dropped (e.g. a span), the lower-level conditions for it (e.g. its span events) are not evaluated.

OTTL is the modern, recommended approach; the per-signal blocks (`traces`, `metrics`, `logs`) carry the condition lists. The top-level `error_mode` controls what happens when a condition fails to evaluate — `propagate` (default) fails the batch, `ignore` logs and continues, `silent` continues without logging. A legacy include/exclude form survives for metrics and logs but cannot be combined with OTTL conditions for the same signal.

## Main use-cases

Use it when:
- You want to drop noise outright — health-check spans, debug/trace logs, internal or test-environment metrics.
- You need to reduce backend volume by removing low-value signals at the collector.
- You want to strip telemetry by resource attribute (environment, service, pod) before it leaves the pipeline.

Avoid it when:
- You want to **modify** rather than drop telemetry — use `transform`.
- You need a content-aware keep/drop decision over a **whole trace** (errors, latency) — use `tail_sampling`.
- Dropping would orphan child spans or break trace/log correlation (see [Known quirks](quirks.md)).

## Related components

- `transform` — rewrites telemetry via OTTL instead of dropping it; same expression language.
- `attributes` / `resource` — add or modify the attributes that filter conditions match on (run before `filter`).
- `tail_sampling` — trace-wide, content-aware keep/drop sampling for traces.
- `probabilistic_sampler` — stateless head sampling by trace ID.
- `routing` — sends telemetry to different pipelines by condition rather than dropping it.

## Details

- [Configuration](configuration.md) — `error_mode`, the per-signal OTTL contexts (`traces.span`, `traces.spanevent`, `metrics.metric`, `metrics.datapoint`, `logs.log_record`), and the legacy include/exclude form.
- [Verification](verification.md) — telemetrygen recipe that drops a subset of logs by severity so the drop is observable.
- [Advanced use-cases](advanced.md) — combining conditions, `error_mode`, metric/datapoint filtering, resource-attribute filters, OTTL functions, named instances.
- [Known quirks](quirks.md) — orphaned telemetry, the drop-last-datapoint rule, error-mode behavior, stability caveats, anti-patterns, troubleshooting.
