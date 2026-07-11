# `memory_limiter`: known quirks

## It measures heap, not RSS

The limit is compared against Go **heap allocation** (`runtime.MemStats.Alloc`), not process RSS. Total process memory typically runs **50–100 MiB higher** (Go runtime, stacks, CGO, OS page caches, mmap'd files). So `limit_mib: 4000` means a process that sits around ~4050 MiB. Set the container memory limit **10–15% above** `limit_mib`, or the kernel OOM-kills the process before the limiter ever refuses. Seeing RSS exceed `limit_mib` is expected, not a bug.

## `check_interval` and one of the limits are required

There are no usable defaults for the two load-bearing keys. `check_interval` defaults to `0s`, which is a validation error, and you must set `limit_mib` **or** `limit_percentage` (`> 0`). A config with only one of these fails at startup:

```
'check_interval' must be greater than zero
'limit_mib' or 'limit_percentage' must be greater than zero
```

## `limit_mib` wins over `limit_percentage`

If both are set, `limit_mib` is used and `limit_percentage` is ignored — silently. Set only the one you mean.

## `spike_limit` must be smaller than the limit

`spike_limit_mib < limit_mib` (and `spike_limit_percentage < limit_percentage`) is validated. If the spike limit equals or exceeds the limit, the soft limit would be `<= 0` and startup fails with `'spike_limit_mib' must be smaller than 'limit_mib'`.

## GC-interval ordering is enforced

`min_gc_interval_when_soft_limited` must be `>=` `min_gc_interval_when_hard_limited` (the hard limit is more urgent, so it cannot GC *less* often). Inverting them fails startup with `'min_gc_interval_when_soft_limited' should be larger than 'min_gc_interval_when_hard_limited'`. The same ordering holds for the `max_gc_interval_*` caps when both are set, and each `max` (unless `0`) must be `>=` its own `min` — see the [validation rules](configuration.md#validation-rules).

## GC backs off when it stops helping

Since v0.156.0, forced GC no longer fires unconditionally on every over-limit check. When a GC reclaims `< 5%` of heap — typically because the memory is pinned by live references such as exporter sending queues during a downstream outage — the processor treats it as ineffective and **doubles** the interval before the next forced GC, up to `max_gc_interval_when_soft_limited` / `max_gc_interval_when_hard_limited` (both default `30s`). This prevents a CPU-burning GC loop that would starve recovery. The interval resets the moment a GC frees memory or heap drops. Set the relevant `max_gc_interval_*` to `0` to opt out and keep GC pinned at the `min` interval.

## It reports component health

Since v0.156.0 the processor publishes health events via `componentstatus.ReportStatus`: a **recoverable-error** event while it is refusing data and `StatusOK` once heap is back within limits. Status-aware extensions (e.g. `healthcheckv2`) surface this, so a collector under memory pressure can report unhealthy without you scraping logs or refusal metrics.

## `limit_percentage` needs cgroups to be useful

Percentage mode reads the container's memory limit from cgroups v2 (`/sys/fs/cgroup/memory.max`) or v1 (`/sys/fs/cgroup/memory/memory.limit_in_bytes`), and treats an "unlimited" cgroup value (`9223372036854771712`) as absent, falling back to `/proc/meminfo`. On macOS/Windows it reads **total system memory**, not a container limit, so the percentage is rarely meaningful there. Prefer `limit_mib` on non-Linux hosts. If total-memory detection fails entirely the collector refuses to start — switch to `limit_mib`.

## It is a safety net, not a sizing tool

The limiter does not make a small collector handle more load — below capacity it simply **refuses data constantly** (refusal metrics, persistent *"Refusing data"* logs, receivers seeing backpressure). Memory-limiter metric names have changed across Collector releases — v0.156.0 renamed them to a component-specific prefix (`otelcol_processor_memory_limiter_accepted_*` / `otelcol_processor_memory_limiter_refused_*`) — so check the exact Collector version before writing alerts. The fix is to size the collector for the workload or scale out, then keep `memory_limiter` as the backstop. Treat sustained refusal as an under-provisioning signal, not a tuning problem.

## Refusal is non-permanent — receivers must retry

When refusing, the processor returns a **non-permanent** error (gRPC `Unavailable`, *"data refused due to high memory usage"*). Well-behaved receivers (otlp, prometheus, …) retry with backoff and clients rate-limit. A receiver or client that does **not** retry will lose that data. This is by design: the limiter applies backpressure rather than buffering.

## Singleton checker across identical configs

Pipelines referencing the same `memory_limiter` config share one background goroutine and timer (keyed on the config), so GC isn't triggered redundantly per pipeline. Two *differently named* instances with different limits run independent checkers — intentional, but watch that their limits don't sum past the container budget.

## The experimental extension variant

Since v0.142.0 the memory limiter is also available as an **extension** that acts as gRPC/HTTP middleware on a receiver (v0.147.0 added streaming support and fixed a multi-interceptor startup panic):

```yaml
extensions:
  memory_limiter:
    check_interval: 1s
    limit_mib: 4000
receivers:
  otlp:
    protocols:
      grpc:
        middlewares: [{id: memory_limiter}]
service:
  extensions: [memory_limiter]
```

It is **experimental** — use the processor form for production. Note: the extension is **not compiled into the stock `otelcol-contrib` distribution** (verified on v0.156.0, which rejects it with `'extensions' unknown type: "memory_limiter"`); it requires a custom build.

## Stability caveats

Traces/metrics/logs are Beta; profiles are Alpha. The profiles path can still change between releases — confirm against the upstream README for the exact collector version before relying on it.
