# `redaction`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`redaction` ships in the `contrib` and `k8s` distributions.

Config (`redaction-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  redaction:
    allow_all_keys: true
    blocked_values:
      - '4[0-9]{12}(?:[0-9]{3})?'   # Visa-like card numbers
    summary: info
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [redaction]
      exporters: [debug]
```

Generate spans carrying an attribute value that matches the blocked pattern. The span-level attribute flag is `--telemetry-attributes` (resource-level `--otlp-attributes` would attach to the Resource, not the span); the value must be quoted (`key="value"`):

```bash
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 \
  --traces 5 --telemetry-attributes 'cc_number="4111111111111111"'
```

**What proves it worked:** the `debug` exporter shows the matching value masked (replaced with asterisks, or hashed if `hash_function` is set) while non-matching attributes pass through; with `summary: info` the span carries a `redaction.masked.count` audit attribute (and, because nothing is removed under `allow_all_keys: true`, an `redaction.allowed.count`).
