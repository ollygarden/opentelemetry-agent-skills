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

Drops telemetry — spans, span events, metrics, datapoints, log records (and, in development, profiles) — from a pipeline when it matches an OTTL condition. Conditions are evaluated per item; when any condition evaluates to `true`, that item is dropped. Conditions in a list are ORed together. Filtering is hierarchical: if a higher-level item is dropped (e.g. a span), the lower-level conditions for it (e.g. its span events) are not evaluated.

The current surface is the per-signal `trace_conditions` / `metric_conditions` / `log_conditions` / `profile_conditions` lists, which carry OTTL conditions with context-prefixed paths (`span.name`, `log.body`, …) and infer their context automatically. The top-level `error_mode` controls what happens when a condition fails to evaluate — `ignore` (default since v0.153.0) logs and continues, `propagate` fails the batch, `silent` continues without logging. Two older forms are deprecated: the fixed-context per-signal blocks (`traces.span`, `metrics.metric`, `logs.log_record`, …) and a pre-OTTL include/exclude matcher (metrics and logs only) that cannot be combined with OTTL conditions for the same signal.

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

- [Configuration](configuration.md) — `error_mode`, the `*_conditions` lists with their inferred contexts (`resource`/`scope`/`span`/`spanevent`, `metric`/`datapoint`, `log`, `profile`), basic vs advanced style, and the deprecated per-signal blocks and include/exclude matchers.
- [Verification](verification.md) — telemetrygen recipe that drops a subset of logs by severity so the drop is observable.
- [Advanced use-cases](advanced.md) — combining conditions, `error_mode`, metric/datapoint filtering, resource-attribute filters, OTTL functions, named instances.
- [Known quirks](quirks.md) — orphaned telemetry, the drop-last-datapoint rule, error-mode behavior, stability caveats, anti-patterns, troubleshooting.
