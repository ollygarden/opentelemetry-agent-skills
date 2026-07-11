# `memory_limiter` processor

| | |
|-|-|
| Kind | processor |
| Type | `memory_limiter` |
| Signals | traces (Beta), metrics (Beta), logs (Beta), profiles (Alpha) |
| Distributions | core, contrib, k8s |
| Go module | `go.opentelemetry.io/collector/processor/memorylimiterprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector/tree/main/processor/memorylimiterprocessor> |

## Description

A safety valve that keeps the Collector from being OOM-killed. A background goroutine reads Go heap allocation (`runtime.MemStats.Alloc`) every `check_interval` and compares it against two thresholds derived from the configured limit: a **soft limit** (`limit_mib - spike_limit_mib`) and a **hard limit** (`limit_mib`). Above the soft limit the processor sets a `mustRefuse` flag and every data path returns a **non-permanent** `ErrDataRefused` to the preceding component — backpressure that well-behaved receivers retry with backoff. Above the hard limit it additionally forces a `runtime.GC()` (with exponential backoff when a GC proves ineffective, capped by the `max_gc_interval_*` keys). It also reports its health via `componentstatus` — recoverable-error while refusing, OK once recovered. When heap drops back below the soft limit it resumes normally; no manual intervention.

It measures **heap**, not process RSS — total process memory runs 50–100 MiB higher, so container limits should sit above `limit_mib`. The processor does not mutate data (`MutatesData: false`); it only accepts or refuses. It is a safety net, not a sizing or performance tool — a chronically undersized collector will simply refuse data constantly. Use it together with the `GOMEMLIMIT` environment variable so Go's GC paces toward the same target.

## Main use-cases

Use it when:
- **Always** — on every production Collector, as the **first** processor in every pipeline.
- You run in a memory-capped environment (Kubernetes pod, Docker container) and must avoid OOM kills under traffic spikes.
- You want early backpressure to receivers so clients rate-limit at the source before buffers fill downstream.

Avoid it when:
- You are reaching for it to fix throughput on an undersized collector — size the collector first, then add this as a safety net (see [Known quirks](quirks.md)).
- A minimal verification/repro pipeline where you are isolating another component on purpose (the skill's verification configs omit it deliberately).

## Related components

- `batch` / exporter `sending_queue.batch` — batching allocates buffers; `memory_limiter` must sit **before** it so refusal happens before data is buffered.
- `tail_sampling`, `transform`, `log_dedup` — stateful/allocating processors; place them after `memory_limiter` so memory pressure refuses data before it reaches them.
- The **`memory_limiter` extension** — same checker exposed as gRPC/HTTP middleware (experimental); see [Known quirks](quirks.md).

## Details

- [Configuration](configuration.md) — full config table (`check_interval`, `limit_mib` vs `limit_percentage`, spike limits, min/max GC intervals), validation rules, and the soft/hard-limit mechanism with GC backoff.
- [Verification](verification.md) — telemetrygen recipe that forces refusal with a deliberately tiny limit and shows `ErrDataRefused` in the logs.
- [Advanced use-cases](advanced.md) — `GOMEMLIMIT` pairing, percentage mode for Kubernetes, spike sizing for bursty traffic, multi-pipeline singleton behavior.
- [Known quirks](quirks.md) — heap-vs-RSS, required fields, `limit_percentage` cgroup detection, GC-interval ordering, the singleton checker, the experimental extension variant, stability caveats.
