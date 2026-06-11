# `filter`: configuration

## Typical config

```yaml
processors:
  filter:
    error_mode: ignore
    traces:
      span:
        - 'name == "health_check"'

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, filter]
      exporters: [otlphttp]
```

## Top-level keys

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `error_mode` | string | `propagate` | How OTTL condition-evaluation errors are handled: `propagate`, `ignore`, or `silent`. See [Error mode](#error-mode). |
| `traces` | block | — | Trace conditions (`span`, `spanevent`). |
| `metrics` | block | — | Metric conditions (`metric`, `datapoint`) or legacy `include`/`exclude`. |
| `logs` | block | — | Log conditions (`log_record`) or legacy `include`/`exclude`. |

> A `processor.filter.defaultErrorModeIgnore` feature gate (alpha, disabled by default, v0.150.0+) flips the default `error_mode` from `propagate` to `ignore`. Until it is enabled, the default is `propagate`.

## Per-signal OTTL conditions

Each field below takes a list of OTTL boolean expressions. An item is **dropped** when any condition in its context evaluates to `true`. The expression language itself — operators, paths, converter functions — is covered in the `otel-ottl` skill; only the filter-specific surface is described here.

| Field | OTTL context | Drops | Representative fields |
|-------|--------------|-------|-----------------------|
| `traces.span` | span | a span | `name`, `attributes[...]`, `status.code`, `kind`, `start_time`, `end_time`, `resource.attributes[...]`, `instrumentation_scope` |
| `traces.spanevent` | spanevent | a span event | `name`, `attributes[...]`, `time_unix_nano`, plus the enclosing span's fields |
| `metrics.metric` | metric | a whole metric | `name`, `description`, `unit`, `type`, `aggregation_temporality`, `resource.attributes[...]` |
| `metrics.datapoint` | datapoint | a single datapoint | `attributes[...]`, `value_int`, `value_double`, `time_unix_nano`, `metric.name`, `metric.type` |
| `logs.log_record` | log | a log record | `body`, `attributes[...]`, `severity_number`, `severity_text`, `resource.attributes[...]`, `instrumentation_scope` |

**OTTL-context note:** each field evaluates in its own context. You cannot reference span fields from a `spanevent` condition's top level except through the span context the spanevent inherits, and metric-level fields (`name`, `type`) are reachable from `datapoint` only via `metric.name` / `metric.type`. Two metric-only converters are added by this processor and must run in the `metrics.metric` context: `HasAttrKeyOnDatapoint(key)` (true if any datapoint carries that attribute key) and `HasAttrOnDatapoint(key, value)` (true if any datapoint has that string key/value).

Hierarchy: if a span is dropped, its `spanevent` conditions are not evaluated; if all datapoints of a metric are dropped, the metric is removed too. See [Known quirks](quirks.md).

> Two surfaces are intentionally not detailed here: an **inferred-context** condition form (`trace_conditions` / `metric_conditions` / `log_conditions`, v0.146.0+) that lets a single list span multiple contexts, and a **`profiles.profile`** block (Development stability). Both are documented in the [upstream README](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/filterprocessor) if you need them.

## Error mode

| Mode | Behavior | When |
|------|----------|------|
| `propagate` | Returns the error up the pipeline, dropping the whole payload. | **Default.** Development/testing — surfaces config mistakes immediately. |
| `ignore` | Logs the error and moves to the next condition. | **Recommended in production** — a bad condition won't drop unrelated data. |
| `silent` | Continues without logging. | When error logging is too noisy. |

## Legacy include/exclude (metrics and logs only)

The pre-OTTL form still works but **cannot be combined with OTTL conditions for the same signal**. Prefer OTTL for new configs.

Metrics — match by name and/or resource attribute:

```yaml
metrics:
  include:
    match_type: strict   # or regexp
    metric_names:
      - "system.cpu.utilization"
    resource_attributes:
      - key: "env"
        value: "production"
  exclude:
    match_type: regexp
    metric_names:
      - "internal\\..*"
```

Logs — match by severity, body, or attribute:

```yaml
logs:
  include:
    match_type: strict   # or regexp
    severity_number:
      min: "INFO"
      match_undefined: true
    bodies:
      - ".*error.*"
    record_attributes:
      - key: "user_id"
        value: ".*"
```

Semantics: `include` alone keeps only matches; `exclude` alone drops matches; both means `include` is applied first, then `exclude`. `match_type` is `strict` (exact) or `regexp`. For metrics, a `regexp` block (`cacheenabled`, `cachemaxnumentries`) tunes the regex cache when `match_type: regexp`.
