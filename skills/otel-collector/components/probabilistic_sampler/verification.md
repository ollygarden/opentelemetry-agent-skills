# `probabilistic_sampler`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`probabilistic_sampler` ships in the `core`, `contrib`, and `k8s` distributions, so a stock contrib collector can run this.

Config (`probabilistic-verify.yaml`) — sample 50% of traces:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  probabilistic_sampler:
    sampling_percentage: 50
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [probabilistic_sampler]
      exporters: [debug]
```

Send a good number of traces — each gets a fresh random TraceID, so roughly half clear the 50% threshold (see the `otel-telemetrygen` skill):

```bash
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 500
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-insecure`, `--otlp-endpoint`, and `--traces` (int, traces per worker; ignored if `--duration` is set, so no `--duration` here or the count would not be 500). telemetrygen assigns a random TraceID per trace, which is what makes the hash spread evenly.

Use a large `N` (≥ 500) so the ratio is clearly readable — with only a handful of traces the kept fraction is noisy.

**What proves it worked:** noticeably fewer traces reach the `debug` exporter than were sent — close to half. Because the decision is statistical, the exact fraction varies run to run, but at `N = 500` it lands near 50% (a verified run kept 257/500 ≈ 51%). The signal is "clearly fewer than sent, in the neighborhood of 50%", not an exact number.
