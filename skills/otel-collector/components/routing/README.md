# `routing` connector

| | |
|-|-|
| Kind | connector |
| Type | `routing` |
| Signals | traces→traces (Alpha), metrics→metrics (Alpha), logs→logs (Alpha) — same-signal routing, no transformation |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/routingconnector> |

## Description

Routes telemetry to different pipelines based on OTTL conditions, keeping the **same signal type** (traces→traces, metrics→metrics, logs→logs) — it forwards data unchanged, it does not transform it. As a connector it is wired as the **exporter** of an input pipeline and the **receiver** of one or more output pipelines; the input pipeline hands its data to `routing`, which then dispatches it to the matching output pipelines.

The decision is driven by an ordered `table` of routes, each an OTTL `condition` (or a `route() where …` `statement`) evaluated in a `context`. Since v0.156.0 the context is **inferred** from context-qualified paths (`resource.attributes[...]`, `span.…`, `log.…`, `metric.…`, `datapoint.…`, `otelcol.…`), so the `context` field is usually omitted; the deprecated `request` context is the exception. Routes are evaluated **in order** and, under the default `action: move`, each piece of telemetry matches **at most one route** — to fan out to several pipelines, list them all under that route's `pipelines`, or use `action: copy` to keep matched data available to later routes. Telemetry that matches no route goes to `default_pipelines`, or is **dropped** if none is set. This replaces the `routingprocessor`, which has since been removed from contrib. The OTTL expression language itself lives in the `otel-ottl` skill; this page documents the connector's config surface.

## Main use-cases

Use when:
- You need to send telemetry to different backends/pipelines by attribute (tenant, region, environment, severity, service).
- You want a fan-out: the same data to multiple pipelines based on one condition.
- You need request-level routing from HTTP headers / gRPC metadata (multi-tenant isolation before parsing telemetry).
- You want to split high-priority from low-priority data (e.g. errors to expensive storage, the rest to cheap storage).

Avoid when:
- You only need to **drop** telemetry — use `filter`.
- You need to **modify** telemetry — use `transform`.
- You need to change signal type (e.g. spans→metrics) — that is what transforming connectors like `spanmetrics` are for.

## Related components

- `filter` — drops telemetry via OTTL; use it when the goal is to remove data, not redirect it.
- `transform` — mutates telemetry via OTTL; use it to rewrite data rather than route it.
- `otel-ottl` skill — the `condition`/`statement` expression language, contexts, and functions used in the routing `table`.
- `routingprocessor` — the removed processor this connector replaces (see [Known quirks](quirks.md) for migration).

## Details

- [Configuration](configuration.md) — full config tables for top-level keys and `table` items, context inference, supported contexts per signal, `otelcol.*` request-metadata routing, statement-vs-condition rules, `error_mode` semantics, and the connector pipeline wiring.
- [Verification](verification.md) — telemetrygen recipe that splits telemetry across two pipelines, each writing to its own `file` exporter, proving routing and the default fallback.
- [Advanced use-cases](advanced.md) — named instances, fan-out, multi-tenant `otelcol.*` routing, severity/environment routing, `action: copy` vs the default `move`, and migrating from `routingprocessor` / the removed `match_once`.
- [Known quirks](quirks.md) — dropped unmatched data, `error_mode` data loss, `move`-vs-`copy` defaults, mutual exclusivity, the deprecated `request` context, split Resource bundles, a validation-error→fix table, and stability caveats.
