# `log_dedup`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

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
  --logs 50 --body "health check ok"
```

Don't add `--duration` here: telemetrygen ignores `--logs` when `--duration` is set, so the count would be whatever its default rate happens to emit instead of 50.

**What proves it worked:** the `debug` exporter prints **one** aggregated log per 5s interval carrying a `log_count` attribute of 50 (plus `first_observed_timestamp` / `last_observed_timestamp`), not 50 separate records.
