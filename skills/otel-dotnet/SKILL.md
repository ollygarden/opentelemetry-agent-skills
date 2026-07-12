---
name: otel-dotnet
description: OpenTelemetry in .NET — DI/builder SDK setup (OpenTelemetry.Extensions.Hosting, AddOpenTelemetry, UseOtlpExporter, OpenTelemetrySdk.Create), native BCL instrumentation (ActivitySource, System.Diagnostics.Metrics Meter, ILogger), zero-code CLR-profiler agent, contrib instrumentation packages, performance tuning, and breaking-change audits. Use when adding, reviewing, or configuring OpenTelemetry in a .NET or ASP.NET Core service. Triggers on "setup otel in dotnet", "dotnet telemetry", ".net tracing", "ASP.NET Core opentelemetry", "AddOpenTelemetry", "UseOtlpExporter", "OpenTelemetrySdk.Create", "ActivitySource", "System.Diagnostics.Metrics Meter", "ILogger opentelemetry", "OpenTelemetry.Extensions.Hosting", "opentelemetry-dotnet-instrumentation", or any C# OTel question.
---

# OpenTelemetry in .NET

Entry point for OpenTelemetry mechanics in .NET services. Load a reference below based on the
task; each reference is self-contained.

## References

| File | Use when |
|---|---|
| [`references/setup.md`](references/setup.md) | Setting up the SDK via the DI/builder path (`AddOpenTelemetry().WithTracing/WithMetrics/WithLogging`, `UseOtlpExporter`, `OpenTelemetrySdk.Create`, `OpenTelemetry.Extensions.Hosting`), exporter wiring, env-var / `IConfiguration` inputs, and why there is no declarative YAML config in .NET. |
| [`references/api.md`](references/api.md) | Instrumenting with native BCL APIs (`ActivitySource`/`Activity`, `System.Diagnostics.Metrics.Meter`, `ILogger`), how the SDK subscribes (`AddSource`/`AddMeter`), attributes, propagation, and the optional OTel API shim. |
| [`references/instrumentation-libraries.md`](references/instrumentation-libraries.md) | Zero-code (the `opentelemetry-dotnet-instrumentation` CLR-profiler agent), the contrib instrumentation-package catalog, and manual instrumentation following semconv. |
| [`references/performance.md`](references/performance.md) | Tuning sampling, batch export processor, periodic metric reader, views, exporter choice, async context, graceful shutdown. |
| [`references/breaking-changes.md`](references/breaking-changes.md) | Auditing existing code for deprecated/renamed APIs and semconv changes across recent core/contrib releases. |

## Sources of Truth

For YAML schema details (language-agnostic), see the `otel-declarative-config` skill — but note
.NET does not yet implement declarative config (see `references/setup.md`).
For .NET-specific facts:

| Fact | Fetch |
|---|---|
| Latest `OpenTelemetry` core / `core-*` tag | `gh api repos/open-telemetry/opentelemetry-dotnet/releases/latest -q '.tag_name'` |
| Latest NuGet package (`OpenTelemetry`, `OpenTelemetry.Extensions.Hosting`, exporters, `OpenTelemetry.Instrumentation.<pkg>`) | `WebFetch https://www.nuget.org/packages/<PackageId>` (Version) |
| Latest auto-instrumentation agent | `gh api repos/open-telemetry/opentelemetry-dotnet-instrumentation/releases/latest -q '.tag_name'` |
| Core CHANGELOG (per package) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet/main/src/<Package>/CHANGELOG.md` |
| Contrib CHANGELOG (per package) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-dotnet-contrib/main/src/<Package>/CHANGELOG.md` |
| .NET getting-started / instrumentation docs | `WebFetch https://opentelemetry.io/docs/languages/dotnet/instrumentation/` |
| Zero-code agent docs | `WebFetch https://opentelemetry.io/docs/zero-code/dotnet/` |

## Cross-References

- Schema-level facts: `otel-declarative-config` skill (language-agnostic YAML schema sources; .NET not yet implemented).
- SDK version selection across languages: `otel-sdk-versions` skill.
- Semantic conventions lookup: `otel-semantic-conventions` skill.
