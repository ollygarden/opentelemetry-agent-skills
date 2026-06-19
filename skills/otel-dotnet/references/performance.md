# OpenTelemetry .NET Performance Tuning

Performance tuning reference for OpenTelemetry .NET SDK. Covers sampling, batch processing, metric readers, Views for cardinality control, exporter configuration, async context, and graceful shutdown.

---

## Performance Impact by Signal

| Signal | Unsampled Overhead | Sampled Overhead | Primary Cost |
|--------|--------------------|------------------|--------------|
| Traces | Near-zero (noop Activity) | Moderate | Allocations, export I/O |
| Metrics | N/A (always collected) | N/A | Aggregation, cardinality |
| Logs | Low (check enabled before building record) | Low-moderate | Serialization, export I/O |

---

## Default Configuration Values

Exact numeric defaults can shift between releases. For authoritative values, check the CHANGELOG or source constants:

- `BatchExportProcessor<T>` defaults: `src/OpenTelemetry/BatchExportProcessor.cs`
- `PeriodicExportingMetricReader` defaults: `src/OpenTelemetry/Metrics/Reader/PeriodicExportingMetricReader.cs`
- `OtlpExporterOptions` defaults: `src/OpenTelemetry.Exporter.OpenTelemetryProtocol/OtlpExporterOptions.cs`

| Parameter | Environment Variable |
|-----------|---------------------|
| Batch span queue size | `OTEL_BSP_MAX_QUEUE_SIZE` |
| Batch span export size | `OTEL_BSP_MAX_EXPORT_BATCH_SIZE` |
| Batch scheduled delay | `OTEL_BSP_SCHEDULE_DELAY` |
| Batch exporter timeout | _(see source)_ |
| Metric export interval | `OTEL_METRIC_EXPORT_INTERVAL` |
| Metric export timeout | `OTEL_METRIC_EXPORT_TIMEOUT` |
| OTLP export timeout | `OTEL_EXPORTER_OTLP_TIMEOUT` |

---

## Sampling

Sampling is the single most impactful performance lever for traces. Unsampled `Activity` objects are never recorded; they are represented as noop instances with near-zero overhead.

### Head Sampling

Configure a sampler at `TracerProviderBuilder` level:

```csharp
using var tracerProvider = Sdk.CreateTracerProviderBuilder()
    // Sample 10% of new traces; respect parent sampling decision for in-flight traces
    .SetSampler(new ParentBasedSampler(new TraceIdRatioBasedSampler(0.1)))
    .AddOtlpExporter()
    .Build();
```

`TraceIdRatioBasedSampler` makes sampling decisions based on the trace ID, ensuring consistent sampling across a distributed trace when applied without a `ParentBasedSampler` wrapper. `ParentBasedSampler` defers to the upstream decision when a parent context is present and applies the root sampler otherwise.

### Via Environment Variable

```bash
# "traceidratio" with ratio, "always_on", "always_off", "parentbased_traceidratio", etc.
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
```

