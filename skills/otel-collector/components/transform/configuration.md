# `transform`: configuration

## Typical config

```yaml
processors:
  transform:
    error_mode: ignore
    log_statements:
      - set(log.attributes["env"], "prod")

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, transform]
      exporters: [otlphttp]
```

## Top-level keys

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `error_mode` | string | `ignore` | How OTTL statement-evaluation errors are handled: `ignore`, `silent`, or `propagate`. See [Error mode](#error-mode). |
| `trace_statements` | list | `[]` | Statements run against trace data. |
| `metric_statements` | list | `[]` | Statements run against metric data. |
| `log_statements` | list | `[]` | Statements run against log data. |
| `profile_statements` | list | `[]` | Statements run against profile data (Development stability). |

> The default `error_mode` is `ignore`, set by the `processor.transform.defaultErrorModeIgnore` feature gate, which reached **beta (enabled by default) in v0.153.0**. To restore the old `propagate` default, disable it: `--feature-gates=-processor.transform.defaultErrorModeIgnore`.

> A `flatten_data` boolean (default `false`, behind the `transform.flatten.logs` alpha feature gate) gives each log record a distinct copy of its resource and scope before transformation, then regroups afterwards — useful when log-level data drives resource/scope edits. It copies and hashes every record, so enable only when needed.

## Error mode

| Mode | Behavior | When |
|------|----------|------|
| `ignore` | Logs the error and moves to the next statement. | **Default** (since v0.153.0) and **recommended in production** — one bad statement won't drop unrelated data. |
| `silent` | Continues without logging. | When error logging is too noisy. |
| `propagate` | Returns the error up the pipeline, dropping the whole payload. | Development/testing — surfaces config mistakes immediately. Was the default before v0.153.0. |

The top-level `error_mode` can be overridden per statement group (see below).

## Statement groups

Each `*_statements` field accepts entries in two styles. **Do not mix the two styles within one signal's list.**

### Flat (inferred-context) form

The recommended form for most cases: a plain list of OTTL statements. The processor **infers the context** from the path prefixes in each statement (e.g. `log.body` → `log` context; `resource.attributes` alone → `resource` context). You write statements without declaring a context.

```yaml
transform:
  error_mode: ignore
  trace_statements:
    - keep_keys(span.attributes, ["service.name", "http.request.method", "http.route"])
    - set(resource.attributes["deployment.environment.name"], "production")
  log_statements:
    - set(log.severity_text, "FAIL") where log.body == "request failed"
```

### Explicit-group form

For complex cases, group related statements and attach options. Each group is an object with an optional `context`, an optional group-level `error_mode`, optional `conditions`, and a required `statements` list.

```yaml
transform:
  error_mode: propagate           # default for all groups
  metric_statements:
    - context: metric             # optional; inferred if omitted
      error_mode: ignore          # optional; overrides top-level for this group
      conditions:                 # optional; OR'd together
        - metric.type == METRIC_DATA_TYPE_SUM
      statements:                 # required
        - set(metric.description, "Sum metric")
        - convert_sum_to_gauge() where metric.name == "system.processes.count"
```

| Group key | Type | Required | Notes |
|-----------|------|----------|-------|
| `context` | string | No | Explicit OTTL context (see [Context values](#context-values)). Usually inferred — only set it to override inference. |
| `error_mode` | string | No | Overrides the top-level `error_mode` for this group only. |
| `conditions` | list | No | Group-level OTTL booleans, OR'd together. If any is true (or the list is empty), the group's statements run. |
| `statements` | list | Yes | The OTTL statements to execute, in order. |

## Context values

Context determines which paths the statements can reach. Contexts are hierarchical — a more specific context can also read the broader ones above it. The processor picks the **most specific context** that covers all paths in a group; set `context:` explicitly only when inference can't reconcile the paths.

| Signal | Contexts (broad → specific) | Reaches |
|--------|------------------------------|---------|
| traces | `resource`, `scope`, `span`, `spanevent` | `spanevent` can read `resource`, `scope`, `span`, `spanevent` |
| metrics | `resource`, `scope`, `metric`, `datapoint`, `exemplar` | `exemplar` can read `resource`, `scope`, `metric`, `datapoint`, `exemplar` |
| logs | `resource`, `scope`, `log` | `log` can read `resource`, `scope`, `log` |
| profiles | `resource`, `scope`, `profile` | `profile` can read `resource`, `scope`, `profile` (Development) |

The `exemplar` metric context (reaching a datapoint's exemplars) was added in v0.156.0; on older versions `metric_statements` stops at `datapoint`.

Some path/function combinations can't share a context (e.g. a metric-context-only function alongside a `datapoint` path). When inference fails, split the conflicting statements into separate groups. See [Known quirks](quirks.md).

## `conditions` vs `where`

- `conditions` (group-level) gate the **whole group**; the entries are OR'd. If none match, no statement in the group runs.
- `where` (per statement) gates a **single statement**; it is AND'd with the group's conditions.

```yaml
metric_statements:
  - conditions:
      - metric.type == METRIC_DATA_TYPE_SUM        # group runs only for Sum metrics
    statements:
      - set(metric.description, "count") where metric.name == "http.requests"  # AND name matches
```

## Statement ordering and short-circuiting

Statements run top-to-bottom; later statements observe the effects of earlier ones (so set-then-delete order matters). With `error_mode: propagate`, the first failing statement **short-circuits the rest of the payload** — remaining statements never run and the batch is dropped. With `ignore`/`silent`, a failing statement is skipped and the next one still runs.

The OTTL expression surface — functions, paths, editors, converters, grammar, enums — is covered in the `otel-ottl` skill and is not reproduced here.
