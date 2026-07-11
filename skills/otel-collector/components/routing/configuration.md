# `routing`: configuration

## Typical config

```yaml
connectors:
  routing:
    error_mode: ignore
    default_pipelines: [logs/default]
    table:
      # context inferred from the qualified path — no `context` field needed
      - condition: resource.attributes["tenant"] == "acme"
        pipelines: [logs/acme]
      - condition: log.severity_number >= SEVERITY_NUMBER_ERROR
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
| `table` | array of RoutingTableItem | — | **yes** (≥1 route) | The routing table. Routes are evaluated in order; under the default `action: move` each piece of telemetry matches at most one route. |
| `default_pipelines` | array of pipeline IDs | none | no | Pipelines for telemetry that matches no route. **If unset, unmatched telemetry is dropped.** |
| `error_mode` | string | `propagate` | no | How OTTL evaluation errors are handled: `propagate`, `ignore`, or `silent` (see below). Defaults to `ignore` when the `connector.routing.defaultErrorModeIgnore` feature gate (alpha, since v0.155.0) is enabled — set it explicitly rather than relying on the default. |

## RoutingTableItem reference

| Key | Type | Default | Required | Meaning |
|-----|------|---------|----------|---------|
| `context` | string | inferred from path | no | OTTL context the expression runs in: `resource`, `span`, `metric`, `datapoint`, `log`, `otelcol`, or the deprecated `request`. Usually omit it — the context is inferred from context-qualified paths (see below). If set, it takes precedence over inference and must be valid for the pipeline's signal (see table below). |
| `statement` | string | — | one of `statement`/`condition` | OTTL statement in `route() where <expr>` form. **Mutually exclusive with `condition`.** Not allowed for the deprecated `request` context. |
| `condition` | string | — | one of `statement`/`condition` | OTTL boolean expression (no `route() where`). **Mutually exclusive with `statement`.** Required for the deprecated `request` context. |
| `action` | string | `move` | no | `move` (default) removes matched data from further route evaluation and `default_pipelines`; `copy` leaves it available to later routes and the default. |
| `pipelines` | array of pipeline IDs | — | **yes** (≥1) | Output pipelines this route forwards matching telemetry to. Listing several fans the data out to all of them. |

Exactly one of `statement` / `condition` must be set per route — setting both, or neither, fails validation.

## Context inference (recommended)

Since v0.156.0 the connector infers the `context` from **context-qualified paths** in the condition/statement, so the `context` field is usually unnecessary. Prefix each path with its context and drop the field:

```yaml
table:
  - condition: resource.attributes["env"] == "prod"     # -> resource context
    pipelines: [logs/prod]
  - condition: span.attributes["http.method"] == "GET"  # -> span context
    pipelines: [traces/http]
  - condition: log.severity_text == "ERROR"             # -> log context
    pipelines: [logs/errors]
```

The inferred (or explicit) context must be compatible with the pipeline signal — e.g. a `span` context only works in a traces pipeline. A condition with no OTTL paths (e.g. the literal `"true"`) can't be inferred, so it still needs an explicit `context`. Bare paths with an explicit `context` (`context: resource` + `attributes["env"]`) remain valid for backward compatibility.

## Supported contexts by signal

A route's context (inferred or explicit) must be supported by the signal of the pipeline it routes:

| Signal | Supported contexts |
|--------|--------------------|
| Traces | `resource`, `span`, `otelcol` (+ deprecated `request`) |
| Metrics | `resource`, `metric`, `datapoint`, `otelcol` (+ deprecated `request`) |
| Logs | `resource`, `log`, `otelcol` (+ deprecated `request`) |

`otelcol` (request metadata) is valid in every signal. Coarser contexts are cheaper: `resource` evaluates once per resource bundle, while `span`/`metric`/`log` evaluate per item and `datapoint` per data point. Use the coarsest context that expresses your condition.

## `statement` vs `condition`

Both express the same boolean test in OTTL; they differ only in syntax and which OTTL functions you can call:

```yaml
# condition: a bare boolean expression
- condition: resource.attributes["env"] == "prod"
  pipelines: [logs/prod]

# statement: route() where <expr> — lets you also run editors (e.g. delete_key)
#            against the data in the same step, before it is routed
- statement: route() where resource.attributes["env"] == "prod" and delete_key(resource.attributes, "internal.debug")
  pipelines: [logs/prod]
```

Use `condition` for a plain match (recommended for clarity); reach for `statement` only when you want to mutate the data in the same pass. Beyond the standard converters, the connector exposes only the `delete_key` and `delete_matching_keys` editors. For the full expression language — paths, converters, editors — see the `otel-ottl` skill.

## Request metadata routing (`otelcol.*`)

To route on HTTP headers or gRPC metadata, use the `otelcol.client.metadata` (HTTP/client) and `otelcol.grpc.metadata` (gRPC) paths. Metadata values are lists, so index the first element with `[0]`. These paths are valid in every signal context and, unlike the deprecated `request` context, support the full OTTL grammar (`and`/`or`, statements, etc.):

```yaml
- condition: otelcol.client.metadata["X-Tenant"][0] == "acme"
  pipelines: [logs/acme]
- condition: otelcol.grpc.metadata["x-tenant"][0] == "acme"
  pipelines: [logs/acme]
```

gRPC metadata keys are lowercased, so use lowercase keys for gRPC traffic. The receiver must propagate request metadata (the `otlp` receiver does).

### Deprecated `request` context

The `request` context is **deprecated as of v0.156.0** (a warning is logged when it is used) — prefer `otelcol.client.metadata` / `otelcol.grpc.metadata`. It requires `condition` (a `statement` is rejected) and supports only a single simple comparison — `==` or `!=`, no `and`/`or`:

```yaml
- context: request
  condition: request["X-Tenant"] == "acme"
  pipelines: [logs/acme]
```

## `error_mode`

Controls what happens when an OTTL expression fails to evaluate (missing attribute, type mismatch, etc.):

| Value | Behavior |
|-------|----------|
| `propagate` | The connector returns an error and the **payload is dropped** from the collector. |
| `ignore` | The error is logged and the payload is sent to `default_pipelines`. |
| `silent` | Same as `ignore`, but the error is not logged. |

Prefer `ignore` in production so a transient OTTL error doesn't drop data. Set `error_mode` explicitly in reusable configs because the default is `propagate` today but flips to `ignore` under the `connector.routing.defaultErrorModeIgnore` feature gate.

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
