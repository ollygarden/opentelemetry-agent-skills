# .NET Instrumentation Libraries

## Detecting Existing Instrumentation

Before adding instrumentation, check whether it is already in place.

**In-process packages** — scan `.csproj` files for `OpenTelemetry.Instrumentation.*` references:

```bash
grep -r "OpenTelemetry\.Instrumentation\." --include="*.csproj" .
```

**SDK wiring** — search startup code for `AddOpenTelemetry()`:

```bash
grep -r "AddOpenTelemetry\(\)" --include="*.cs" .
```

**Zero-code agent** — check whether the CLR Profiler env vars are set in the process environment
or the container/service definition:

```bash
env | grep -E "CORECLR_ENABLE_PROFILING|CORECLR_PROFILER|OTEL_DOTNET_AUTO_HOME"
```

If the profiler vars are present, the `opentelemetry-dotnet-instrumentation` agent is active and
is injecting instrumentation without code changes.

---

## Zero-Code Path: `opentelemetry-dotnet-instrumentation`

The [`opentelemetry-dotnet-instrumentation`](https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation)
agent instruments .NET applications without source changes by loading a CLR Profiler at startup.

**Source of Truth:** [https://opentelemetry.io/docs/zero-code/dotnet/](https://opentelemetry.io/docs/zero-code/dotnet/)
Do not pin the agent version in prose; fetch the latest release from the repo or the zero-code docs.

### Install

**Linux / macOS:**

```bash
curl -sSfL https://github.com/open-telemetry/opentelemetry-dotnet-instrumentation/releases/latest/download/otel-dotnet-auto-install.sh \
  | bash
```

The script downloads the agent to `$OTEL_DOTNET_AUTO_HOME` (defaults to `$HOME/.otel-dotnet-auto`).

**Windows (PowerShell):**

The Windows install uses a PowerShell module flow. For the exact current command, see the
[zero-code .NET docs](https://opentelemetry.io/docs/zero-code/dotnet/) — the Source of Truth
for Windows installation steps and script filenames.

### Activation (CLR Profiler)

The install script generates an `instrument.sh` (Linux/macOS) or sets registry/env values (Windows).
Source it before running the process:

```bash
. $OTEL_DOTNET_AUTO_HOME/instrument.sh
dotnet run
```

The script exports these required vars:

| Variable | Purpose |
|----------|---------|
| `CORECLR_ENABLE_PROFILING` | Must be `1` to activate the profiler |
| `CORECLR_PROFILER` | GUID identifying the OTel profiler |
| `OTEL_DOTNET_AUTO_HOME` | Path to the agent installation directory |

### Configuration

Configure via standard `OTEL_*` env vars:

```bash
export OTEL_SERVICE_NAME=my-service
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_TRACES_EXPORTER=otlp
export OTEL_METRICS_EXPORTER=otlp
export OTEL_LOGS_EXPORTER=otlp
```

For the full list of env vars (including `OTEL_DOTNET_AUTO_*` agent-specific vars), see the
[zero-code .NET docs](https://opentelemetry.io/docs/zero-code/dotnet/).

---

## In-Process Library Instrumentation

The in-process alternative to the zero-code agent: add NuGet packages and register them in
the OTel builder. These run inside the application process and give the most control over
which instrumentation is active.

The snippet below is the verified baseline from the spike (all three signals confirmed):

```csharp
builder.Services.AddOpenTelemetry()
    .ConfigureResource(r => r.AddService(ServiceName, serviceVersion: "0.1.0"))
    .WithTracing(t => t
        .AddSource(ServiceName)
        .AddAspNetCoreInstrumentation()   // OpenTelemetry.Instrumentation.AspNetCore
        .AddHttpClientInstrumentation()   // OpenTelemetry.Instrumentation.Http
        .AddSqlClientInstrumentation()    // OpenTelemetry.Instrumentation.SqlClient
        .AddOtlpExporter())
    .WithMetrics(m => m
        .AddMeter(ServiceName)
        .AddAspNetCoreInstrumentation()   // also emits HTTP server metrics
        .AddHttpClientInstrumentation()   // also emits HTTP client metrics
        .AddOtlpExporter())
    .WithLogging(l => l
        .AddOtlpExporter());
```

See `references/setup.md` for the full `AddOpenTelemetry()` setup pattern.

---

## Contrib Catalog

All packages below are confirmed to exist under
[`open-telemetry/opentelemetry-dotnet-contrib/src/`](https://github.com/open-telemetry/opentelemetry-dotnet-contrib/tree/main/src).

**Version rule:** instrumentation packages follow their own release cadence and are not
in lockstep with the core SDK. Do not pin versions in prose — fetch the current version
for each package from NuGet:

```
https://www.nuget.org/packages/<PackageId>
```

Or from the per-package CHANGELOG in the contrib repo (source of truth for breaking changes):

```
https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet-contrib/main/src/<Package>/CHANGELOG.md
```

### Web / HTTP

| Package | Builder extension | NuGet |
|---------|-------------------|-------|
| `OpenTelemetry.Instrumentation.AspNetCore` | `.AddAspNetCoreInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.AspNetCore) |
| `OpenTelemetry.Instrumentation.Http` | `.AddHttpClientInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.Http) |
| `OpenTelemetry.Instrumentation.AspNet` | `.AddAspNetInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.AspNet) — .NET Framework only |

### Data

| Package | Builder extension | NuGet |
|---------|-------------------|-------|
| `OpenTelemetry.Instrumentation.SqlClient` | `.AddSqlClientInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.SqlClient) |
| `OpenTelemetry.Instrumentation.EntityFrameworkCore` | `.AddEntityFrameworkCoreInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.EntityFrameworkCore) |
| `OpenTelemetry.Instrumentation.StackExchangeRedis` | `.AddRedisInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.StackExchangeRedis) |

### RPC

| Package | Builder extension | NuGet |
|---------|-------------------|-------|
| `OpenTelemetry.Instrumentation.GrpcNetClient` | `.AddGrpcClientInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.GrpcNetClient) |
| `OpenTelemetry.Instrumentation.Wcf` | `.AddWcfInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.Wcf) |

