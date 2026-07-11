# `filter`: advanced use-cases

## Combining conditions

Conditions within a list are ORed — any match drops the item. Use `and`/`or`/`not()` and parentheses inside a single condition for AND logic:

```yaml
processors:
  filter:
    error_mode: ignore
    trace_conditions:
      # OR across the list — drop either name
      - 'span.name == "readiness"'
      - 'span.name == "liveness"'
      # AND within one condition — short, successful spans only
      - '(span.end_time - span.start_time) < Duration("10ms") and span.status.code == STATUS_CODE_OK'
```

## Advanced style: explicit context and per-group error_mode

Each entry in a `*_conditions` list can be an object that pins a `context` and/or overrides `error_mode` for its group of `conditions`. Set `context` only when inference can't resolve it — e.g. a bare `IsRootSpan()` with no path prefix:

```yaml
processors:
  filter:
    error_mode: ignore
    trace_conditions:
      - context: span            # required: IsRootSpan() has no prefix to infer from
        error_mode: propagate    # override just this group
        conditions:
          - IsRootSpan()
      - conditions:
          - spanevent.name == "grpc.timeout"
```

Basic (flat strings) and advanced (objects) styles cannot be mixed within one signal.

## error_mode

```yaml
# Default: ignore — a bad condition logs but does not drop unrelated data
processors:
  filter:
    error_mode: ignore

# Development: catch broken conditions immediately (must set explicitly; ignore is the default)
processors:
  filter:
    error_mode: propagate   # a failing condition drops the whole batch

# Noisy environments: continue without logging the error at all
processors:
  filter:
    error_mode: silent
```

## Metric and datapoint filtering

Drop whole metrics by name or type (cheapest), or individual datapoints by value/attribute:

```yaml
processors:
  filter:
    error_mode: ignore
    metric_conditions:
      - 'IsMatch(metric.name, "internal\\..*")'
      - 'metric.type == METRIC_DATA_TYPE_HISTOGRAM'
      # metric-only converters (inferred as metric context)
      - 'HasAttrKeyOnDatapoint("internal")'
      - 'HasAttrOnDatapoint("environment", "test")'
      - 'metric.name == "k8s.pod.phase" and datapoint.value_int == 4'
```

Filter at `metric` level when you want to drop the whole series — it is cheaper than evaluating every datapoint. See [Known quirks](quirks.md) for the drop-last-datapoint rule.

## Filtering by resource attribute

Resource attributes are visible in every signal, so the same condition shape works across them — useful for stripping an environment or service wholesale:

```yaml
processors:
  filter:
    error_mode: ignore
    trace_conditions:
      - 'resource.attributes["deployment.environment"] == "dev"'
    metric_conditions:
      - 'resource.attributes["deployment.environment"] == "dev"'
    log_conditions:
      - 'resource.attributes["deployment.environment"] == "dev"'
```

## OTTL function usage

The processor exposes the standard OTTL converters (see the `otel-ottl` skill) plus the two metric-only functions above:

```yaml
processors:
  filter:
    error_mode: ignore
    trace_conditions:
      - 'IsMatch(resource.attributes["k8s.pod.name"], "canary-.*")'
    log_conditions:
      - 'IsMatch(log.body, ".*(password|secret|api[_-]?key).*")'
      - 'log.attributes["http.request.method"] == nil'
```

## Named instances

```yaml
processors:
  filter/health:
    error_mode: ignore
    trace_conditions:
      - 'span.name == "health_check"'
  filter/debug-logs:
    error_mode: ignore
    log_conditions:
      - 'log.severity_number < SEVERITY_NUMBER_INFO'

service:
  pipelines:
    traces:
      processors: [memory_limiter, filter/health]
    logs:
      processors: [memory_limiter, filter/debug-logs]
```

## OTTL context

The `*_conditions` lists infer their context (`resource`/`scope`/`span`/`spanevent`, `metric`/`datapoint`, `log`, `profile`) from the path prefixes used; the filter processor does **not** expose an `exemplar` context. Path and converter inventories live in the `otel-ottl` skill; the field summary per context is in [configuration.md](configuration.md).
