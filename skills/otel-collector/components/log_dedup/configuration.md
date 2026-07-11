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
| `conditions` | []string | `[]` | OTTL log-context expressions. Empty = deduplicate all logs. Non-empty = only deduplicate logs matching at least one condition; non-matching logs pass through unchanged. Prefix paths with their context name (`log.attributes["x"]`, `resource.attributes["y"]`, `log.severity_number`). |
| `include_fields` | []string | `[]` | Whitelist of fields used to compute log identity. Mutually exclusive with `exclude_fields`. |
| `exclude_fields` | []string | `[]` | Fields removed before hashing **and** stripped from emitted logs. Mutually exclusive with `include_fields`. |
| `metadata_keys` | []string | `[]` | Client-metadata keys (e.g. gRPC/HTTP headers like `x-scope-orgid`) that partition aggregation into separate buckets. Logs with different values for these keys are counted independently, and each emitted log keeps a context carrying its original metadata so downstream extensions (e.g. `headers_setter`) can route it. Case-insensitive; duplicates rejected. Empty = one shared bucket. Added in v0.155.0. |
| `metadata_cardinality_limit` | uint32 | `0` | Caps the number of distinct `metadata_keys` combinations tracked at once. `0` = unbounded (a startup warning is logged when `metadata_keys` is set but this stays `0`). Once the cap is hit, new combinations are rejected with a permanent error. Added in v0.155.0. |

## Validation rules

- `include_fields` and `exclude_fields` cannot both be set.
- Field paths must start with `body.` or `attributes.`.
- The entire `body` cannot be included or excluded — only nested fields when `body` is a map.
- Use `.` as the nesting delimiter. Escape literal dots in keys with `\` (e.g., `attributes.host\.name`).
- Duplicate entries in `include_fields`, `exclude_fields`, or `metadata_keys` (case-insensitive for the latter) are rejected.
- `timezone` must resolve in the IANA database.

## Fallback behavior

If `include_fields` is set but none of the listed fields exist on a record, the processor falls back to comparing all fields (body, attributes, severity).

## What gets matched

Two log records are considered identical when **all** of these match exactly: resource attributes, instrumentation scope (name, version, attributes), log body (content and type), log attributes, and severity (number and text). The processor stores a 64-bit hash per unique combination.

`include_fields` / `exclude_fields` customize which parts of the record feed the hash. When `metadata_keys` is set, matching happens **within** each metadata bucket — two otherwise-identical logs arriving with different metadata values are never merged (see [advanced](advanced.md#multi-tenant-buckets-with-metadata_keys)).

## What gets emitted

At the end of each `interval`, one log per unique hash is emitted with:
- The original body, attributes, severity, resource.
- `log_count` (or the configured name) — duplicate count, int.
- `first_observed_timestamp` — RFC3339 string in the configured timezone.
- `last_observed_timestamp` — RFC3339 string in the configured timezone.
- `Timestamp` and `ObservedTimestamp` set to **emission time** (not original log time).

Counter resets after emission.
