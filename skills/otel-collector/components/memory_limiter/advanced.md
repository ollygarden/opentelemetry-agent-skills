# `memory_limiter`: advanced use-cases

## Pair with `GOMEMLIMIT`

Set the `GOMEMLIMIT` environment variable to roughly **80% of the hard limit**. It tunes Go's garbage collector to target that heap level, so the runtime reclaims memory *before* the limiter has to start refusing — the two work together rather than the limiter being the only brake:

```yaml
# Container env
env:
  - name: GOMEMLIMIT
    value: "3200MiB"   # ~80% of limit_mib

processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 4000
    spike_limit_mib: 800
```

`GOMEMLIMIT` is a soft target for the GC pacer; `memory_limiter` is the hard backstop that refuses data. Use both. See <https://pkg.go.dev/runtime#hdr-Environment_Variables>.

## Percentage mode for Kubernetes

When the pod's memory limit varies across environments, use `limit_percentage` so the collector adapts without a config change. It reads the cgroup memory limit (v1/v2) or falls back to `/proc/meminfo`:

```yaml
# Pod spec: resources.limits.memory: 2Gi
processors:
  memory_limiter:
    check_interval: 1s
    limit_percentage: 75
    spike_limit_percentage: 15
# Hard limit = 1.5 GiB (75% of 2 GiB), soft limit = 1.2 GiB (60%)
```

`limit_mib` takes precedence if both are set. Percentage mode is most reliable on Linux with cgroups; on macOS/Windows it reads total *system* memory, which is rarely what you want (see [Known quirks](quirks.md)).

## Size `spike_limit_mib` for the traffic shape

`spike_limit_mib` must cover the largest heap increase that can occur in a single `check_interval`. The buffer between soft and hard limit is the recovery headroom:

| Traffic | Config |
|---------|--------|
| Steady | `check_interval: 1s`, `spike_limit_mib: 800` (20% of 4000) |
| Bursty (2× spikes) | `check_interval: 500ms`, `spike_limit_mib: 1200` (30%) — check more often *and* widen the buffer |
| Low-latency, predictable | `check_interval: 1s`, `spike_limit_mib: 400` (10%) — tight control |

If the collector still OOMs, the spike outran the buffer between two checks: lower `check_interval` and/or raise `spike_limit_mib`.

## Multi-pipeline: one config, one checker

A single `memory_limiter` definition referenced from every pipeline shares one background checker (the [singleton](quirks.md), keyed on identical config), so there is no redundant GC:

```yaml
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 4000
    spike_limit_mib: 800

service:
  pipelines:
    traces:  { processors: [memory_limiter, batch] }
    metrics: { processors: [memory_limiter, batch] }
    logs:    { processors: [memory_limiter, batch] }
```

## Tuning the GC intervals

Four keys shape forced GC so it does not pin the CPU when memory hovers near the limit. The `min_gc_interval_*` keys are the **floor** between forced GCs; the `max_gc_interval_*` keys **cap the exponential backoff** the processor applies when a GC reclaims `< 5%` of heap (memory pinned by live references — see [Known quirks](quirks.md)). Keep the soft-limited interval `>=` the hard-limited one on both pairs (the hard limit is more urgent, so it should GC at least as eagerly):

```yaml
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 4000
    min_gc_interval_when_soft_limited: 10s   # default — opportunistic floor
    min_gc_interval_when_hard_limited: 0s    # default — GC every over-hard-limit check
    max_gc_interval_when_soft_limited: 30s   # default — backoff cap
    max_gc_interval_when_hard_limited: 30s   # default — backoff cap
```

Reach for these only if debug logs show `Forcing a GC` firing so often that CPU suffers (raise the `min`), or the backoff messages (`Forced GC did not reclaim enough memory…`) show GC still burning CPU during a downstream outage (lower the `max`, or set it to `0` only if you want to keep GC pinned at the `min`). The defaults are right for most deployments.
