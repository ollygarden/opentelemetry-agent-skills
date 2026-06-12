# `memory_limiter`: verification

See [Verification harness](../../SKILL.md#verification-harness) for how to run this end-to-end.

`memory_limiter` ships in the `core`, `contrib`, and `k8s` distributions, so a stock contrib collector can run this.

Because the processor reacts to **real heap pressure**, the trick to verifying it deterministically is to set the limit *below the collector's own idle heap* so it refuses from the first check — no need to actually exhaust memory.

Config (`memlimit-verify.yaml`) — limit far below the idle heap so the soft limit is breached immediately:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 20
    spike_limit_mib: 5
exporters:
  debug:
    verbosity: detailed
service:
  telemetry:
    logs:
      level: debug
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter]
      exporters: [debug]
```

Wait ~5s for the first memory check, then send a few traces (see the `otel-telemetrygen` skill):

```bash
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 10 --workers 1
```

Flags confirmed against the `otel-telemetrygen` skill (`references/flags.md`): `--otlp-insecure`, `--otlp-endpoint`, `--traces` (int, traces **per worker**; ignored if `--duration` is set, so no `--duration` here), and `--workers` (default 1). The exact count does not matter here — every export is refused regardless.

**What proves it worked** (verified run, contrib 0.152.0, idle heap ~17–20 MiB > the 15 MiB soft limit):

1. The collector logs the limit being breached:
   ```
   warn  Memory usage is above hard limit. Forcing a GC.   {"cur_mem_mib": 20}
   info  Memory usage after GC.                            {"cur_mem_mib": 17}
   warn  Memory usage is above soft limit. Refusing data.  {"cur_mem_mib": 17}
   ```
2. telemetrygen's export fails with the non-permanent gRPC error (it retries with backoff, which is exactly the intended backpressure):
   ```
   traces export: ... rpc error: code = Unavailable desc = data refused due to high memory usage
   ```
3. **Zero** spans reach the `debug` exporter (`grep -c '^Span #'` in the collector logs returns `0`) — the data was refused before the pipeline, not dropped silently downstream.

To watch normal operation instead, raise `limit_mib` well above the idle heap (e.g. `limit_mib: 4000`) and the same send passes through with spans printed by `debug`.
