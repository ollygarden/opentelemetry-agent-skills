# `cardinality_guardian`: advanced use-cases

## Tag-and-route to cheap storage (production-safe)

Keep the offending series but split it off to cheap storage with the `routingconnector`, avoiding the single-writer problem:

```yaml
processors:
  cardinality_guardian:
    tag_only: true
connectors:
  routing:
    default_pipelines: [metrics/clean]
    table:
      - condition: 'attributes["otel.metric.overflow"] == true'
        pipelines: [metrics/overflow]
service:
  pipelines:
    metrics/in:
      receivers: [otlp]
      processors: [cardinality_guardian]
      exporters: [routing]
    metrics/clean:
      receivers: [routing]
      exporters: [prometheusremotewrite]
    metrics/overflow:
      receivers: [routing]
      exporters: [file]   # cheap cold storage
```

## Per-metric overrides

A legitimately wide metric (high but expected cardinality) can be given a looser threshold so it is not enforced at the global level:

```yaml
processors:
  cardinality_guardian:
    max_cardinality_delta_per_epoch: 100
    metric_overrides:
      http.server.request.duration: 5000
```

## Bounding memory with `max_tracker_count`

Each `(metric, label)` pair holds a HyperLogLog sketch. On a pipeline with a huge metric/label fan-out, set `max_tracker_count` to cap memory. Once the cap is hit, new pairs pass through untracked and `otelcol_processor_cardinality_trackers_rejected` increments — watch that counter to know whether the cap is too low.

## Interpreting the internal telemetry

| Metric | Type | Use |
|--------|------|-----|
| `otelcol_processor_cardinality_trackers_active` | Gauge | Active `(metric, label)` trackers across all shards. |
| `otelcol_processor_cardinality_labels_stripped` | Counter | Labels stripped or tagged. Alert on `rate(...[5m])` for spike detection. |
| `otelcol_processor_cardinality_top_offenders` | Gauge | Top-N highest-delta pairs (`metric_name`, `label_key` attributes) — find the exact pair exploding. |
| `otelcol_processor_cardinality_trackers_rejected` | Counter | New pairs ignored after hitting `max_tracker_count`. |
| `otelcol_processor_cardinality_savings_estimated` | Counter | Estimated dollar value of series prevented from reaching the TSDB. |
