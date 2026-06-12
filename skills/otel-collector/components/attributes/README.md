# `attributes` processor

| | |
|-|-|
| Kind | processor |
| Type | `attributes` |
| Signals | traces (Beta), logs (Beta), metrics (Beta) |
| Distributions | core, contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/attributesprocessor> |

## Description

Modifies the **attributes** of spans, log records, and metric data points through an ordered list of `actions`. Each action names an attribute `key` and one of `insert`, `update`, `upsert`, `delete`, `hash`, `extract`, or `convert`; actions run in the order listed, so later actions see the effect of earlier ones. Values can be literal (`value`), copied from another attribute (`from_attribute`), or pulled from request context (`from_context` — client IP, transport metadata, or authenticator data).

An optional `include`/`exclude` matching block scopes the processor to a subset of telemetry (by service, span name, attribute, resource attribute, instrumentation library, log body, log severity, metric name, …) so only matching records are touched. This processor operates on **span/log/metric-datapoint attributes only** — it does **not** touch resource attributes; use the `resource` processor for those.

## Main use-cases

Use it when:
- You want to standardize, backfill, or rename attribute keys across services (`insert` defaults, `upsert` to copy with `from_attribute`).
- You need to redact or hash sensitive attribute values (`hash` PII, `delete` credentials) by key or regex `pattern`.
- You want to extract structured fields from a string attribute (`extract` with named capture groups) or coerce types (`convert`).
- You want to inject request context — client IP, headers, authenticated subject — onto telemetry (`from_context`).

Avoid it when:
- You need to edit **resource** attributes — use `resource` (same action set, resource scope).
- You need arbitrary conditional logic, math, or cross-field transforms — use `transform` (OTTL is a superset).
- You only need to drop records — use `filter`.
- You need a fail-closed allow list for egress governance — use `redaction`.

## Related components

- `resource` — same `actions` model but applied to **resource** attributes instead of span/log/datapoint attributes.
- `transform` — OTTL-based mutation; a superset of what `attributes` can do, at the cost of more verbosity.
- `redaction` — allow/block-list masking with a fail-closed allow list and URL/DB sanitizers.
- `filter` — drops whole records that match a condition (run `attributes` before it to shape the matchable attributes).

## Details

- [Configuration](configuration.md) — the `actions` list (every action type and its `key`/`value`/`pattern`/`from_attribute`/`from_context`/`converted_type` fields), the `include`/`exclude` matching block, and context-value sources.
- [Verification](verification.md) — telemetrygen recipe that adds/overwrites an attribute and confirms it in `debug` output.
- [Advanced use-cases](advanced.md) — include/exclude scoping, hashing PII, regex `extract`, type conversion, ordered-action pipelines, named instances, and combining with `resource`/`filter`.
- [Known quirks](quirks.md) — action order, the resource-attribute confusion, `insert`/`update`/`upsert` semantics, `extract` overwrites, regexp performance, metric identity-conflict caveats, and stability.
