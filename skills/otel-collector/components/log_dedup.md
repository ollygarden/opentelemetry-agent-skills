# `log_dedup` processor

| | |
|-|-|
| Kind | processor |
| Signals | logs |
| Stability | Alpha (processor); Development (telemetry metric `otelcol_dedup_processor_aggregated_logs`) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/logdedupprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/logdedupprocessor> |

**Rename in v0.151.0:** the type was renamed `logdedup` → `log_dedup` to match snake_case. The legacy name is a deprecated alias. New configs should use `log_dedup`; existing `logdedup` configs keep working.

## Description

Aggregates identical log records over a time window and emits a single log with a `log_count` attribute. Instead of forwarding every occurrence of a repeated message, the processor counts duplicates seen within an `interval` and emits one representative record per unique combination at the end of the window.

### What gets matched

Two log records are considered identical when **all** of these match exactly: resource attributes, instrumentation scope (name, version, attributes), log body (content and type), log attributes, and severity (number and text). The processor stores a 64-bit hash per unique combination.

`include_fields` / `exclude_fields` customize which parts of the record feed the hash.

### What gets emitted

At the end of each `interval`, one log per unique hash is emitted with:
- The original body, attributes, severity, resource.
- `log_count` (or the configured name) — duplicate count, int.
- `first_observed_timestamp` — RFC3339 string in the configured timezone.
- `last_observed_timestamp` — RFC3339 string in the configured timezone.
- `Timestamp` and `ObservedTimestamp` set to **emission time** (not original log time).

Counter resets after emission.

## Main use-cases

Use it when:
- The pipeline carries high-volume repetitive logs (health checks, polling, retry storms, connection errors).
- You want to reduce backend storage cost while preserving frequency information.
- You care about first/last observed timestamps of a recurring event, not every individual occurrence.

Avoid it when:
- Every entry must be preserved (audit, compliance, security logs).
- Sub-second precision per occurrence is required.
- The backend already deduplicates — you'll deduplicate twice.
- Each log carries a unique identifier or timestamp that makes every record unique (use `exclude_fields` to fix this, or skip the processor).

## Typical config

```yaml
processors:
  log_dedup:

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [memory_limiter, log_dedup]
      exporters: [otlphttp]
```

### Configuration reference

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `interval` | duration | `10s` | Aggregation window. Must be `> 0`. Counter resets after each emission. |
| `log_count_attribute` | string | `log_count` | Attribute name for the duplicate count. Must be non-empty. |
| `timezone` | string | `UTC` | IANA timezone for `first_observed_timestamp`/`last_observed_timestamp` (e.g., `America/New_York`). |
| `conditions` | []string | `[]` | OTTL log-context expressions. Empty = deduplicate all logs. Non-empty = only deduplicate matches; non-matching logs pass through unchanged. |
| `include_fields` | []string | `[]` | Whitelist of fields used to compute log identity. Mutually exclusive with `exclude_fields`. |
| `exclude_fields` | []string | `[]` | Fields removed before hashing **and** stripped from emitted logs. Mutually exclusive with `include_fields`. |

### Validation rules

- `include_fields` and `exclude_fields` cannot both be set.
- Field paths must start with `body.` or `attributes.`.
- The entire `body` cannot be included or excluded — only nested fields when `body` is a map.
- Use `.` as the nesting delimiter. Escape literal dots in keys with `\` (e.g., `attributes.host\.name`).
- `timezone` must resolve in the IANA database.

### Fallback behavior

If `include_fields` is set but none of the listed fields exist on a record, the processor falls back to comparing all fields (body, attributes, severity).

## Verification

`log_dedup` ships in the `contrib` and `k8s` distributions, so a stock contrib collector can run this.

Config (`dedup-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  log_dedup:
    interval: 5s
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [log_dedup]
      exporters: [debug]
