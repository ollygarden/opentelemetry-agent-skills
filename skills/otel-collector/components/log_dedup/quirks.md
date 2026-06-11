# `log_dedup`: known quirks

## Memory model

- One entry per unique hash for the duration of the interval.
- High-cardinality fields (`request_id`, `trace_id`, timestamps) blow up memory — exclude them or use `conditions` to scope what's deduplicated.
- Long intervals accumulate more unique entries; short intervals emit faster with smaller buffers.

## Telemetry

The processor emits a histogram `otelcol_dedup_processor_aggregated_logs` with the count of logs aggregated per emission. Useful for tuning. The metric is currently **Development** stability — the name or shape can change between releases.

## Troubleshooting

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

## Validation errors

| Error | Fix |
|-------|-----|
| `cannot define both exclude_fields and include_fields` | Choose one. |
| `an excludefield must start with body or attributes` | Prefix every path with `body.` or `attributes.`. |
| `cannot exclude the entire body` | Only nested fields are exclud-able (e.g., `body.timestamp`). |

## Anti-patterns

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
