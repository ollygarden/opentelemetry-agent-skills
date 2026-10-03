# `interval`: known quirks

## State and restart behavior

- Internal state is keyed by metric name + attribute set and is held entirely in memory.
- After each emission the buffer is cleared. **If no new points arrive in the next interval, nothing is emitted for that series in that interval.**
- The state is not persisted — a collector restart drops every buffered point. For monotonic cumulative metrics this is usually fine (the next scrape carries the running total); for buffered gauges/summaries the most recent value is simply lost until the next sample.

## Troubleshooting

**Nothing comes out of the processor.**
- All upstream metrics are non-monotonic sums — those pass through but produce no buffered emission. Confirm a monotonic sum, histogram, gauge, or summary reaches the processor.
- No new samples arrived in the most recent interval — state is cleared at emission, so a silent source means a silent output. Reduce the source's scrape gap or increase `interval`.

**Delta metrics show no volume drop.**
- Before v0.162.0, delta metrics passed through unchanged. Upgrade the collector.

**Gauge spikes disappeared from dashboards.**
- Aggregation flattened them. Set `pass_through.gauge: true`.

**Memory growth.**
- Cardinality issue. The processor stores one entry per unique (metric, attribute set). Filter or drop high-cardinality attributes upstream (`transform` / `attributes` processor) before reaching `interval`.

**Output cadence is irregular after a restart.**
- Expected. The first interval after startup may emit less than usual because state was empty. Subsequent intervals settle once the upstream resends.

## Anti-patterns

**Stacking `interval` with pass-through-only metrics.**

```yaml
# BAD when the source emits only non-monotonic sums — they pass through anyway
processors:
  interval:
service:
  pipelines:
    metrics:
      processors: [interval]
```

If nothing the source emits is aggregated, remove the processor.

**Using it as a backpressure / rate-limit substitute.**

`interval` reduces *point count* per series at emission boundaries; it does not throttle. Use `memory_limiter` (and the exporter's `sending_queue`) for backpressure.

**Aggregating spiky gauges silently.**

The default aggregates gauges and summaries. If you didn't explicitly think about it, flip `pass_through.gauge: true` until you have — losing a latency spike is harder to debug than ingesting one extra point per minute.
