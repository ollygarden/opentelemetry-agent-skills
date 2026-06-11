# `cardinality_guardian`: configuration

## Typical config

```yaml
processors:
  cardinality_guardian:

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, cardinality_guardian]
      exporters: [prometheusremotewrite]
```

## Configuration reference

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `max_cardinality_delta_per_epoch` | int | `100` | Max new unique values per `(metric, label key)` per epoch. Must be `> 0`. Global threshold; overridable per metric. |
| `epoch_duration_seconds` | int | `300` | Epoch (sliding window) rotation interval. Must be `>= 10`. Shorter = more sensitive but noisier; longer = smoother but slower to react. |
| `tag_only` | bool | `false` | `false` strips the offending label; `true` keeps it and adds `otel.metric.overflow: true`. `true` is the production-safe mode (see Known quirks). |
| `never_drop_labels` | []string | `[http.status_code, region]` | Label keys never stripped or tagged regardless of growth. O(1) exempt-list, read only at startup. |
| `metric_overrides` | map[string]int | `nil` | Per-metric delta overrides. Each value must be `> 0`; empty metric names are rejected at validation. Unspecified metrics use the global default. |
| `top_offenders_count` | int | `10` | Number of highest-delta `(metric, label)` pairs reported via gauge. `0` disables. Range `0`–`500`. Snapshot computed once per epoch (no hot-path cost). |
| `max_tracker_count` | int | `0` | Max concurrent trackers. `0` = unlimited. Range `0`–`10,000,000`. Once reached, new pairs pass through untracked and increment `..._trackers_rejected`. |
| `estimated_cost_per_metric_month` | float64 | `0.05` | Dollar value per series prevented, for the savings counter. `0` disables. Must be `>= 0`. Does not affect enforcement. |
| `drop_log_max_per_epoch` | int | `10` | Cap on per-epoch "dropping attribute" warning logs. `0` = no cap (logs every drop, not recommended at scale). Must be `>= 0`. |

Validation runs at pipeline construction; any out-of-range field prevents startup.

## How detection works

The processor measures cardinality **growth**, not absolute cardinality. For each `(metric, label key)` it keeps a HyperLogLog sketch and, at the end of every epoch, promotes the current sketch to "previous" and starts fresh. The enforcement check compares the current epoch's unique-value estimate against the previous epoch's — only the **delta** is tested against the threshold. A metric that has already reached a large but stable label space is left alone; only metrics whose label space is actively *exploding* trigger enforcement.

## Internal telemetry

| Metric | Type | Use |
|--------|------|-----|
| `otelcol_processor_cardinality_trackers_active` | Gauge | Active `(metric, label)` trackers across all shards. |
| `otelcol_processor_cardinality_labels_stripped` | Counter | Labels stripped or tagged. Alert on `rate(...[5m])` for spike detection. |
| `otelcol_processor_cardinality_top_offenders` | Gauge | Top-N highest-delta pairs (`metric_name`, `label_key` attributes) — find the exact pair exploding. |
| `otelcol_processor_cardinality_trackers_rejected` | Counter | New pairs ignored after hitting `max_tracker_count`. |
| `otelcol_processor_cardinality_savings_estimated` | Counter | Estimated dollar value of series prevented from reaching the TSDB. |
