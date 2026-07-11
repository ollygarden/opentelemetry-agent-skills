# OTEP 4430: Span Event API Deprecation Plan

This reference summarizes the [OTEP 4430](https://github.com/open-telemetry/opentelemetry-specification/blob/main/oteps/4430-span-event-api-deprecation-plan.md) deprecation plan.

## What Is Being Deprecated

- `Span.AddEvent` -- the API method for attaching events to spans
- `Span.RecordException` -- the API method for recording exceptions on spans

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

## Current Status (2026-07)

- **Proto**: log-based Events are stable; `event_name` is a stable LogRecord field.
- **Spec**: the Logs API is Stable, including the `event_name` field and the optional `Exception` parameter to Emit, so log-based exception/event emission is specified. The "event to span event bridge" `LogRecordProcessor` is now specified in `specification/logs/sdk.md` (Status: Development), with a matching `event_to_span_event_bridge/development` declarative-config key.
- **Not yet done in spec**: `Span.AddEvent` and `Span.RecordException` are **not** yet marked Deprecated in `specification/trace/api.md`. That step remains pending.
- **SDKs**: some have started deprecating equivalents ahead of the spec (e.g., OpenTelemetry .NET's `Activity.RecordException` extension is `[Obsolete]`, pointing to `Activity.AddException`).

Treat the deprecation as the accepted direction, not a completed spec change; verify the current status of each step against the linked sources before making hard claims.

## Key Principle

The migration is about WHERE events are emitted (Logs API instead of Span API), not about removing event data from spans. The SDK bridge ensures span events can still appear in the span proto when needed.
