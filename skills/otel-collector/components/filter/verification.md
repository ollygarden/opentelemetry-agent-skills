# `filter`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`filter` ships in the `core`, `contrib`, and `k8s` distributions, so a stock contrib collector can run this.

Config (`filter-verify.yaml`) — drop `ERROR`-and-above logs, let everything below pass:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  filter:
    error_mode: ignore
    log_conditions:
      - 'log.severity_number >= SEVERITY_NUMBER_ERROR'
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [filter]
      exporters: [debug]
```

Send two batches — one that the condition matches (dropped) and one that it doesn't (kept) — see the `otel-telemetrygen` skill:

```bash
# Info logs (severity 9) — below ERROR, expect these to survive
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 10 --severity-number 9 --body "info log keep me"
# Error logs (severity 17) — at/above ERROR, expect these to be dropped
telemetrygen logs --otlp-insecure --otlp-endpoint localhost:4317 \
  --logs 10 --severity-number 17 --body "error log drop me"
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-insecure`, `--otlp-endpoint`, `--logs`, `--severity-number` (int32, 1–24; 9=Info, 17=Error per the severity-number reference), and `--body`. No `--duration` here — telemetrygen ignores `--logs` when `--duration` is set, so the count would not be 10.

**What proves it worked:** the `debug` exporter prints the 10 Info records ("info log keep me") and **none** of the 10 Error records ("error log drop me"). Fewer records reach `debug` than were sent, and only the non-matching severity survives.
