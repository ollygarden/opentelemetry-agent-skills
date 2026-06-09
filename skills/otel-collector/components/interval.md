# `interval` processor

| | |
|-|-|
| Kind | processor |
| Type | `interval` |
| Signals | metrics |
| Stability | Alpha |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/intervalprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/intervalprocessor> |

## Description

Buffers cumulative metrics and emits the latest value once per interval. Reduces metric point volume from chatty sources while preserving the most recent reading per series.

### What it does to each metric type

| Metric type | Default behavior | Pass-through option |
|-------------|------------------|---------------------|
| Cumulative monotonic sum | Aggregated (only latest value per series emitted per interval) | — |
| Cumulative monotonic histogram | Aggregated | — |
| Cumulative monotonic exponential histogram | Aggregated | — |
| Gauge | Aggregated (lossy — see below) | `pass_through.gauge: true` |
| Summary | Aggregated (lossy — see below) | `pass_through.summary: true` |
| Delta (any) | Passed through unchanged, immediately | — (always pass-through) |
| Non-monotonic sum | Passed through unchanged, immediately | — (always pass-through) |

### What "lossy" means here

- Monotonic cumulative series: you lose **precision** (intermediate values), not totals — the final cumulative value still represents the full count.
- Gauges and summaries: actual **data loss**. A value that rose and fell back inside the interval is reduced to whatever the last reading happened to be. If the up-and-down matters (latency spikes, queue depth bursts), set `pass_through.gauge: true` / `pass_through.summary: true` to keep them flowing as-is.

## Main use-cases

Use it when:
- A receiver scrapes or emits cumulative metrics far more often than your backend needs (e.g., Prometheus receiver at 1s for a backend that ingests at 60s).
- You want to smooth bursty ingest into a fixed cadence downstream.
- Backend cost scales with point count and you have many slow-changing series.

Avoid it when:
- The pipeline carries mostly delta metrics — they bypass the processor entirely and you've added a stateful component for nothing.
- Gauge spikiness matters and you can't set `pass_through.gauge: true` for the relevant signals.
- The collector restarts often — buffered state is in-memory only and is lost on restart (see [State and restart behavior](#state-and-restart-behavior)).

## Typical config

```yaml
processors:
  interval:

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, interval]
      exporters: [otlphttp]
```

### Configuration reference

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `interval` | duration | `60s` | Emission cadence. After each emission, internal state is cleared. |
| `pass_through.gauge` | bool | `false` | If `true`, gauges are forwarded unchanged instead of buffered. |
| `pass_through.summary` | bool | `false` | If `true`, summaries are forwarded unchanged instead of buffered. |

Upstream documents no explicit validation rules. The Collector will reject an unparseable `interval` duration at startup.

> **Doc typo, upstream:** the README's configuration block shows `summary: <boo>l`. It's a typo for `<bool>` — the parameter is a plain boolean.

## Verification

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

Generate cumulative-sum metrics frequently (see the `otel-telemetrygen` skill):

```bash
telemetrygen metrics --otlp-insecure --otlp-endpoint localhost:4317 \
  --metric-type Sum --rate 5 --duration 25s
```

**What proves it worked:** instead of ~5 points/sec reaching `debug`, the exporter prints the metric roughly once per 10s interval (one latest value per series). Compare against the same run with the processor removed to see the volume drop.

## Advanced use-cases

### Keep gauges spiky, smooth everything else

```yaml
processors:
  interval:
    interval: 30s
    pass_through:
      gauge: true
      summary: true
```

Use this when your gauges represent things you can't afford to flatten — request latency, queue depth, current connections.

### Multiple intervals for different pipelines

```yaml
processors:
  interval/fast:
    interval: 15s
  interval/slow:
    interval: 5m

service:
  pipelines:
    metrics/realtime:
      receivers: [otlp]
      processors: [memory_limiter, interval/fast]
      exporters: [otlp/dashboard]
    metrics/longterm:
      receivers: [otlp]
      processors: [memory_limiter, interval/slow]
      exporters: [otlp/warehouse]
```

### Behavior example

Source metrics arriving into the processor:

| Time | Metric | Temporality | Attributes | Value |
|------|--------|-------------|------------|------:|
| 0 | `test_metric` | Cumulative | `labelA: foo` | 4.0 |
| 2 | `test_metric` | Cumulative | `labelA: bar` | 3.1 |
| 4 | `other_metric` | Delta | `fruitType: orange` | 77.4 |
| 6 | `test_metric` | Cumulative | `labelA: foo` | 8.2 |
| 8 | `test_metric` | Cumulative | `labelA: foo` | 12.8 |
| 10 | `test_metric` | Cumulative | `labelA: bar` | 6.4 |

`other_metric` (delta) is forwarded immediately. At the next interval boundary, only the latest value per series is forwarded:

| Time | Metric | Temporality | Attributes | Value |
|------|--------|-------------|------------|------:|
| 8 | `test_metric` | Cumulative | `labelA: foo` | 12.8 |
| 10 | `test_metric` | Cumulative | `labelA: bar` | 6.4 |

Then state is cleared.

## Known quirks

### State and restart behavior

- Internal state is keyed by metric name + attribute set and is held entirely in memory.
- After each emission the buffer is cleared. **If no new points arrive in the next interval, nothing is emitted for that series in that interval.**
- The state is not persisted — a collector restart drops every buffered point. For monotonic cumulative metrics this is usually fine (the next scrape carries the running total); for buffered gauges/summaries the most recent value is simply lost until the next sample.

### Troubleshooting

**Nothing comes out of the processor.**
- All upstream metrics are delta or non-monotonic sums — those would already pass through but produce no buffered emission. Confirm a cumulative monotonic metric reaches the processor.
- No new samples arrived in the most recent interval — state is cleared at emission, so a silent source means a silent output. Reduce the source's scrape gap or increase `interval`.

**Gauge spikes disappeared from dashboards.**
- Aggregation flattened them. Set `pass_through.gauge: true`.

**Memory growth.**
- Cardinality issue. The processor stores one entry per unique (metric, attribute set). Filter or drop high-cardinality attributes upstream (`transform` / `attributes` processor) before reaching `interval`.

**Output cadence is irregular after a restart.**
- Expected. The first interval after startup may emit less than usual because state was empty. Subsequent intervals settle once the upstream resends.

### Anti-patterns

**Stacking `interval` with delta-only metrics.**

```yaml
# BAD — adds a stateful component for no benefit; deltas pass through anyway
processors:
  interval:
service:
  pipelines:
    metrics:
      processors: [interval]
```

If the source emits only deltas, remove the processor.

**Using it as a backpressure / rate-limit substitute.**

`interval` reduces *point count* per series at emission boundaries; it does not throttle. Use `memory_limiter` (and the exporter's `sending_queue`) for backpressure.

**Aggregating spiky gauges silently.**

The default aggregates gauges and summaries. If you didn't explicitly think about it, flip `pass_through.gauge: true` until you have — losing a latency spike is harder to debug than ingesting one extra point per minute.

## Related components

- `batch` / exporter `sending_queue.batch` — batches by size/time, does not collapse per-series points. Complementary, not a replacement.
- `metricstransform`, `transform` — rewrite metrics; do not aggregate over time.
- `cumulativetodelta` — converts temporality. Often paired with `interval` so cumulative-source metrics reach a delta-only backend as one delta per interval.
