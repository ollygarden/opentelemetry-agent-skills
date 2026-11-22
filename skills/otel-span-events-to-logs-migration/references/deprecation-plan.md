# OTEP 4430: Span Event API Deprecation Plan

This reference summarizes the [OTEP 4430](https://github.com/open-telemetry/opentelemetry-specification/blob/v1.62.0/oteps/4430-span-event-api-deprecation-plan.md) deprecation plan.

## Planned Deprecation Targets

- `Span.AddEvent` -- the API method for attaching events to spans
- `Span.RecordException` -- the API method for recording exceptions on spans

These methods are planned deprecation targets from the accepted OTEP; for their spec status, see [Current Status](#current-status).

## What Is NOT Being Deprecated

- Span Events as a concept -- they can still be emitted via the Logs API
- The ability to have events correlated with spans in the same proto envelope (via the SDK bridge)

## Why

- Provide a single consistent guidance: use the Logs API for events
- Reduce API surface duplication between spans and logs
- Events emitted via logs are more flexible (can be exported as logs, as span events, or both)

## Phased Rollout

### In the proto
Stabilize log-based Events. (The `event_name` LogRecord field is part of the stable proto.)

### Specification
1. Stabilize emitting exceptions and events via the Logs API
2. Mark `Span.RecordException` as deprecated
3. Mark `Span.AddEvent` as deprecated (can happen in parallel with 2)

### Per API and SDK
1. Implement and stabilize log-based exception and event emission
2. Implement the backward-compatibility SDK bridge (log processor that converts log-based events to span events)
3. Mark `Span.RecordException` as deprecated
4. Mark `Span.AddEvent` as deprecated (can happen in parallel with 3)

### Per Instrumentation
- Current major version: continue using existing span event methods
- Next major version: migrate to the Logs API; for span-detail-without-a-timestamp cases, record span attributes instead ([semantic-conventions#2010](https://github.com/open-telemetry/semantic-conventions/issues/2010), [opentelemetry-specification#4446](https://github.com/open-telemetry/opentelemetry-specification/issues/4446))
- Users opt into the SDK bridge if they need span events in the proto envelope

## Current Status

Values hold for the named releases; check the current release before relying on them. Sources: [proto v1.11.1](https://github.com/open-telemetry/opentelemetry-proto/blob/v1.11.1/opentelemetry/proto/logs/v1/logs.proto), [specification v1.62.0](https://github.com/open-telemetry/opentelemetry-specification/tree/v1.62.0/specification), and [semantic conventions v1.45.0](https://github.com/open-telemetry/semantic-conventions/blob/v1.45.0/docs/exceptions/exceptions-logs.md).

- **Proto 1.11.1**: log-based Events are stable; `event_name` is a stable LogRecord field.
- **Specification 1.62.0**: the Logs API is Stable, including the `event_name` field and the optional `Exception` parameter to Emit, so log-based exception/event emission is specified. The "event to span event bridge" `LogRecordProcessor` is specified in `specification/logs/sdk.md` (Status: Development), with a matching `event_to_span_event_bridge/development` declarative-config key.
- **Not yet done in spec**: `Span.AddEvent` and `Span.RecordException` are **not** yet marked Deprecated in `specification/trace/api.md`. That step remains pending.
- **Semantic conventions 1.45.0**: exception events normally use an operation-specific name with a `.exception` suffix; the generic `exception` name is for handlers not tied to an operation or domain. Existing instrumentations introducing log-based exceptions should use `OTEL_SEMCONV_EXCEPTION_SIGNAL_OPT_IN` with `logs` or `logs/dup` for a phased rollout while keeping span events as the default in the current major version.
- **SDKs**: implementation status varies. Do not claim all `AddEvent`/`RecordException` equivalents are deprecated unless that language SDK marks them so.

## Language Status

Use current released source for the target language before editing user code.

| Language | Current migration-relevant status |
|---|---|
| Go 1.47.0 (trace, log, sdk/log) | `trace.Span.AddEvent` / `RecordError` are present and not marked deprecated. The Logs API and SDK (`go.opentelemetry.io/otel/log`, `go.opentelemetry.io/otel/sdk/log`) are stable since 1.47.0; get loggers with `otel.Logger` from the root `go.opentelemetry.io/otel` package (`go.opentelemetry.io/otel/log/global` is deprecated). `log.Record.SetEventName` and `SetErr` are available; the log SDK derives `exception.type` and `exception.message` from the error. These releases have no event-to-span-event bridge implementation. |
| Java 1.68.0 | `Span.addEvent` / `recordException` are present and not marked deprecated; `LogRecordBuilder.setEventName` is available since 1.50.0 and `setException(Throwable)` since 1.60.0. The bridge is available in `opentelemetry-sdk-extension-incubator`; declarative config wires the `event_to_span_event_bridge/development` key to it ([details](backward-compat.md#via-declarative-configuration)). The older Java contrib bridge was removed in opentelemetry-java-contrib v1.60.0. |
| JavaScript / TypeScript API 1.10.0 / SDK 3.1.1 / experimental 0.301.2 | `Span.addEvent` / `recordException` are present and not marked deprecated. The Logs API is in `@opentelemetry/api` since 1.10.0 (`logs`, `SeverityNumber`); experimental 0.300.0 removed the standalone `@opentelemetry/api-logs` package. `@opentelemetry/sdk-logs` is stable since 3.0.0. `LogRecord.eventName` is supported; the `exception` log-record field is marked experimental. These releases have no event-to-span-event bridge implementation. |
| Python 1.45.2 / 0.66b2 | `span.add_event` / `record_exception` are present and not marked deprecated; `opentelemetry._logs.Logger.emit` accepts `event_name` and `exception`, and the SDK derives exception attributes. The standalone Events API/SDK is removed; use `LogRecord` with `event_name`. This release has no event-to-span-event bridge implementation. |
| .NET 1.21.0 | `ActivityExtensions.RecordException` is `[Obsolete]` and points to `Activity.AddException`; both record span events. `TelemetrySpan.AddEvent` / `RecordException` are not marked obsolete. `ILogger` supplies an event name through `EventId.Name`; the lower-level Logs API is pre-release. This release has no event-to-span-event bridge implementation. |
| Ruby API 1.11.1 / Logs API 0.7.0 / Logs SDK 0.8.1 | `Span#add_event` / `record_exception` are present and not marked deprecated. `Logger#on_emit` accepts `event_name` and inherits the current span context, but has no exception parameter; set `exception.*` attributes explicitly. Since Logs API 0.6.0 / SDK 0.8.0, `OpenTelemetry.logger_provider` is defined only after requiring `opentelemetry-logs-sdk` or `opentelemetry/logs/global`. These releases have no event-to-span-event bridge implementation. |

Tags: Go [v1.47.0](https://github.com/open-telemetry/opentelemetry-go/tree/v1.47.0); Java [v1.68.0](https://github.com/open-telemetry/opentelemetry-java/tree/v1.68.0); JavaScript [api/v1.10.0](https://github.com/open-telemetry/opentelemetry-js/tree/api/v1.10.0), [v3.1.1](https://github.com/open-telemetry/opentelemetry-js/tree/v3.1.1), [experimental/v0.301.2](https://github.com/open-telemetry/opentelemetry-js/tree/experimental/v0.301.2); Python [v1.45.2](https://github.com/open-telemetry/opentelemetry-python/tree/v1.45.2); .NET [core-1.21.0](https://github.com/open-telemetry/opentelemetry-dotnet/tree/core-1.21.0); Ruby [opentelemetry-api/v1.11.1](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-api/v1.11.1), [opentelemetry-logs-api/v0.7.0](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-logs-api/v0.7.0), [opentelemetry-logs-sdk/v0.8.1](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-logs-sdk/v0.8.1).

Treat the deprecation as the accepted direction, not a completed spec change; verify the current status of each step against the linked sources before making hard claims.

## Key Principle

The migration is about WHERE events are emitted (Logs API instead of Span API), not about removing event data from spans. The SDK bridge ensures span events can still appear in the span proto when needed.