### Messaging

| Package | Builder extension | NuGet |
|---------|-------------------|-------|
| `OpenTelemetry.Instrumentation.ConfluentKafka` | `.AddKafkaConsumerInstrumentation<TKey,TValue>()` / `.AddKafkaProducerInstrumentation<TKey,TValue>()` (both available on `TracerProviderBuilder` and `MeterProviderBuilder`) | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.ConfluentKafka) |
| `OpenTelemetry.Instrumentation.MassTransit` | `.AddMassTransitInstrumentation()` — **deprecated; MassTransit ≤ v7 only**. For MassTransit v8+ use built-in support: call `.AddSource("MassTransit")` on the `TracerProviderBuilder` instead. | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.MassTransit) |
| `OpenTelemetry.Instrumentation.Hangfire` | `.AddHangfireInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.Hangfire) |

### Runtime / System

| Package | Builder extension | NuGet |
|---------|-------------------|-------|
| `OpenTelemetry.Instrumentation.Runtime` | `.AddRuntimeInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.Runtime) — emits GC/thread metrics |
| `OpenTelemetry.Instrumentation.Process` | `.AddProcessInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.Process) — CPU/memory metrics |
| `OpenTelemetry.Instrumentation.EventCounters` | `.AddEventCountersInstrumentation()` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Instrumentation.EventCounters) |

### Cloud / Infrastructure

| Package | NuGet |
|---------|-------|
| `OpenTelemetry.Resources.AWS` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Resources.AWS) |
| `OpenTelemetry.Resources.Azure` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Resources.Azure) |
| `OpenTelemetry.Resources.Container` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Resources.Container) |
| `OpenTelemetry.Resources.Gcp` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Resources.Gcp) |
| `OpenTelemetry.Resources.Host` | [NuGet](https://www.nuget.org/packages/OpenTelemetry.Resources.Host) |

---

## Manual Instrumentation Patterns

Use these when no contrib package covers the target library. Attributes must follow semantic
conventions — load the `otel-semantic-conventions` skill for the authoritative attribute names.

These patterns use the native BCL `ActivitySource` API (the primary path for .NET), not the
OTel API shim (`tracerProvider.GetTracer(...)`). Declare a module-level source and start
activities from it directly.

### HTTP Client Call

```csharp
using System.Diagnostics;

// Module-level — one instance per library/component
static readonly ActivitySource ActivitySource = new("my-service");

// At call site:
using var activity = ActivitySource.StartActivity("GET", ActivityKind.Client);
activity?.SetTag("http.request.method", "GET");
activity?.SetTag("url.full", requestUrl);

var response = await httpClient.GetAsync(requestUrl);
activity?.SetTag("http.response.status_code", (int)response.StatusCode);
```

Semconv reference: `otel-semantic-conventions` → HTTP client spans.

### Database Call

```csharp
using var activity = ActivitySource.StartActivity("SELECT users", ActivityKind.Client);
activity?.SetTag("db.system.name", "postgresql");
activity?.SetTag("db.namespace", databaseName);
activity?.SetTag("db.operation.name", "SELECT");
activity?.SetTag("db.collection.name", "users");

// execute query …

if (ex != null)
{
    activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
    activity?.SetTag("error.type", ex.GetType().Name);
}
```

Semconv reference: `otel-semantic-conventions` → database spans.

### Background Job

```csharp
using var activity = ActivitySource.StartActivity("process batch", ActivityKind.Internal);
activity?.SetTag("batch.id", batchId);
activity?.SetTag("worker.id", workerId);

var items = await repo.GetBatchItems(batchId);
activity?.SetTag("batch.size", items.Count);

int processed = 0;
foreach (var item in items)
{
    try { await ProcessItem(item); processed++; }
    catch (Exception ex)
    {
        logger.LogWarning(ex, "item {ItemId} failed", item.Id);
    }
}

activity?.SetTag("batch.processed", processed);
activity?.SetTag("batch.failed", items.Count - processed);
```

Semconv reference: `otel-semantic-conventions` → messaging/general spans (use `messaging.system`,
`messaging.operation`, etc. for queue-backed jobs; omit if purely internal).

---

## Enriching Auto-Instrumented Spans

When `AddAspNetCoreInstrumentation()` or the zero-code agent has already created a span,
add business attributes to it via `Activity.Current` rather than starting a new span:

```csharp
app.MapGet("/orders/{id}", async (string id, IOrderService svc) =>
{
    // Span already created by AddAspNetCoreInstrumentation()
    Activity.Current?.SetTag("business.operation", "order_lookup");
    Activity.Current?.SetTag("order.id", id);

    var order = await svc.GetOrder(id);

    if (order is null)
    {
        Activity.Current?.SetStatus(ActivityStatusCode.Error, "order not found");
        return Results.NotFound();
    }

    Activity.Current?.SetTag("customer.tier", order.CustomerTier);
    return Results.Ok(order);
});
```

`Activity.Current` is the ambient span set by the instrumentation library. Calling
`SetTag` on it adds attributes to the existing span without creating additional spans.
