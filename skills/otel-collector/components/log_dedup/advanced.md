# `log_dedup`: advanced use-cases

## Exclude volatile fields

```yaml
processors:
  log_dedup:
    interval: 30s
    exclude_fields:
      - body.timestamp
      - attributes.request_id
      - attributes.trace_id
      - attributes.span_id
```

Records that differ only in those fields are treated as duplicates.

## Deduplicate only errors

```yaml
processors:
  log_dedup:
    interval: 60s
    conditions:
      - severity_number >= SEVERITY_NUMBER_ERROR
```

INFO/WARN/DEBUG logs pass through immediately. Only ERROR+ are buffered.

## Whitelist match keys

```yaml
processors:
  log_dedup:
    interval: 60s
    include_fields:
      - attributes.error_code
      - attributes.service.name
      - body.error_message
```

Groups all errors with the same code from the same service, regardless of every other attribute.

## Per-source named instances

```yaml
processors:
  log_dedup/access:
    interval: 10s
    conditions:
      - 'attributes["log.type"] == "access"'
  log_dedup/health:
    interval: 30s
    conditions:
      - 'attributes["log.type"] == "health_check"'

service:
  pipelines:
    logs:
      processors: [memory_limiter, log_dedup/access, log_dedup/health]
```

## OTTL context for `conditions`

`conditions` runs in the [OTTL log context](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottllog): `body`, `attributes["…"]`, `resource.attributes["…"]`, `severity_number`, `severity_text`, plus all OTTL converters. See the `otel-ottl` skill for full path and function inventories.
