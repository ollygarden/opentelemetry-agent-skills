# `cardinality_guardian`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`cardinality_guardian` is not bundled in any distribution — build a custom collector that includes `processor/cardinalityguardianprocessor` via the OpenTelemetry Collector Builder (OCB) first.

Config (`cardguard-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  cardinality_guardian:
    max_cardinality_delta_per_epoch: 5
    epoch_duration_seconds: 10
    tag_only: true
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [cardinality_guardian]
      exporters: [debug]
```

Emit metrics whose attribute set explodes — many unique values for one label key within an epoch. **`telemetrygen` cannot do this directly:** its `--telemetry-attributes key="value"` flag sets one fixed value, so every data point carries the *same* label value and no per-epoch growth is ever observed. The experimental `--unique-timeseries` flag varies the timeseries but does not let you target a specific named label key with a growing value space. To exercise the processor you need varying values on one label key, which means either:

- Prepend a `transform`/OTTL `metrics` statement upstream in the same pipeline that rewrites a label to an exploding value (e.g. derive it from a high-cardinality source), so the input shape grows over the epoch; see the `otel-ottl` skill for `set(...)` on metric attributes, or
- Use a small custom emitter (a few lines with the metrics SDK) that increments a counter while setting `runaway="value-N"` with `N` incrementing per export.

A constant-attribute run is still useful as a baseline — it should produce **no** overflow tagging, confirming stable label spaces are left alone:

```bash
telemetrygen metrics --otlp-insecure --otlp-endpoint localhost:4317 \
  --metric-type Sum --rate 50 --duration 30s --telemetry-attributes 'runaway="value-N"'
```

**What proves it worked (`tag_only: true`):** once a label crosses the per-epoch delta, the `debug` exporter shows the offending data points carrying `otel.metric.overflow: true` while other labels remain intact; the internal metric `otelcol_processor_cardinality_labels_stripped` increments. With a constant label value (the `telemetrygen` baseline above) nothing is tagged — that confirms growth, not absolute cardinality, is what triggers enforcement.
