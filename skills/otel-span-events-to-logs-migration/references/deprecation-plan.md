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

### Specification
1. Stabilize log-based exceptions and events
2. Mark `Span.RecordException` as deprecated
3. Mark `Span.AddEvent` as deprecated

### Per SDK
1. Implement log-based event emission
2. Implement the backward-compatibility SDK bridge (log processor that converts log events to span events)
3. Mark the span methods as deprecated

### Per Instrumentation
- Current major version: continue using existing span event methods
- Next major version: migrate to the Logs API
- Users opt into the SDK bridge if they need span events in the proto envelope

## Key Principle

The migration is about WHERE events are emitted (Logs API instead of Span API), not about removing event data from spans. The SDK bridge ensures span events can still appear in the span proto when needed.
