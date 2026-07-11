# `transform` processor

| | |
|-|-|
| Kind | processor |
| Type | `transform` |
| Signals | traces (Beta), metrics (Beta), logs (Beta), profiles (Development) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor> |

## Description

Mutates telemetry — spans, span events, metrics, datapoints, log records (and, in development, profiles) — **in place** by running OTTL statements against it. You supply a per-signal list of statements (`trace_statements`, `metric_statements`, `log_statements`); each statement executes sequentially against incoming data, optionally guarded by a `where` clause. Unlike `filter`, which drops items, `transform` rewrites them: set, delete, rename, redact, convert types, parse structured fields, and more.

The top-level `error_mode` controls what happens when a statement fails to evaluate — `ignore` (default) logs and continues to the next statement, `silent` continues without logging, `propagate` drops the whole payload. (The default became `ignore` in v0.153.0, when the `processor.transform.defaultErrorModeIgnore` gate reached beta / on-by-default; earlier versions defaulted to `propagate`.) Statement groups run in the order written, so later statements see the effects of earlier ones. The OTTL language itself (functions, paths, editors/converters, grammar) lives in the `otel-ottl` skill; this page documents only the processor's config surface.

## Main use-cases

Use it when:
- You need to **modify** telemetry — rename or set attributes, set defaults for missing fields, normalize formats across sources.
- You want to redact sensitive values (passwords, tokens, PII) in place rather than dropping the record.
- You need metric reshaping — convert sum↔gauge, extract count/sum/percentile from histograms, scale or aggregate datapoints.
- You want to parse structured data out of an unstructured log body into attributes.

Avoid it when:
- You only need to **drop** whole items — use `filter` (it complements `transform`).
- You need a whole-trace keep/drop decision — use `tail_sampling`.
- You need cross-batch metric aggregation over time — use `interval` or a metrics connector.

## Related components

- `filter` — drops telemetry via the same OTTL language; pair it with `transform` when some items should be removed and others rewritten.
- `attributes` — simpler key/value attribute edits without OTTL, for straightforward add/update/delete.
- `resource` — resource-attribute edits specifically.
- `metricstransform` — name/label-oriented metric renaming and aggregation outside OTTL.
- `routing` — sends telemetry to different pipelines by condition instead of mutating it.

## Details

- [Configuration](configuration.md) — `error_mode`, the statement-group structure, `context` values per signal, the inferred-context form, group-level `conditions`, and statement ordering/short-circuiting.
- [Verification](verification.md) — telemetrygen recipe that sets an attribute on logs so the change is observable in `debug` output.
- [Advanced use-cases](advanced.md) — multiple statement groups, scoping with conditions, context selection trade-offs, combining with `filter`, error_mode for resilience, named instances, common transforms.
- [Known quirks](quirks.md) — statement ordering, shared-resource mutation, propagate dropping the batch, context performance, OTTL version drift, stability caveats, anti-patterns.
