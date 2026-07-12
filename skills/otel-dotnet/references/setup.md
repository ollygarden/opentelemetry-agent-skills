# .NET SDK Setup — DI/Builder Path

The idiomatic .NET setup is the DI/builder path: call `AddOpenTelemetry()` on the service
collection, then chain `.ConfigureResource(...)`, `.WithTracing(...)`, `.WithMetrics(...)`,
and `.WithLogging(...)` via `OpenTelemetry.Extensions.Hosting`. For non-hosted
multi-signal processes on SDK 1.10.0 or newer, use `OpenTelemetrySdk.Create(...)`.

## Sources of Truth

| Fact | Fetch |
|---|---|
| Latest `OpenTelemetry` core / `core-*` tag | `gh api repos/open-telemetry/opentelemetry-dotnet/releases/latest -q '.tag_name'` |
| Latest NuGet package (`OpenTelemetry`, `OpenTelemetry.Extensions.Hosting`, exporters) | `WebFetch https://www.nuget.org/packages/<PackageId>` (Version) |
| Core CHANGELOG (per package) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet/main/src/<Package>/CHANGELOG.md` |
| .NET getting-started / instrumentation docs | `WebFetch https://opentelemetry.io/docs/languages/dotnet/instrumentation/` |

## Hosted Setup

For ASP.NET Core or any service using the .NET generic host, install
`OpenTelemetry.Extensions.Hosting` plus the exporter package you use (for example
`OpenTelemetry.Exporter.OpenTelemetryProtocol` for `AddOtlpExporter()`), and call
`AddOpenTelemetry()` on `builder.Services`.
The snippet below is the verified baseline from the spike (all three signals produced
working output):

```csharp
using System.Diagnostics;
using System.Diagnostics.Metrics;
using OpenTelemetry.Logs;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

var builder = WebApplication.CreateBuilder(args);

const string ServiceName = "my-service";
var activitySource = new ActivitySource(ServiceName);
var meter = new Meter(ServiceName);

builder.Services.AddOpenTelemetry()
    .ConfigureResource(r => r.AddService(ServiceName, serviceVersion: "0.1.0"))
    .WithTracing(t => t
        .AddSource(ServiceName)
        .AddAspNetCoreInstrumentation()
        .AddOtlpExporter())
    .WithMetrics(m => m
        .AddMeter(ServiceName)
        .AddOtlpExporter())
    .WithLogging(l => l
        .AddOtlpExporter());

var app = builder.Build();
// ... routes, app.Run()
```

Replace `AddOtlpExporter()` with `AddConsoleExporter()` during local development
(requires `OpenTelemetry.Exporter.Console`). The console exporter was used in the spike to
confirm all three signals.

**OTLP shortcut:** If the process exports all enabled signals through the same OTLP
destination, `OpenTelemetry.Exporter.OpenTelemetryProtocol` also exposes
`.UseOtlpExporter()` on the `AddOpenTelemetry()` builder. It automatically enables the
OTLP exporter for logs, metrics, and traces, supports signal-specific `OTEL_EXPORTER_OTLP_*`
overrides, and must not be mixed with signal-specific `.AddOtlpExporter()` calls.

**Logging alternative — `ILoggingBuilder`:** Instead of `.WithLogging(...)` on the OTel
builder, you can wire logging through the standard host logging builder:

```csharp
builder.Logging.AddOpenTelemetry(o =>
{
    o.IncludeScopes = true;
    o.AddOtlpExporter();
});
```

Both approaches work; `WithLogging(...)` keeps all OTel config co-located.

**Resource:** `ConfigureResource` accepts any `ResourceBuilder` callback. The resource is
shared across all three signals, which is why `service.name`, `service.version`, and
`telemetry.sdk.*` appear on every export.

## Non-Hosted Setup

For console apps or worker processes that do not use the generic host, SDK 1.10.0 and newer
support `OpenTelemetrySdk.Create(...)` as the single lifecycle boundary for all enabled
signals:

```csharp
using OpenTelemetry;
using OpenTelemetry.Logs;
using OpenTelemetry.Metrics;
using OpenTelemetry.Trace;

using var sdk = OpenTelemetrySdk.Create(b => b
    .ConfigureResource(r => r.AddService("my-console-app"))
    .WithTracing(t => t
        .AddSource("my-source")
        .AddOtlpExporter())
    .WithMetrics(m => m
        .AddMeter("my-meter")
        .AddOtlpExporter())
    .WithLogging(l => l
        .AddOtlpExporter()));
```

Disposing the returned `OpenTelemetrySdk` flushes and shuts down all configured signals.
For older SDKs or when you need separate provider lifetimes, build and own the providers
manually:

```csharp
using OpenTelemetry;
using OpenTelemetry.Trace;
using OpenTelemetry.Metrics;

// Tracing
using var tracerProvider = Sdk.CreateTracerProviderBuilder()
    .AddSource("my-source")
    .AddOtlpExporter()
    .Build();

// Metrics
using var meterProvider = Sdk.CreateMeterProviderBuilder()
    .AddMeter("my-meter")
    .AddOtlpExporter()
    .Build();

// Logging (host-free): use LoggerFactory.Create(b => b.AddOpenTelemetry(...))
// or OpenTelemetryLoggerProvider directly. The ILoggingBuilder route shown in
// the Hosted section above requires the generic host and does not apply here.
```

`using var` ensures `Dispose()` is called on exit, which flushes the batch processors and
shuts down the providers. For explicit control, call
`tracerProvider.Shutdown()` / `meterProvider.Shutdown()` before the process exits.

## Configuration Inputs

The .NET SDK reads configuration from two sources; they can be combined:

**`OTEL_*` environment variables** — the standard cross-language signal. Examples:

```sh
export OTEL_SERVICE_NAME=my-service
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_TRACES_SAMPLER=parentbased_traceidratio
export OTEL_TRACES_SAMPLER_ARG=0.1
```

Env vars are read automatically by the SDK; no extra code is needed once `AddOpenTelemetry()`
is called.

**`IConfiguration` / `appsettings.json`** — the .NET-native config system. Pass
configuration values into the builder lambdas as you would any other config:

```json
// appsettings.json
{
  "OTel": {
    "ServiceName": "my-service",
    "Endpoint": "http://localhost:4317"
  }
}
```

```csharp
var cfg = builder.Configuration;
builder.Services.AddOpenTelemetry()
    .ConfigureResource(r => r.AddService(cfg["OTel:ServiceName"]!))
    .WithTracing(t => t
        .AddOtlpExporter(o => o.Endpoint = new Uri(cfg["OTel:Endpoint"]!)));
```

`IConfiguration` is the standard .NET way to surface config from JSON files, env vars,
Azure Key Vault, and other providers — it is not the OTel declarative YAML format (see
below).

## No Declarative Config in .NET

> **Note:** .NET does not implement the OpenTelemetry declarative YAML (`file_format`)
> specification. There is no `file_format:` config file equivalent in the .NET SDK.
>
> Upstream tracking issue:
> [open-telemetry/opentelemetry-dotnet#6380](https://github.com/open-telemetry/opentelemetry-dotnet/issues/6380)
>
> To check current status, fetch the issue or the latest core CHANGELOG (see Sources of
> Truth above). For the language-agnostic YAML schema, load the `otel-declarative-config`
> skill.

Use `OTEL_*` env vars and `IConfiguration` (described above) for runtime configuration
until declarative config lands.
