# `routing`: advanced use-cases

Examples below use context-qualified paths (`resource.attributes[...]`, `span.…`, `log.…`), the recommended style since v0.156.0 — the `context` field is inferred and omitted. Bare paths with an explicit `context` still work for backward compatibility.

## Named instances for independent dimensions

Under the default `action: move`, each piece of telemetry matches at most one route, so a single `routing` instance can't route on two independent dimensions at once. Use one named instance per dimension and list both as exporters of the input pipeline — each gets an independent copy of the data:

```yaml
connectors:
  routing/tenant:
    table:
      - condition: resource.attributes["tenant"] == "acme"
        pipelines: [logs/acme]
      - condition: resource.attributes["tenant"] == "ecorp"
        pipelines: [logs/ecorp]
  routing/region:
    table:
      - condition: resource.attributes["region"] == "us-east"
        pipelines: [logs/east]
      - condition: resource.attributes["region"] == "us-west"
        pipelines: [logs/west]

service:
  pipelines:
    logs/in:
      receivers: [otlp]
      exporters: [routing/tenant, routing/region]   # both evaluate independently
```

## Fan-out to multiple pipelines in one route

To send the same matched telemetry to several pipelines, list them all under one route — telemetry is copied to each:

```yaml
table:
  - condition: span.end_time_unix_nano - span.start_time_unix_nano > 5000000000   # > 5s
    pipelines: [traces/slow, traces/all]       # slow traces go to BOTH
```

## Multi-tenant routing on request metadata

Route on HTTP headers or gRPC metadata using the `otelcol.client.metadata` (HTTP/client) and `otelcol.grpc.metadata` (gRPC) paths — valid in every signal context. Metadata values are lists, so index with `[0]`:

```yaml
connectors:
  routing:
    default_pipelines: [logs/default]
    table:
      - condition: otelcol.client.metadata["X-Tenant"][0] == "acme"
        pipelines: [logs/acme]
      - condition: otelcol.client.metadata["X-Tenant"][0] == "ecorp"
        pipelines: [logs/ecorp]
```

For gRPC traffic use lowercase keys: `otelcol.grpc.metadata["x-tenant"][0]`. The older `request["X-Tenant"]` / `context: request` form still works but is **deprecated as of v0.156.0** (see [quirks](quirks.md)).

## Severity and environment routing

```yaml
# logs: cheap storage for low severity, expensive for errors
table:
  - condition: log.severity_number < SEVERITY_NUMBER_ERROR
    pipelines: [logs/cheap]
  - condition: log.severity_number >= SEVERITY_NUMBER_ERROR
    pipelines: [logs/important]
```

```yaml
# traces: production to one backend, everything else to the default
connectors:
  routing:
    default_pipelines: [traces/dev]
    table:
      - condition: resource.attributes["deployment.environment"] == "production"
        pipelines: [traces/prod]
```

Order routes from specific to general — the first match wins, so a broad route placed first will shadow a narrower one below it.

## `action: copy` vs the default `move`

`move` (the default) removes matched data from any further evaluation, so it never also reaches later routes or `default_pipelines`. `copy` leaves matched data available downstream — use it for an "archive everything, then route" pattern where the same data must hit both the archive route and a more specific one:

```yaml
table:
  - condition: resource.attributes["archive"] == "true"
    action: copy                       # copied to archive, still evaluated below
    pipelines: [logs/archive]
  - condition: log.severity_number >= SEVERITY_NUMBER_ERROR
    pipelines: [logs/errors]           # default move: errors leave here
```

Because `move` is already the default, you rarely set it explicitly; reach for `copy` only when a match must not be terminal.

## Migrating from `routingprocessor`

The connector replaces the `routingprocessor`, which has since been removed from contrib entirely — the OLD config below no longer loads on current releases. The processor sat inside one pipeline and routed to **exporters** via a `from_attribute` + `value` table; the connector bridges pipelines and routes to **pipelines** via OTTL.

```yaml
# OLD (routingprocessor)
processors:
  routing:
    from_attribute: X-Tenant
    table:
      - value: acme
        exporters: [otlp/acme]
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [routing]
      exporters: [otlp/acme]
```

```yaml
# NEW (routing connector)
connectors:
  routing:
    default_pipelines: [logs/default]
    table:
      - condition: otelcol.client.metadata["X-Tenant"][0] == "acme"
        pipelines: [logs/acme]
service:
  pipelines:
    logs/in:   {receivers: [otlp], exporters: [routing]}
    logs/acme: {receivers: [routing], exporters: [otlp/acme]}
    logs/default: {receivers: [routing], exporters: [otlp/default]}
```

## Migrating from `match_once`

`match_once` was removed in v0.120.0; each item now matches at most one route (under the default `move`). To reproduce the old `match_once: false` (multi-match) behavior:

- **Parallel routers** — split independent dimensions into separate named instances (see above). Each gets its own copy, so an item can match in both. Simplest when no `default_pipelines` is needed.
- **Enumerate combinations** — for a small fixed set, write one route per combination and list all target pipelines in it (e.g. `condition: resource.attributes["env"] == "prod" and resource.attributes["region"] == "east"` → `pipelines: [logs/prod, logs/east]`). Practical only for a handful of dimensions.

Prefer parallel routers; enumeration grows combinatorially.
