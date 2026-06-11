# `log_dedup`: configuration

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

## Configuration reference

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `interval` | duration | `10s` | Aggregation window. Must be `> 0`. Counter resets after each emission. |
| `log_count_attribute` | string | `log_count` | Attribute name for the duplicate count. Must be non-empty. |
| `timezone` | string | `UTC` | IANA timezone for `first_observed_timestamp`/`last_observed_timestamp` (e.g., `America/New_York`). |
| `conditions` | []string | `[]` | OTTL log-context expressions. Empty = deduplicate all logs. Non-empty = only deduplicate matches; non-matching logs pass through unchanged. |
| `include_fields` | []string | `[]` | Whitelist of fields used to compute log identity. Mutually exclusive with `exclude_fields`. |
| `exclude_fields` | []string | `[]` | Fields removed before hashing **and** stripped from emitted logs. Mutually exclusive with `include_fields`. |

## Validation rules

- `include_fields` and `exclude_fields` cannot both be set.
- Field paths must start with `body.` or `attributes.`.
- The entire `body` cannot be included or excluded — only nested fields when `body` is a map.
- Use `.` as the nesting delimiter. Escape literal dots in keys with `\` (e.g., `attributes.host\.name`).
- `timezone` must resolve in the IANA database.

## Fallback behavior

If `include_fields` is set but none of the listed fields exist on a record, the processor falls back to comparing all fields (body, attributes, severity).

## What gets matched

Two log records are considered identical when **all** of these match exactly: resource attributes, instrumentation scope (name, version, attributes), log body (content and type), log attributes, and severity (number and text). The processor stores a 64-bit hash per unique combination.

`include_fields` / `exclude_fields` customize which parts of the record feed the hash.

## What gets emitted

At the end of each `interval`, one log per unique hash is emitted with:
- The original body, attributes, severity, resource.
- `log_count` (or the configured name) — duplicate count, int.
- `first_observed_timestamp` — RFC3339 string in the configured timezone.
- `last_observed_timestamp` — RFC3339 string in the configured timezone.
- `Timestamp` and `ObservedTimestamp` set to **emission time** (not original log time).

Counter resets after emission.
