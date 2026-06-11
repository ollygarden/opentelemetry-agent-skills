# `filter`: advanced use-cases

## Combining conditions

Conditions within a context are ORed — any match drops the item. Use `and`/`or`/`not()` and parentheses inside a single condition for AND logic:

```yaml
processors:
  filter:
    error_mode: ignore
    traces:
      span:
        # OR across the list — drop either name
        - 'name == "readiness"'
        - 'name == "liveness"'
        # AND within one condition — short, successful spans only
        - '(end_time - start_time) < Duration("10ms") and status.code == STATUS_CODE_OK'
```

## error_mode

```yaml
# Development: catch broken conditions immediately
processors:
  filter:
    error_mode: propagate   # a failing condition drops the whole batch

# Production: a bad condition logs but does not drop unrelated data
processors:
  filter:
    error_mode: ignore

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
    metrics:
      metric:
        - 'IsMatch(name, "internal\\..*")'
        - 'type == METRIC_DATA_TYPE_HISTOGRAM'
        # metric-only converters (must run in the metric context)
        - 'HasAttrKeyOnDatapoint("internal")'
        - 'HasAttrOnDatapoint("environment", "test")'
      datapoint:
        - 'metric.name == "k8s.pod.phase" and value_int == 4'
```

Filter at `metric` level when you want to drop the whole series — it is cheaper than evaluating every datapoint. See [Known quirks](quirks.md) for the drop-last-datapoint rule.

## Filtering by resource attribute

Resource attributes are visible in every context, so the same condition shape works across signals — useful for stripping an environment or service wholesale:

```yaml
processors:
  filter:
    error_mode: ignore
    traces:
      span:
        - 'resource.attributes["deployment.environment"] == "dev"'
    metrics:
      metric:
        - 'resource.attributes["deployment.environment"] == "dev"'
    logs:
      log_record:
        - 'resource.attributes["deployment.environment"] == "dev"'
```

## OTTL function usage

The processor exposes the standard OTTL converters (see the `otel-ottl` skill) plus the two metric-only functions above:

```yaml
processors:
  filter:
    error_mode: ignore
    traces:
      span:
        - 'IsMatch(resource.attributes["k8s.pod.name"], "canary-.*")'
    logs:
      log_record:
        - 'IsMatch(body, ".*(password|secret|api[_-]?key).*")'
        - 'attributes["http.request.method"] == nil'
```

## Named instances

```yaml
processors:
  filter/health:
    error_mode: ignore
    traces:
      span:
        - 'name == "health_check"'
  filter/debug-logs:
    error_mode: ignore
    logs:
      log_record:
        - 'severity_number < SEVERITY_NUMBER_INFO'

service:
  pipelines:
    traces:
      processors: [memory_limiter, filter/health]
    logs:
      processors: [memory_limiter, filter/debug-logs]
```

## OTTL context

The condition fields evaluate in their per-signal contexts (span, spanevent, metric, datapoint, log). Path and converter inventories live in the `otel-ottl` skill; the field summary per context is in [configuration.md](configuration.md).