See the [OpenTelemetry spec sampler names](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/#general-sdk-configuration) for valid values.

### Sampling Decision Impact

```
AlwaysOnSampler             -> Full Activity lifecycle: allocation + recording + export
ParentBasedSampler(0.1)     -> 90% noop (near-zero cost), 10% fully recorded
TraceIdRatioBasedSampler(0.1) -> Consistent 10% sampling ignoring parent
AlwaysOffSampler            -> All Activities noop; useful for load testing
```

> **Tail sampling**: For decisions based on complete traces (error rate, latency thresholds), use the OpenTelemetry Collector's tail sampling processor. Combine SDK head sampling with Collector tail sampling for a common production pattern.

---

## Batch Processor Tuning

`BatchActivityExportProcessor` buffers completed `Activity` objects in a queue and exports them asynchronously in batches. It wraps `BatchExportProcessor<Activity>`.

### How It Works

```
Application thread                 BatchActivityExportProcessor thread
      |                                          |
  activity.Stop() --enqueue-->  queue (MaxQueueSize)
      |                                          |
      |                            timer fires (ScheduledDelayMilliseconds)
      |                            OR batch full (MaxExportBatchSize)
      |                                          |
      |                             --export batch--> Exporter
```

### Configuration via OtlpExporterOptions

When using `AddOtlpExporter()`, the batch options are accessible through `OtlpExporterOptions.BatchExportProcessorOptions`:

```csharp
using var tracerProvider = Sdk.CreateTracerProviderBuilder()
    .AddSource("MyApp")
    .AddOtlpExporter(options =>
    {
        options.ExportProcessorType = ExportProcessorType.Batch;
        options.BatchExportProcessorOptions.MaxQueueSize = 4096;
        options.BatchExportProcessorOptions.MaxExportBatchSize = 1024;
        options.BatchExportProcessorOptions.ScheduledDelayMilliseconds = 10000;
    })
    .Build();
```

Check `BatchExportActivityProcessorOptions` in source for current defaults; they are also configurable via the `OTEL_BSP_*` environment variables.

### Tuning for Throughput

For high-volume services (many spans per second), increase queue and batch sizes to absorb bursts and reduce the number of export calls per unit time.

### Tuning for Latency

For services where trace delivery speed matters (live debugging, alerting), decrease `ScheduledDelayMilliseconds` and `MaxExportBatchSize` to export smaller batches more frequently.

### Queue Full Behavior

When the queue fills, new spans are dropped silently. Telemetry loss is preferred over application slowdown. The SDK emits internal diagnostic events when drops occur.

### SimpleActivityExportProcessor

`SimpleActivityExportProcessor` exports activities synchronously on `activity.Stop()`. It adds latency to every activity ending. Use it for:
- Tests and local development (deterministic behavior)
- Short-lived CLI tools that must flush before exit

```csharp
// Direct use (low-level)
var processor = new SimpleActivityExportProcessor(exporter);
```

When using `AddOtlpExporter()`, set `ExportProcessorType = ExportProcessorType.Simple`.

---

## Metric Reader Tuning

### PeriodicExportingMetricReader

`PeriodicExportingMetricReader` drives metric collection and export on a configurable interval. The metrics console exporter, for instance, flushes on this cycle.

```csharp
using var meterProvider = Sdk.CreateMeterProviderBuilder()
    .AddMeter("MyApp")
    .AddReader(new PeriodicExportingMetricReader(
        exporter: new OtlpMetricExporter(new OtlpExporterOptions()),
        exportIntervalMilliseconds: 30000,  // More frequent than the default
        exportTimeoutMilliseconds: 15000))
    .Build();
```

Check `PeriodicExportingMetricReader.cs` for the current defaults (`DefaultExportIntervalMilliseconds`, `DefaultExportTimeoutMilliseconds`).

Interval tradeoffs:
- Default: standard for dashboards and alerting
- Shorter intervals: near-real-time visibility, higher export overhead
- Longer intervals: reduced overhead, coarser freshness

---

## Views for Cardinality Control

Views filter or transform metric streams before aggregation, reducing the number of unique time series stored and exported. They are the primary lever for cardinality control in .NET.

### Drop High-Cardinality Attributes

Use `TagKeys` on `MetricStreamConfiguration` to allowlist the attributes to retain. Attributes not in the list are dropped before aggregation:

```csharp
using var meterProvider = Sdk.CreateMeterProviderBuilder()
    .AddMeter("MyApp")
    .AddView(
        instrumentName: "http.server.request.duration",
        metricStreamConfiguration: new MetricStreamConfiguration
        {
            // Keep only these attributes; drop everything else (e.g. user.id)
            TagKeys = new[] { "http.request.method", "http.response.status_code", "url.scheme" },
        })
    .AddOtlpExporter()
    .Build();
```

### Drop a Metric Entirely

```csharp
.AddView(
    instrumentName: "runtime.dotnet.gc.collections",
    metricStreamConfiguration: MetricStreamConfiguration.Drop)
```

### Wildcard / Predicate-Based Views

```csharp
.AddView(viewConfig: instrument =>
{
    // Drop all metrics whose name starts with "runtime."
    if (instrument.Name.StartsWith("runtime.", StringComparison.OrdinalIgnoreCase))
    {
        return MetricStreamConfiguration.Drop;
    }
    return null; // null = use default configuration
})
```

### Per-Instrument Cardinality Limit

```csharp
.AddView(
    instrumentName: "http.server.request.duration",
    metricStreamConfiguration: new MetricStreamConfiguration
    {
        CardinalityLimit = 500,  // Max unique tag combinations for this metric
    })
```

Metric attribute cardinality is the product of unique values across all recorded attributes. Attributes like user IDs or request IDs are unbounded. Attributes like HTTP method or status code are bounded. Always apply `TagKeys` allowlists to instruments that record unbounded attributes.

---

## Exporter Configuration

### Protocol Choice

| Aspect | gRPC (`OtlpExportProtocol.Grpc`) | HTTP/Protobuf (`OtlpExportProtocol.HttpProtobuf`) |
|--------|----------------------------------|---------------------------------------------------|
| Default port | 4317 | 4318 |
| .NET target | .NET only (deprecated for .NET Framework / .NET Standard) | All targets |
| Connection | Single long-lived connection | HTTP/1.1 or HTTP/2 |
| Best for | Persistent connections, high throughput | Firewalls, load balancers, .NET Framework |

> **Note**: `OtlpExportProtocol.Grpc` is marked `[Obsolete]` for .NET Framework / .NET Standard targets. For cross-target libraries, prefer `HttpProtobuf`. The default protocol on .NET is `Grpc`; on .NET Framework / .NET Standard it is `HttpProtobuf`. See `OtlpExporterOptions.cs` source for the compile-time conditional.

```csharp
.AddOtlpExporter(options =>
{
    options.Protocol = OtlpExportProtocol.HttpProtobuf;
    options.Endpoint = new Uri("http://collector:4318");
})
```

### Timeout

The OTLP exporter's `TimeoutMilliseconds` property controls how long an individual export attempt waits before being cancelled. Route the default value to `OtlpExporterOptions.cs` source rather than asserting it here; it is controlled by `OTEL_EXPORTER_OTLP_TIMEOUT` (milliseconds).

```csharp
.AddOtlpExporter(options =>
{
    options.TimeoutMilliseconds = 5000; // Override the default
})
```

### Retry

The OTLP exporter supports opt-in retry behavior via the experimental feature flag `OTEL_DOTNET_EXPERIMENTAL_OTLP_RETRY`. Valid values:
- `in_memory`: retry failed exports in memory
- `disk`: persist failed payloads to disk and retry later (requires `OTEL_DOTNET_EXPERIMENTAL_OTLP_DISK_RETRY_DIRECTORY_PATH`)

Retry is not enabled by default. For production resilience, consider placing an OpenTelemetry Collector on localhost and delegating retry/buffering to it.

### Exporter Processor Type

The OTLP exporter defaults to `ExportProcessorType.Batch`. Switch to `Simple` only for development or CLI tools:

```csharp
.AddOtlpExporter(options =>
{
    options.ExportProcessorType = ExportProcessorType.Simple; // Development only
})
```

---

## Async Context and Activity Propagation

### Automatic Flow via AsyncLocal

`Activity.Current` is backed by `AsyncLocal<Activity>`, part of the .NET runtime's `ActivitySource` implementation. This means the current span flows automatically across `await` points within a single logical async chain:

```csharp
// Activity.Current flows automatically into all awaited calls
using var activity = tracer.StartActiveSpan("outer");
await DoSomethingAsync(); // Activity.Current == "outer" inside here
await DoSomethingElseAsync(); // Still "outer"
```

No manual propagation is needed for standard `async`/`await` code.

### Non-Awaited Boundaries Require Manual Propagation

`Activity.Current` does NOT flow into fire-and-forget tasks, `Task.Run` with a captured context that diverges, or thread-pool callbacks that run after the originating scope has exited:

```csharp
using var activity = tracer.StartActiveSpan("producer");
var traceContext = activity?.Context; // Capture before forking

// Fire-and-forget: Activity.Current is NOT automatically available here
_ = Task.Run(async () =>
{
    // Must manually restore or propagate the context
    using var childActivity = tracer.StartActiveSpan(
        "consumer",
        SpanKind.Consumer,
        traceContext ?? default);
    await ProcessAsync();
});
```

For background workers, message consumers, and queued work items, extract the propagation context at the producer boundary and inject it into the work item payload (e.g., as W3C Trace Context headers), then restore it at the consumer boundary.

### Context Propagation Across Service Boundaries

Always extract context at ingress and inject at egress:

```csharp
// At ingress (e.g., ASP.NET Core middleware handles this automatically)
var propagator = Propagators.DefaultTextMapPropagator;
var context = propagator.Extract(default, request.Headers, (headers, name) =>
    headers.TryGetValue(name, out var value) ? new[] { value.ToString() } : []);

// At egress
propagator.Inject(new PropagationContext(activity.Context, Baggage.Current),
    request.Headers, (headers, name, value) => headers[name] = value);
```

---

## Graceful Shutdown

Proper shutdown ensures buffered telemetry is flushed before the process exits.

### With .NET Generic Host (Recommended)

When using `services.AddOpenTelemetry()`, the SDK registers a hosted service (`TelemetryHostedService`) that initializes providers on startup. Providers are registered as singletons and disposed by the DI container on host shutdown — no manual shutdown code is required:

```csharp
builder.Services.AddOpenTelemetry()
    .WithTracing(b => b.AddSource("MyApp").AddOtlpExporter())
    .WithMetrics(b => b.AddMeter("MyApp").AddOtlpExporter());

// On host.StopAsync() or Ctrl+C, providers are disposed automatically
var app = builder.Build();
await app.RunAsync();
```

### Without Generic Host

Dispose providers explicitly, or call `ForceFlush` before exit:

```csharp
using var tracerProvider = Sdk.CreateTracerProviderBuilder()
    .AddSource("MyApp")
    .AddOtlpExporter()
    .Build();

using var meterProvider = Sdk.CreateMeterProviderBuilder()
    .AddMeter("MyApp")
    .AddOtlpExporter()
    .Build();

// ... application runs ...

// On exit: Dispose flushes buffered data and shuts down exporters.
// The `using` declarations above handle this automatically at scope end.
```

### ForceFlush

`ForceFlush` synchronously flushes all buffered telemetry without shutting down the provider. Use it when you need to guarantee delivery before a checkpoint (e.g., before a blue/green swap, before a long sleep):

```csharp
// Traces
bool flushed = tracerProvider.ForceFlush(timeoutMilliseconds: 5000);

// Metrics
bool flushed = meterProvider.ForceFlush(timeoutMilliseconds: 5000);
```

`ForceFlush` returns `true` if all data was flushed within the timeout, `false` if the timeout was exceeded or a processor failed. Both `TracerProvider` and `MeterProvider` expose `ForceFlush` as extension methods in `OpenTelemetry`.

> **Hot path warning**: Calling `ForceFlush` on the request hot path synchronously exports all buffered data, adding latency to that request. Reserve it for shutdown paths or administrative endpoints.

---

## Pipeline Reliability

The OpenTelemetry .NET SDK is designed so that telemetry failures do not crash or block the application:

- **`Activity` creation never throws** — noop activities are returned on failure
- **Metric recording never throws** — measurements are silently dropped on failure
- **Export failures are logged internally** — via `EventSource`; check ETW/dotnet-trace for diagnostics
- **Queue overflow drops spans** — the application is not blocked

Export retries are opt-in (see the Retry section above). For reliable delivery in production, route telemetry through a local OpenTelemetry Collector that handles buffering and retry on your behalf.

---

## Sources of Truth

- Batch processor defaults: `src/OpenTelemetry/BatchExportProcessor.cs` and `src/OpenTelemetry/Trace/Processor/BatchExportActivityProcessorOptions.cs`
- Metric reader defaults: `src/OpenTelemetry/Metrics/Reader/PeriodicExportingMetricReader.cs`
- OTLP exporter options and protocol defaults: `src/OpenTelemetry.Exporter.OpenTelemetryProtocol/OtlpExporterOptions.cs`
- OTLP retry (experimental): `src/OpenTelemetry.Exporter.OpenTelemetryProtocol/Implementation/ExperimentalOptions.cs`
- Hosting lifecycle: `src/OpenTelemetry.Extensions.Hosting/Implementation/TelemetryHostedService.cs`
- Environment variable names: `OTEL_BSP_*`, `OTEL_METRIC_EXPORT_*`, `OTEL_EXPORTER_OTLP_*`
