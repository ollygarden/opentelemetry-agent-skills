# `routing`: advanced use-cases

## Named instances for independent dimensions

Each piece of telemetry matches at most one route, so a single `routing` instance can't route on two independent dimensions at once. Use one named instance per dimension and list both as exporters of the input pipeline — each gets an independent copy of the data:

```yaml
connectors:
  routing/tenant:
    table:
      - condition: attributes["tenant"] == "acme"
        pipelines: [logs/acme]
      - condition: attributes["tenant"] == "ecorp"
        pipelines: [logs/ecorp]
  routing/region:
    table:
      - condition: attributes["region"] == "us-east"
        pipelines: [logs/east]
      - condition: attributes["region"] == "us-west"
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
  - context: span
    condition: duration > 5000000000          # > 5s
    pipelines: [traces/slow, traces/all]       # slow traces go to BOTH
```

## Multi-tenant routing on request metadata

`request` context routes once per incoming request, before telemetry is parsed — ideal for tenant isolation from an HTTP header or gRPC metadata. It requires `condition` and supports only `==`/`!=`:

```yaml
connectors:
  routing:
    default_pipelines: [logs/default]
    table:
      - context: request
        condition: request["X-Tenant"] == "acme"
        pipelines: [logs/acme]
      - context: request
        condition: request["X-Tenant"] == "ecorp"
        pipelines: [logs/ecorp]
```

HTTP headers match case-insensitively; for gRPC metadata use lowercase keys (`request["x-tenant"]`).

## Severity and environment routing

```yaml
# logs: cheap storage for low severity, expensive for errors
table:
  - context: log
    condition: severity_number < SEVERITY_NUMBER_ERROR
    pipelines: [logs/cheap]
  - context: log
    condition: severity_number >= SEVERITY_NUMBER_ERROR
    pipelines: [logs/important]
```

```yaml
# traces: production to one backend, everything else to the default
connectors:
  routing:
    default_pipelines: [traces/dev]
    table:
      - context: resource
        condition: attributes["deployment.environment"] == "production"
        pipelines: [traces/prod]
```

Order routes from specific to general — the first match wins, so a broad route placed first will shadow a narrower one below it.

## `action: move` vs `copy`

`copy` (default) leaves matched data available to later routes and `default_pipelines`; `move` removes it from any further evaluation. Use `move` when a match is terminal and the data must not also reach the default:

```yaml
table:
  - context: log
    condition: severity_number >= SEVERITY_NUMBER_ERROR
    action: move                       # errors leave here, never hit default_pipelines
    pipelines: [logs/errors]
```

## Migrating from `routingprocessor`

The connector replaces the deprecated `routingprocessor`. The processor sat inside one pipeline and routed to **exporters** via a `from_attribute` + `value` table; the connector bridges pipelines and routes to **pipelines** via OTTL.

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
      - context: resource
        condition: attributes["X-Tenant"] == "acme"
        pipelines: [logs/acme]
service:
  pipelines:
    logs/in:   {receivers: [otlp], exporters: [routing]}
    logs/acme: {receivers: [routing], exporters: [otlp/acme]}
    logs/default: {receivers: [routing], exporters: [otlp/default]}
```

## Migrating from `match_once`

`match_once` was removed in v0.120.0; each item now matches at most one route. To reproduce the old `match_once: false` (multi-match) behavior:

- **Parallel routers** — split independent dimensions into separate named instances (see above). Each gets its own copy, so an item can match in both. Simplest when no `default_pipelines` is needed.
- **Enumerate combinations** — for a small fixed set, write one route per combination and list all target pipelines in it (e.g. `condition: env=="prod" and region=="east"` → `pipelines: [logs/prod, logs/east]`). Practical only for a handful of dimensions.

Prefer parallel routers; enumeration grows combinatorially.
