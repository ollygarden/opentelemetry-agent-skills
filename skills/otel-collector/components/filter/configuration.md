# `filter`: configuration

## Typical config

```yaml
processors:
  filter:
    error_mode: ignore
    trace_conditions:
      - span.name == "health_check"

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
| `error_mode` | string | `ignore` | How OTTL condition-evaluation errors are handled: `propagate`, `ignore`, or `silent`. See [Error mode](#error-mode). |
| `trace_conditions` | list | — | Trace conditions (contexts: `resource`, `scope`, `span`, `spanevent`). |
| `metric_conditions` | list | — | Metric conditions (contexts: `resource`, `scope`, `metric`, `datapoint`). |
| `log_conditions` | list | — | Log conditions (contexts: `resource`, `scope`, `log`). |
| `profile_conditions` | list | — | Profile conditions (contexts: `resource`, `scope`, `profile`). **Development** stability. |
| `traces` / `metrics` / `logs` / `profiles` | block | — | **Deprecated** per-signal blocks and legacy `include`/`exclude` matchers. See [Legacy configuration](#legacy-configuration). |

> **Default `error_mode` is `ignore` since v0.153.0.** The `processor.filter.defaultErrorModeIgnore` feature gate (**beta, enabled by default**) sets the default to `ignore`. Disable it — `--feature-gates=-processor.filter.defaultErrorModeIgnore` — to restore the old `propagate` default.

## The `*_conditions` fields

`trace_conditions`, `metric_conditions`, `log_conditions`, and `profile_conditions` are the current, recommended surface (documented upstream from v0.146.0). Each takes a list of OTTL boolean conditions; an item is **dropped** when any condition evaluates to `true` (conditions are ORed). The expression language — operators, paths, converter functions — is covered in the `otel-ottl` skill; only the filter-specific surface is described here.

### Contexts and path prefixes

Within a `*_conditions` list, paths are **prefixed by context** and the processor **infers** the context from the prefixes used, so you rarely set it by hand.

| Field | Available contexts | Representative paths |
|-------|--------------------|----------------------|
| `trace_conditions` | `resource`, `scope`, `span`, `spanevent` | `span.name`, `span.attributes[...]`, `span.status.code`, `span.kind`, `span.start_time`, `span.end_time`, `spanevent.name`, `resource.attributes[...]`, `scope.name` |
| `metric_conditions` | `resource`, `scope`, `metric`, `datapoint` | `metric.name`, `metric.type`, `metric.unit`, `datapoint.attributes[...]`, `datapoint.value_int`, `datapoint.value_double`, `resource.attributes[...]` |
| `log_conditions` | `resource`, `scope`, `log` | `log.body`, `log.attributes[...]`, `log.severity_number`, `log.severity_text`, `resource.attributes[...]`, `scope.name` |
| `profile_conditions` | `resource`, `scope`, `profile` | `profile.duration_unix_nano`, `resource.attributes[...]` |

The filter processor supports **only** the contexts above — notably **no `exemplar` context** (that path exists in the `transform` processor, which mutates rather than drops).

**Hierarchy.** Conditions run higher-to-lower (`resource` → `scope` → signal-specific). If a higher-level item is dropped, lower-level conditions for it are skipped: a dropped span skips its `spanevent` conditions. If all datapoints of a metric are dropped, the metric is dropped too; if all span events of a span are dropped, the span is left intact. When one condition mixes paths from two contexts, it is evaluated in the **lower** context (e.g. `resource` + `spanevent` → evaluated per span event). See [Known quirks](quirks.md).

**Metric-only converters.** This processor adds two functions that must run in the **`metric`** context: `HasAttrKeyOnDatapoint(key)` (true if any datapoint carries that attribute key) and `HasAttrOnDatapoint(key, value)` (true if any datapoint has that string key/value). Because they carry no path prefix, their context **cannot be inferred** — pin `context: metric` explicitly (advanced style), or validation fails with `unable to infer context from conditions`.

### Basic vs advanced style

**Basic** — a flat list of OTTL strings, ORed, context inferred per condition:

```yaml
filter:
  error_mode: ignore
  trace_conditions:
    - span.attributes["container.name"] == "app_container_1"
    - resource.attributes["host.name"] == "localhost"
    - span.name == "app_3"
```

**Advanced** — objects that pin a `context` and/or override `error_mode` for a group of `conditions`:

```yaml
filter:
  error_mode: ignore
  trace_conditions:
    - context: span            # set explicitly only when inference can't (e.g. IsRootSpan())
      error_mode: propagate    # overrides the top-level error_mode for this group
      conditions:
        - IsRootSpan()
    - conditions:
        - spanevent.name == "grpc.timeout"
```

Set `context` explicitly only when a condition has no path prefix to infer from (e.g. a bare `IsRootSpan()`), or when a combination of paths/functions/enums is not allowed in a single inferred context. Basic and advanced (and the deprecated forms) **cannot be mixed within one signal** — validation rejects mixing configuration styles.

## Error mode

| Mode | Behavior | When |
|------|----------|------|
| `ignore` | Logs the error and moves to the next condition. | **Default and recommended** — a bad condition won't drop unrelated data. |
| `silent` | Continues without logging. | When error logging is too noisy. |
| `propagate` | Returns the error up the pipeline, dropping the whole payload. | Development/testing — surfaces config mistakes immediately. |

## Legacy configuration

Two older forms still work but are **deprecated** and slated for removal; both are documented upstream only for versions before v0.146.0. Prefer `*_conditions` for new configs.

**Deprecated per-signal OTTL blocks** — same conditions, unprefixed paths inside a fixed context. Migrate by moving each into the matching `*_conditions` list with the context prefix:

| Deprecated | Migrate to |
|------------|------------|
| `traces.resource` / `traces.span` / `traces.spanevent` | `trace_conditions` with `resource.` / `span.` / `spanevent.` prefix |
| `metrics.resource` / `metrics.metric` / `metrics.datapoint` | `metric_conditions` with `resource.` / `metric.` / `datapoint.` prefix |
| `logs.resource` / `logs.log_record` | `log_conditions` with `resource.` / `log.` prefix |
| `profiles.resource` / `profiles.profile` | `profile_conditions` with `resource.` / `profile.` prefix |

**Pre-OTTL include/exclude** (metrics and logs only) — name/attribute/severity matchers. Cannot be combined with any OTTL conditions for the same signal.

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
