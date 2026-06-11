# `tail_sampling`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`tail_sampling` ships in the `contrib` and `k8s` distributions.

Config (`tailsampling-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  tail_sampling:
    decision_wait: 5s
    num_traces: 1000
    policies:
      - name: errors-only
        type: status_code
        status_code:
          status_codes: [ERROR]
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [tail_sampling]
      exporters: [debug]
```

Generate traces, some with error status (see the `otel-telemetrygen` skill):

```bash
# OK traces — expect these to be dropped by the errors-only policy
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 20
# Error traces — expect these to survive
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 20 --status-code Error
```

The `--status-code Error` flag is confirmed in the `otel-telemetrygen` skill (accepted values: `Unset`/`0`, `Error`/`1`, `Ok`/`2`). It sets the span status to ERROR, which the `status_code` policy matches. Note telemetrygen's own integer mapping (`1`=Error, `2`=Ok) intentionally differs from the OpenTelemetry status-code enum (`1`=Ok, `2`=Error) — pass the string `Error` to avoid ambiguity.

**What proves it worked:** after `decision_wait`, the `debug` exporter shows the error traces and not the OK traces.
