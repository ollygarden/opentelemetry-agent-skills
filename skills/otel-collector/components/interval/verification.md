# `interval`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`interval` ships in the `contrib` and `k8s` distributions.

Config (`interval-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  interval:
    interval: 10s
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [interval]
      exporters: [debug]
```

Generate cumulative-sum metrics frequently (see the `otel-telemetrygen` skill). The explicit `--aggregation-temporality cumulative` is load-bearing — `interval` only aggregates cumulative series; delta sums pass through unchanged and would show no volume drop:

```bash
telemetrygen metrics --otlp-insecure --otlp-endpoint localhost:4317 \
  --metric-type Sum --aggregation-temporality cumulative --rate 5 --duration 25s
```

**What proves it worked:** instead of ~5 points/sec reaching `debug`, the exporter prints the metric roughly once per 10s interval (one latest value per series). Compare against the same run with the processor removed to see the volume drop.
