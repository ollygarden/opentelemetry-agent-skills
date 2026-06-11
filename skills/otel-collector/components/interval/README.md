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

By default it aggregates cumulative monotonic sums/histograms plus gauges and summaries; delta and non-monotonic sums pass through unchanged. Gauges and summaries can be kept flowing as-is with `pass_through`.

## Main use-cases

Use it when:
- A receiver scrapes or emits cumulative metrics far more often than your backend needs (e.g., Prometheus receiver at 1s for a backend that ingests at 60s).
- You want to smooth bursty ingest into a fixed cadence downstream.
- Backend cost scales with point count and you have many slow-changing series.

Avoid it when:
- The pipeline carries mostly delta metrics — they bypass the processor entirely and you've added a stateful component for nothing.
- Gauge spikiness matters and you can't set `pass_through.gauge: true` for the relevant signals.
- The collector restarts often — buffered state is in-memory only and is lost on restart (see [State and restart behavior](quirks.md#state-and-restart-behavior)).

## Related components

- `batch` / exporter `sending_queue.batch` — batches by size/time, does not collapse per-series points. Complementary, not a replacement.
- `metricstransform`, `transform` — rewrite metrics; do not aggregate over time.
- `cumulativetodelta` — converts temporality. Often paired with `interval` so cumulative-source metrics reach a delta-only backend as one delta per interval.

## Details

- [Configuration](configuration.md) — config keys, defaults, per-metric-type behavior, and what "lossy" means.
- [Verification](verification.md) — telemetrygen recipe to confirm point volume drops.
- [Advanced use-cases](advanced.md) — keeping gauges spiky, multiple intervals per pipeline.
- [Known quirks](quirks.md) — state/restart behavior, troubleshooting, anti-patterns.
