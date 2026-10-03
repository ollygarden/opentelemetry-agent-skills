# Ruby performance and lifecycle

## Start with the measured bottleneck

Separate instrumentation overhead from exporter/network delay, application allocation, and backend
ingestion. Benchmark the real framework or job path with representative concurrency and payloads.
Record throughput, latency, allocations, queue drops, and exported telemetry before changing knobs.

## Sampling and batching

Use a parent-based sampler when distributed parent decisions must be preserved. Configure standard
sampler variables before `OpenTelemetry::SDK.configure`:

```sh
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
```

The default OTLP trace path uses a batch span processor. Its queue, delay, batch, timeout, and
thread-start behavior are configurable in the locked SDK release. Inspect the constructor and
standard `OTEL_BSP_*` variables before tuning; `OTEL_RUBY_BSP_START_THREAD_ON_BOOT` is Ruby-specific.
Larger queues can reduce transient drops but retain more objects. A simple processor exports on the
request path and is unsuitable for normal production traffic.

## Forks, threads, and fibers

Pre-fork servers copy provider and processor state. Verify the exact SDK/server combination rather
than assuming a background export thread survives. The released batch span processor and metrics
SDK include fork handling, but startup timing, inherited buffers, logs processing, and graceful
shutdown still matter. Exercise a request in a worker and prove each enabled signal exports after
the fork.

Context is fiber-local. Test parentage across any custom thread pools, fibers, async schedulers,
jobs, and callbacks that framework instrumentation does not already cover.

## Metrics cardinality and cost

The experimental metrics SDK applies cardinality limits (default 2000 per stream in `0.19.0`) and
supports views. Prefer bounded
dimensions; use a view to retain only useful attribute keys or drop a noisy instrument. Confirm
the locked release's limit and overflow behavior before relying on it. Check delta/cumulative
temporality against the receiving backend when changing the OTLP metrics preference.

## Attribute and payload bounds

Use standard `OTEL_ATTRIBUTE_*`, `OTEL_SPAN_*`, event, and link limits supported by the locked SDK.
Ruby also accepts `OTEL_RUBY_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT` for compatibility, but prefer the
standard variable for new configuration. Limits protect process memory and payload size; they do
not make sensitive values safe to collect.

## Export and shutdown

- Prefer OTLP through a nearby Collector over synchronous export from request handling.
- Use compression only after measuring CPU and network impact.
- Keep exporter timeouts bounded below the application's shutdown budget.
- Call provider `shutdown` from the server/job lifecycle so queued telemetry is flushed.
- Use `force_flush(timeout: ...)` for bounded checkpoints, not on every request.

An empty backend query is not proof of zero telemetry. Enable `OTEL_LOG_LEVEL=debug`, inspect SDK
warnings and processor drop signals, and compare with a disposable local receiver.

Sources: [trace SDK 1.13.1](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-sdk/v1.13.1/sdk),
[metrics SDK 0.19.0](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-metrics-sdk/v0.19.0/metrics_sdk), and
[Ruby API 1.11.1 benchmarks](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-api/v1.11.1/api/benchmarks).
