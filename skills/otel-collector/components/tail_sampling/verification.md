# `tail_sampling`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`tail_sampling` ships in the `contrib` and `k8s` distributions.

This recipe targets `otel/opentelemetry-collector-contrib:0.159.0` with
`ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:v0.159.0`.
The same status-code-policy config works on v0.160.0; it does not exercise tracestate-aware limiting.

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
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 \
  --traces 20 --status-code Ok
# Error traces — expect these to survive
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 20 --status-code Error
```

The `--status-code` flag is documented in the `otel-telemetrygen` skill (accepted values: `Unset`/`0`, `Error`/`1`, `Ok`/`2`). It sets the span status that the `status_code` policy evaluates. telemetrygen's integer mapping (`1`=Error, `2`=Ok) intentionally differs from the OpenTelemetry status-code enum (`1`=Ok, `2`=Error), so pass the strings shown above.

**What proves it worked:** after `decision_wait`, the `debug` exporter shows the 20 error traces and none of the 20 OK traces. telemetrygen emits two spans per generated trace, so the debug output contains 40 Error spans and no Ok spans.
