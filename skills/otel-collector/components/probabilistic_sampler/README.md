# `probabilistic_sampler` processor

| | |
|-|-|
| Kind | processor |
| Type | `probabilistic_sampler` |
| Signals | traces (Beta), logs (Alpha) |
| Distributions | core, contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/probabilisticsamplerprocessor> |

## Description

Head sampling: makes a deterministic keep/drop decision **per item** by hashing the TraceID (or, for logs, a configurable attribute) and comparing it against a threshold derived from `sampling_percentage`. The decision is stateless and content-blind — no buffering, no waiting — so the same TraceID is always kept or always dropped, and all collectors that share the same `hash_seed` agree on the outcome. Because the hash is uniform over trace IDs, sampling roughly `sampling_percentage`% of items.

This is the opposite of **tail sampling** (`tail_sampling`), which buffers the **whole trace** and makes one decision after a wait window using trace-wide signals (errors, latency, span count). Head sampling is cheap and immediate but cannot react to span content; tail sampling is content-aware but stateful, requires buffering, adds latency, and needs all spans of a trace to land on the same instance. In the W3C-randomness modes (`proportional` / `equalizing`), the threshold is propagated downstream so a later sampler can stay consistent — in the trace's `tracestate` (`ot=th:...`) for traces, and in log attributes (`sampling.threshold` / `sampling.randomness`) for logs. The default `hash_seed` mode is self-contained: it decides locally and does not propagate a threshold.

## Main use-cases

Use it when:
- You want cheap, stateless volume reduction at the edge with no buffering or co-location requirement.
- You want a predictable, reproducible keep/drop decision per TraceID across a fleet of collectors (consistent `hash_seed`).
- You want to sample logs by TraceID or by a record attribute.
- You want to combine head sampling at the edge with `tail_sampling` at a gateway (see [Advanced use-cases](advanced.md)).

Avoid it when:
- You need a content-aware decision over a **whole trace** — keep all errors, keep slow traces — use `tail_sampling`. See its README for the head-vs-tail trade-off.
- You need exact counts; sampling is statistical, so the kept fraction only approaches `sampling_percentage`% over many items.
- Dropping would orphan child spans or break correlation — note the child-≥-parent completeness rule in [Known quirks](quirks.md) mitigates but does not eliminate this.

## Related components

- `tail_sampling` — head's counterpart: trace-wide, content-aware keep/drop after buffering. Stateful and latency-adding, but can act on errors/latency.
- `filter` — drops telemetry by OTTL condition rather than by a probability; use it for deterministic noise removal, not statistical sampling.
- `transform` — mutates telemetry via OTTL instead of dropping it; run it before/after to set the attributes a record-based sampler reads.

## Details

- [Configuration](configuration.md) — full config table, the three sampling modes explained, and the traces-vs-logs differences.
- [Verification](verification.md) — telemetrygen recipe that sends many traces and shows roughly half reach `debug`.
- [Advanced use-cases](advanced.md) — mode selection, consistent `hash_seed` across a fleet, logs sampling by attribute, `sampling_priority` overrides, head+tail combinations, the `fail_closed` trade-off.
- [Known quirks](quirks.md) — `fail_closed` dropping zero/invalid randomness, hash-seed agreement, precision caps, tracestate persistence, child-≥-parent completeness, statistical counts, stability caveats.
