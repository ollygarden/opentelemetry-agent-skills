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

See the [Internal telemetry](configuration.md#internal-telemetry) table in `configuration.md` for the full metric list. When tuning, the most useful signals are `otelcol_processor_cardinality_top_offenders` (find the exact `(metric, label)` pair exploding), `otelcol_processor_cardinality_labels_stripped` (alert on `rate(...[5m])` for spikes), and `otelcol_processor_cardinality_trackers_rejected` (whether `max_tracker_count` is too low).
