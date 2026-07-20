# .NET Instrumentation API — BCL-First Reference

.NET instruments telemetry through native Base Class Library (BCL) APIs in
`System.Diagnostics` and `Microsoft.Extensions.Logging`. The OpenTelemetry SDK
**subscribes** to these APIs at startup; application code does not depend on OTel packages
at compile time.

## Sources of Truth

| Fact | Fetch |
|---|---|
| Latest `OpenTelemetry` core / `core-*` tag | `gh api repos/open-telemetry/opentelemetry-dotnet/releases/latest -q '.tag_name'` |
| Latest NuGet package versions | `WebFetch https://www.nuget.org/packages/<PackageId>` |
| Core CHANGELOG (per package) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet/main/src/<Package>/CHANGELOG.md` |
| Instrumentation docs | `WebFetch https://opentelemetry.io/docs/languages/dotnet/instrumentation/` |

---

## Traces — `System.Diagnostics.ActivitySource` / `Activity`

`ActivitySource` and `Activity` are the BCL tracing primitives, part of
`System.Diagnostics` (no extra package needed). The OTel SDK collects spans from any
source registered via `.AddSource(name)`.

### Creating a source

```csharp
// Declare once — typically as a static or instance field.
// Name must match what you pass to .AddSource() in SDK setup.
var activitySource = new ActivitySource("my-service");
```

### Starting a span

```csharp
// StartActivity returns null if no listener is subscribed (e.g. in test environments).
// Use a null-conditional or 'using var' — both patterns are idiomatic.
using var span = activitySource.StartActivity("manual-work");

// With explicit kind (default is Internal)
using var span = activitySource.StartActivity(
    "my-operation",
    ActivityKind.Server);
```

> **Verified spike:** `activitySource.StartActivity("manual-work")` produced the
> `manual-work` span in console output with `Activity.Kind: Internal` and correct
> `TraceId`/`SpanId`.

### Setting tags (attributes)

```csharp
span?.SetTag("http.request.method", "GET");
span?.SetTag("db.system.name", "postgresql");
```

Tags become span attributes in the OTLP export. For key names, follow semantic conventions
— see the `otel-semantic-conventions` skill.

### Setting status

```csharp
// Success (implicit when span ends without error)
span?.SetStatus(ActivityStatusCode.Ok);

// Error
span?.SetStatus(ActivityStatusCode.Error, "something went wrong");
```

### Adding events

```csharp
span?.AddEvent("cache-miss");

// Event with attributes
var tags = new ActivityTagsCollection
{
    { "cache.key", "user:42" },
};
span?.AddEvent("cache-miss", DateTimeOffset.UtcNow, tags);
```

### Adding links

```csharp
// Link to another span by its ActivityContext
var link = new ActivityLink(otherActivity.Context);
using var span = activitySource.StartActivity(
    "batch-process",
    ActivityKind.Internal,
    parentContext: default,
    links: new[] { link });
```

### `Activity.Current` — ambient context

`Activity.Current` is an `AsyncLocal`-backed property. The runtime maintains it
automatically across `await` boundaries; you do not need to pass context explicitly in
most cases.

```csharp
// Read the current active span from anywhere in the call stack
var current = Activity.Current;
var traceId = current?.TraceId.ToString();
var spanId  = current?.SpanId.ToString();
```

### SDK subscription

The SDK collects spans only from sources you name at startup:

```csharp
.WithTracing(t => t
    .AddSource("my-service")          // matches ActivitySource name
    .AddAspNetCoreInstrumentation()   // subscribes to Microsoft.AspNetCore source
    ...
)
```

Spans from un-subscribed sources are silently dropped — this is intentional, not a bug.

---

## Metrics — `System.Diagnostics.Metrics.Meter`

`Meter` and its instrument types are part of `System.Diagnostics.Metrics` (BCL, no extra
package). The OTel SDK collects metrics from any meter registered via `.AddMeter(name)`.

### Creating a meter

```csharp
// Declare once — name must match what you pass to .AddMeter() in SDK setup.
var meter = new Meter("my-service");
```

### Counter

Monotonically increasing. Use for counts of events, requests, errors.

```csharp
var hits = meter.CreateCounter<long>("validation.hits");

// Record with attributes
hits.Add(1, new KeyValuePair<string, object?>("route", "/work"));
```

> **Verified spike:** `meter.CreateCounter<long>("validation.hits")` + `.Add(1, ...)` was
> confirmed in console output as `Metric Name: validation.hits, Metric Type: LongSum`.

### Histogram

Distribution of values. Use for durations, sizes, latencies.

```csharp
var duration = meter.CreateHistogram<double>("request.duration", unit: "ms");

duration.Record(123.4, new KeyValuePair<string, object?>("route", "/work"));
```

### UpDownCounter

Non-monotonic sum. Use for queue depths, active connections, in-flight requests.

```csharp
var queueDepth = meter.CreateUpDownCounter<long>("queue.depth");

queueDepth.Add(1);   // item enqueued
queueDepth.Add(-1);  // item dequeued
```

### Observable instruments (asynchronous / pull-based)

Observable instruments report their value via a callback, invoked by the SDK when it
collects metrics (e.g. on each periodic-reader interval). Use them for values you poll
(memory, CPU, connection pool size) rather than values you increment on events.

```csharp
// Observable gauge — reports the current value on each collection
meter.CreateObservableGauge<long>(
    "process.memory.bytes",
    observeValue: () => GC.GetTotalMemory(forceFullCollection: false));

// Observable counter — cumulative monotonic value (e.g. total GC collections)
meter.CreateObservableCounter<long>(
    "gc.collections",
    observeValue: () => GC.CollectionCount(0));

// Observable up-down counter — non-monotonic cumulative value
meter.CreateObservableUpDownCounter<long>(
    "thread.pool.queue.length",
    observeValue: () => ThreadPool.PendingWorkItemCount);
```

