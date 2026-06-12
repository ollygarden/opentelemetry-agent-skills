# `routing`: configuration

## Typical config

```yaml
connectors:
  routing:
    error_mode: ignore
    default_pipelines: [logs/default]
    table:
      - context: resource
        condition: attributes["tenant"] == "acme"
        pipelines: [logs/acme]
      - context: log
        condition: severity_number >= SEVERITY_NUMBER_ERROR
        pipelines: [logs/errors]

service:
  pipelines:
    logs/in:
      receivers: [otlp]
      exporters: [routing]            # connector as exporter of the input pipeline
    logs/acme:
      receivers: [routing]            # connector as receiver of an output pipeline
      exporters: [otlphttp/acme]
    logs/errors:
      receivers: [routing]
      exporters: [otlphttp/errors]
    logs/default:
      receivers: [routing]
      exporters: [otlphttp/default]
```

## Top-level configuration reference

| Key | Type | Default | Required | Meaning |
|-----|------|---------|----------|---------|
| `table` | array of RoutingTableItem | — | **yes** (≥1 route) | The routing table. Routes are evaluated in order; each piece of telemetry matches at most one route. |
| `default_pipelines` | array of pipeline IDs | none | no | Pipelines for telemetry that matches no route. **If unset, unmatched telemetry is dropped.** |
| `error_mode` | string | `propagate` | no | How OTTL evaluation errors are handled: `propagate`, `ignore`, or `silent` (see below). |

## RoutingTableItem reference

| Key | Type | Default | Required | Meaning |
|-----|------|---------|----------|---------|
| `context` | string | `resource` | no | OTTL context the expression runs in: `resource`, `span`, `metric`, `datapoint`, `log`, or `request`. Must be valid for the pipeline's signal (see table below). |
| `statement` | string | — | one of `statement`/`condition` | OTTL statement in `route() where <expr>` form. **Mutually exclusive with `condition`.** Not allowed for `request` context. |
| `condition` | string | — | one of `statement`/`condition` | OTTL boolean expression (no `route() where`). **Mutually exclusive with `statement`.** Required for `request` context. |
| `action` | string | `copy` | no | `copy` leaves matched data available to later routes and `default_pipelines`; `move` removes it from further evaluation. |
| `pipelines` | array of pipeline IDs | — | **yes** (≥1) | Output pipelines this route forwards matching telemetry to. Listing several fans the data out to all of them. |

Exactly one of `statement` / `condition` must be set per route — setting both, or neither, fails validation.

## Supported contexts by signal

A route's `context` must be supported by the signal of the pipeline it routes:

| Signal | Supported contexts |
|--------|--------------------|
| Traces | `resource`, `span`, `request` |
| Metrics | `resource`, `metric`, `datapoint`, `request` |
| Logs | `resource`, `log`, `request` |

Coarser contexts are cheaper: `resource` evaluates once per resource bundle, while `span`/`metric`/`log` evaluate per item and `datapoint` per data point. Use the coarsest context that expresses your condition. `request` evaluates once per incoming request, before telemetry is parsed.

## `statement` vs `condition`

Both express the same boolean test in OTTL; they differ only in syntax and which OTTL functions you can call:

```yaml
# condition: a bare boolean expression
- condition: attributes["env"] == "prod"
  pipelines: [logs/prod]

# statement: route() where <expr> — lets you also run editors (e.g. delete_key)
#            against the data in the same step, before it is routed
- statement: route() where attributes["env"] == "prod" and delete_key(attributes, "internal.debug")
  pipelines: [logs/prod]
```

Use `condition` for a plain match (recommended for clarity); reach for `statement` only when you want to mutate the data in the same pass. For the full expression language — paths, converters, editors — see the `otel-ottl` skill.

### `request` context (limited grammar)

`request` reads HTTP headers / gRPC metadata and requires `condition` (a `statement` is rejected). It supports only a single simple comparison — `==` or `!=` — with no `and`/`or`:

```yaml
- context: request
  condition: request["X-Tenant"] == "acme"
  pipelines: [logs/acme]
```

HTTP headers are matched case-insensitively; gRPC metadata keys are lowercased, so use lowercase keys when routing gRPC traffic (`request["x-tenant"]`). The receiver must propagate request metadata (the `otlp` receiver does).

## `error_mode`

Controls what happens when an OTTL expression fails to evaluate (missing attribute, type mismatch, etc.):

| Value | Behavior |
|-------|----------|
| `propagate` (default) | The connector returns an error and the **payload is dropped** from the collector. |
| `ignore` | The error is logged and the payload is sent to `default_pipelines`. |
| `silent` | Same as `ignore`, but the error is not logged. |

Prefer `ignore` in production so a transient OTTL error doesn't drop data.

## Pipeline wiring

`routing` is a connector: it bridges pipelines rather than living inside one. Wire it as the **exporter** of the input pipeline and the **receiver** of each output pipeline (including the default). Pipeline definition order does not matter — only that the connector name appears in both roles:

```yaml
service:
  pipelines:
    logs/in:                # input
      receivers: [otlp]
      exporters: [routing]
    logs/acme:              # output
      receivers: [routing]
      exporters: [otlphttp/acme]
    logs/default:           # default fallback output
      receivers: [routing]
      exporters: [otlphttp/default]
```

Every pipeline ID referenced in `table[].pipelines` and in `default_pipelines` must exist as a pipeline that lists `routing` as a receiver, otherwise the collector fails to start.