```

Generate many identical log records (see the `otel-telemetrygen` skill):

```bash
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 50 --body "health check ok" --duration 4s
```

**What proves it worked:** the `debug` exporter prints **one** aggregated log per 5s interval carrying a `log_count` attribute near 50 (plus `first_observed_timestamp` / `last_observed_timestamp`), not 50 separate records.

## Advanced use-cases

### Exclude volatile fields

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

### Deduplicate only errors

```yaml
processors:
  log_dedup:
    interval: 60s
    conditions:
      - severity_number >= SEVERITY_NUMBER_ERROR
```

INFO/WARN/DEBUG logs pass through immediately. Only ERROR+ are buffered.

### Whitelist match keys

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

### Per-source named instances

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

### OTTL context for `conditions`

`conditions` runs in the [OTTL log context](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl/contexts/ottllog): `body`, `attributes["…"]`, `resource.attributes["…"]`, `severity_number`, `severity_text`, plus all OTTL converters. See the `otel-ottl` skill for full path and function inventories.

## Known quirks

### Memory model

- One entry per unique hash for the duration of the interval.
- High-cardinality fields (`request_id`, `trace_id`, timestamps) blow up memory — exclude them or use `conditions` to scope what's deduplicated.
- Long intervals accumulate more unique entries; short intervals emit faster with smaller buffers.

### Telemetry

The processor emits a histogram `otelcol_dedup_processor_aggregated_logs` with the count of logs aggregated per emission. Useful for tuning. The metric is currently **Development** stability — the name or shape can change between releases.

### Troubleshooting

**No logs are deduplicated.**
- `conditions` may exclude everything — comment them out to confirm.
- Every record may have a unique identifier (`request_id`, body timestamp). Add those to `exclude_fields`.
- Records may arrive slower than `interval` allows duplicates to form — increase `interval`.

**Memory grows without bound.**
- High-cardinality unique fields in the hash input — narrow with `exclude_fields` or `include_fields`.
- Interval too long for the volume — reduce it.
- Scope what's deduplicated with `conditions`.

**Logs arrive in bursts matching the interval.**
- Expected. The processor emits at interval boundaries. Reduce `interval` for lower latency, or move repetitive logs into a dedicated pipeline so other logs aren't held back.

**Different resources/severities aren't being grouped.**
- Resource attributes and severity are always part of the hash — there is no option to ignore them. Use a `resource` or `transform` processor upstream to normalize them first if that's the goal.

### Validation errors

| Error | Fix |
|-------|-----|
| `cannot define both exclude_fields and include_fields` | Choose one. |
| `an excludefield must start with body or attributes` | Prefix every path with `body.` or `attributes.`. |
| `cannot exclude the entire body` | Only nested fields are exclud-able (e.g., `body.timestamp`). |

### Anti-patterns

**Indiscriminate dedup.**

```yaml
# BAD — critical events get hidden behind log_count
processors:
  log_dedup:
    interval: 60s
```

Scope with `conditions` so audit, security, and one-off error logs pass through individually:

```yaml
processors:
  log_dedup:
    conditions:
      - 'attributes["log.type"] == "health_check"'
      - 'attributes["log.type"] == "heartbeat"'
```

**`include_fields` that includes unique IDs.**

```yaml
# BAD — request_id makes every record unique; nothing dedupes
processors:
  log_dedup:
    include_fields:
      - attributes.error_code
      - attributes.request_id
```

Include only identifying fields, never per-record identifiers.

**Very long intervals.**

```yaml
# BAD — 30 minutes of in-memory buffering, large bursty emissions
processors:
  log_dedup:
    interval: 30m
```

Prefer shorter intervals plus aggressive `exclude_fields` for cardinality control.

**Forgetting timezone for ops teams.**

```yaml
# Confusing if the team operates in PST
processors:
  log_dedup:
    timezone: UTC  # default
```

Set the timezone your operators read timestamps in.

## Related components

- `groupbyattrs` — groups by attribute, does not deduplicate.
- `transform` — can rewrite/strip fields via OTTL, does not aggregate.
- `filter` — drops records, does not aggregate.