### SDK subscription

```csharp
.WithMetrics(m => m
    .AddMeter("my-service")   // matches Meter name
    ...
)
```

Instruments from un-subscribed meters are silently dropped.

---

## Logs — `Microsoft.Extensions.Logging.ILogger`

Logging in .NET goes through `ILogger` / `ILogger<T>` from
`Microsoft.Extensions.Logging`. The OTel SDK captures log records from the standard
logging pipeline — no separate "OTel logger" is needed in application code.

### Injecting and using `ILogger`

```csharp
// In a minimal-API handler (injected by the DI container)
app.MapGet("/work", (ILogger<Program> logger) =>
{
    logger.LogInformation("handled /work");
    return Results.Ok(new { ok = true });
});

// In a class (constructor injection)
public class OrderService(ILogger<OrderService> logger)
{
    public void Process(Order order)
    {
        logger.LogInformation("Processing order {OrderId}", order.Id);
    }
}
```

Structured log parameters (`{OrderId}`) are captured as log attributes.

> **Verified spike:** `logger.LogInformation("handled /work")` appeared in console output
> with `LogRecord.TraceId` and `LogRecord.SpanId` matching the active `manual-work` span,
> confirming automatic trace-log correlation.

### How OTel captures logs

The SDK attaches to the `ILoggingBuilder` pipeline:

```csharp
// Option A — co-located with other OTel config (OpenTelemetry.Extensions.Hosting)
builder.Services.AddOpenTelemetry()
    .WithLogging(l => l.AddOtlpExporter());

// Option B — via ILoggingBuilder directly
builder.Logging.AddOpenTelemetry(o =>
{
    o.IncludeScopes = true;
    o.AddOtlpExporter();
});
```

Both options work; Option A keeps all OTel wiring co-located.

### Trace-log correlation

When a log statement executes inside an active `Activity` (span), the SDK automatically
stamps `TraceId`, `SpanId`, and `TraceFlags` onto the log record. No extra code is needed.

---

## Attributes / Tags

Both `Activity.SetTag` (traces) and metric instrument `.Add(...)` / `.Record(...)` accept
key-value attributes. Use semantic-convention names for interoperability:

- Traces: `http.request.method`, `db.system.name`, `server.address`, `error.type`, etc.
- Metrics: `http.route`, `rpc.method`, `messaging.system`, etc.

For the authoritative attribute registry, consult the `otel-semantic-conventions` skill.

**Types supported in BCL instruments:**

```csharp
// Tags on Activity
span?.SetTag("key", "string-value");
span?.SetTag("key", 42);       // int
span?.SetTag("key", 3.14);     // double
span?.SetTag("key", true);     // bool

// Attributes on metric instruments — pass as KeyValuePair or TagList
hits.Add(1, new KeyValuePair<string, object?>("route", "/work"));

// TagList (struct, zero-allocation for small sets)
var tags = new TagList
{
    { "route", "/work" },
    { "status", 200 },
};
hits.Add(1, tags);
```

---

## Propagation

### How context flows

`Activity.Current` is backed by `AsyncLocal<T>`. The runtime propagates it automatically
across `await` boundaries, thread-pool callbacks, and `Task` continuations. You do not
pass a context object explicitly — the active span is always available via
`Activity.Current`.

### Default format: W3C TraceContext

The SDK uses W3C TraceContext (`traceparent` / `tracestate` headers) by default. For
ASP.NET Core apps with `AddAspNetCoreInstrumentation()`, incoming `traceparent` headers are
extracted and the `Activity` is parented correctly without extra code.

### Manual inject/extract

For transports not automatically instrumented (custom TCP protocols, message queues,
background jobs via a custom dispatcher), use the `Propagators` API:

```csharp
using System.Diagnostics;
using OpenTelemetry;
using OpenTelemetry.Context.Propagation;

// --- Injecting (producer / outbound side) ---
var propagator = Propagators.DefaultTextMapPropagator;
var carrier = new Dictionary<string, string>();
propagator.Inject(
    new PropagationContext(Activity.Current?.Context ?? default, Baggage.Current),
    carrier,
    (dict, key, value) => dict[key] = value);
// Serialize 'carrier' alongside your message/request.

// --- Extracting (consumer / inbound side) ---
var parentContext = propagator.Extract(
    default,
    carrier,
    (dict, key) => dict.TryGetValue(key, out var v) ? new[] { v } : Array.Empty<string>());

using var span = activitySource.StartActivity(
    "process-message",
    ActivityKind.Consumer,
    parentContext.ActivityContext);
```

Once the OpenTelemetry SDK is initialized, `Propagators.DefaultTextMapPropagator`
defaults to W3C TraceContext + Baggage. The .NET SDK does not read
`OTEL_PROPAGATORS`; replace the default programmatically with
`Sdk.SetDefaultTextMapPropagator(...)` when a different propagator is required.

---

## Optional: OpenTelemetry API Shim

> **Note:** An `OpenTelemetry.Api` tracing shim exists for developers who prefer
> OTel trace terminology (`Tracer`/`TelemetrySpan`) over `ActivitySource`/`Activity`.
> It wraps the native BCL tracing types under the hood; metrics still use
> `System.Diagnostics.Metrics.Meter` and its instruments directly.
>
> The BCL path (`System.Diagnostics.ActivitySource`, `System.Diagnostics.Metrics.Meter`)
> is the recommended and idiomatic choice for .NET. Prefer it unless you are porting
> multi-language instrumentation code or an existing OTel-API-based library.
>
> See: `WebFetch https://opentelemetry.io/docs/languages/dotnet/instrumentation/` for
> the shim's package and usage.
