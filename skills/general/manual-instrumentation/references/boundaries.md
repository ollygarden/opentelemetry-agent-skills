# Boundaries And Error Ownership

Instrumentation exists to explain runtime behavior in production. Add telemetry where it improves debugging or operational understanding. Skip it where it only confirms routine behavior.

## High-Value Boundaries

Create spans at boundaries such as:
- incoming HTTP, RPC, or message handling
- outgoing HTTP, RPC, database, cache, or messaging calls
- critical business operations with clear runtime meaning
- long-running workflows where progress and failure visibility matter

## Low-Value Locations

Do not create spans for:
- helper functions
- loop bodies
- serialization or mapping glue
- validation that only confirms the happy path
- pure computation or in-memory plumbing

If an operation has no diagnostic value outside the current stack frame, it usually should not be a span.

## Instrument The Unexpected

Prefer telemetry for:
- failures and degraded outcomes
- retries and fallback paths
- cache misses rather than every cache hit
- rate limits and circuit-breaker transitions
- slow or abnormal operations
- business milestones that need positive confirmation

## Error Ownership

Record an error on the span that owns the final failed outcome.

Use these rules:
- if the operation ends in failure, mark the owning span as failed
- if a retry later succeeds, keep retry evidence as events or attributes but do not mark the final span as failed
- if an error is handled and processing continues normally, record useful context without treating the span as a terminal failure

Avoid:
- recording the same failure on multiple nested spans without reason
- turning every exception into a terminal failure when the user-visible operation succeeded
- creating spans only to record errors
