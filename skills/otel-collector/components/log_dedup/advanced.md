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
      - log.severity_number >= SEVERITY_NUMBER_ERROR
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
      - 'log.attributes["log.type"] == "access"'
  log_dedup/health:
    interval: 30s
    conditions:
      - 'log.attributes["log.type"] == "health_check"'

service:
  pipelines:
    logs:
      processors: [memory_limiter, log_dedup/access, log_dedup/health]
```

## Multi-tenant buckets with `metadata_keys`

In gateway deployments where one pipeline carries logs for many tenants, aggregating across tenants would merge their counts and lose the routing metadata. `metadata_keys` partitions aggregation per unique combination of client-metadata values (gRPC/HTTP headers), and each emitted log keeps a context carrying its original metadata so a downstream `headers_setter` can re-attach the header.

```yaml
processors:
  log_dedup:
    interval: 30s
    metadata_keys: [x-scope-orgid]
    metadata_cardinality_limit: 1000
```

Always set `metadata_cardinality_limit` when using `metadata_keys` in production: `0` (the default) means the number of tenant buckets is unbounded and memory can grow without limit — the processor logs a startup warning in that case. When the limit is reached, logs with a new metadata combination are rejected with a permanent error rather than silently merged.

Requires the receiver to propagate client metadata (e.g. the OTLP receiver's `include_metadata: true`).

## OTTL context for `conditions`

`conditions` runs in the [OTTL log context](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottllog). Prefix paths with their context name: `log.body`, `log.attributes["…"]`, `resource.attributes["…"]`, `log.severity_number`, `log.severity_text`, plus all OTTL converters. Un-prefixed paths (`attributes["…"]`, `severity_number`) still work but are **deprecated** since v0.153.0 — the processor rewrites and logs them at startup, and support may be removed. See the `otel-ottl` skill for full path and function inventories.
