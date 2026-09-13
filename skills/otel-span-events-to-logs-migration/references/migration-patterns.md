# Migration Patterns: Before and After

Language-specific patterns for migrating from `AddEvent` / `RecordException` to the Logs API.

When applying these patterns, always check the project's actual SDK imports and version. The examples below use idiomatic API calls; adapt to the project's style.

Do not assume the span-event method is formally deprecated in the target SDK. The migration target comes from OTEP 4430 and per-language release status.

In exception examples, `exceptionEventName` / `exception_event_name` is the
event name defined by the applicable semantic convention, normally an
operation-specific name with a `.exception` suffix. Use the generic
`exception` name only for handlers not tied to an operation or domain.

## Go

### Exception Recording

Before:
```go
span.RecordError(err)
span.SetStatus(codes.Error, err.Error())
```

After:
```go
// Use the event logger to record the exception as a log-based event
logger := global.Logger("my-package")
record := log.Record{}
record.SetTimestamp(time.Now())
record.SetEventName(exceptionEventName)
record.SetSeverity(log.SeverityError)
record.SetErr(err) // the SDK derives exception.type and exception.message
// exception.stacktrace is intentionally omitted: the Go error value does
// not carry its origin stack, and capturing runtime.Stack here would
// point at the emit site, not where the error arose. Per semconv, the
// attribute is Recommended, not Required. If the project uses an error
// library that preserves the origin stack (e.g., github.com/pkg/errors,
// github.com/cockroachdb/errors), see the note below. Do not call
// runtime.Stack here.
logger.Emit(ctx, record)
span.SetStatus(codes.Error, err.Error())
```

Go log SDK 0.22.0 derives `exception.type` and `exception.message` from
`SetErr`. An explicitly supplied exception attribute is preserved, while each
missing `exception.type` or `exception.message` attribute is derived independently.
When an error library preserves the origin stack, add `exception.stacktrace` and
let the SDK derive whichever of type or message you did not set explicitly.

Prefer `record.SetEventName(name)` over adding an `event.name` attribute in Go: it is a first-class field on `log.Record` and produces cleaner output in backends that special-case event records.

### General Event

Before:
```go
span.AddEvent("cache.miss", trace.WithAttributes(
    attribute.String("cache.key", key),
))
```

After:
```go
logger := global.Logger("my-package")
record := log.Record{}
record.SetTimestamp(time.Now())
record.SetEventName("cache.miss")
record.AddAttributes(
    attribute.String("cache.key", key),
)
logger.Emit(ctx, record)
```

### Convert to Span Attributes

Before:
```go
span.AddEvent("request.details", trace.WithAttributes(
    attribute.String("http.request.body.summary", summary),
))
```

After:
```go
span.SetAttributes(
    attribute.String("http.request.body.summary", summary),
)
```

## Python

### Exception Recording

Before:
```python
span.record_exception(exc)
span.set_status(Status(StatusCode.ERROR, str(exc)))
```

After:
```python
from opentelemetry._logs import SeverityNumber, get_logger

logger = get_logger(__name__)
logger.emit(
    event_name=exception_event_name,
    severity_number=SeverityNumber.ERROR,
    body="exception",
    exception=exc,
)
span.set_status(Status(StatusCode.ERROR, str(exc)))
```

### General Event

Before:
```python
span.add_event("retry.attempt", attributes={"retry.count": count})
```

After:
```python
from opentelemetry._logs import get_logger

logger = get_logger(__name__)
logger.emit(
    event_name="retry.attempt",
    body="retry.attempt",
    attributes={"retry.count": count},
)
```

The Python SDK derives exception semantic-convention attributes when `exception=` is provided. If the project already uses stdlib `logging` with `LoggingHandler`, verify how that bridge maps event names before relying on it for `event_name`.

## Java

### Exception Recording

Before:
```java
span.recordException(exception);
span.setStatus(StatusCode.ERROR, exception.getMessage());
```

After:
```java
Logger logger = GlobalOpenTelemetry.get().getLogsBridge().loggerBuilder("my-class").build();
logger.logRecordBuilder()
    .setSeverity(Severity.ERROR)
    .setEventName(exceptionEventName)
    .setException(exception)
    .emit();
span.setStatus(StatusCode.ERROR, exception.getMessage());
```

`setEventName(...)` is a first-class method on the stable `LogRecordBuilder`
(since 1.50.0); `setException(Throwable)` is available since 1.60.0. Prefer
these over setting an `event.name` attribute or manually maintaining
`exception.*` attributes when the project is on a current Java SDK.

### General Event

Before:
```java
span.addEvent("state.transition", Attributes.of(
    AttributeKey.stringKey("state.from"), oldState,
    AttributeKey.stringKey("state.to"), newState
));
```

After:
```java
logger.logRecordBuilder()
    .setEventName("state.transition")
    .setAttribute(AttributeKey.stringKey("state.from"), oldState)
    .setAttribute(AttributeKey.stringKey("state.to"), newState)
    .emit();
```

## JavaScript / TypeScript

### Exception Recording

Before:
```typescript
span.recordException(error);
span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
```

After:
```typescript
const logger = logs.getLogger('my-module');
logger.emit({
  severityNumber: SeverityNumber.ERROR,
  eventName: exceptionEventName,
  body: 'exception',
  exception: error,
});
span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
```

The JS log-record `exception` field is present in current `@opentelemetry/api-logs` / `@opentelemetry/sdk-logs` source and marked experimental. If the project avoids experimental fields, set `exception.type`, `exception.message`, and `exception.stacktrace` attributes manually.

### General Event

Before:
```typescript
span.addEvent('queue.enqueue', { 'queue.name': queueName, 'queue.size': size });
```

After:
```typescript
logger.emit({
  eventName: 'queue.enqueue',
  attributes: {
    'queue.name': queueName,
    'queue.size': size,
  },
});
```

## .NET

Note: OpenTelemetry .NET's `Activity.RecordException(...)` extension is
`[Obsolete]` (it points callers to the runtime's `Activity.AddException(ex)`).
Both `RecordException` and `AddException` record an exception *span event*, so
either one is a migration source for this skill -- the target is still the
Logs API as shown below.

### Exception Recording

Before:
```csharp
activity?.RecordException(ex); // or the newer activity?.AddException(ex);
activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
```

After:
```csharp
var logger = loggerFactory.CreateLogger("MyClass");
logger.LogError(new EventId(0, exceptionEventName), ex, "{ExceptionEventName}", exceptionEventName);
activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
```

### General Event

Before:
```csharp
activity?.AddEvent(new ActivityEvent("validation.failure", default, new ActivityTagsCollection
{
    { "validation.field", fieldName },
    { "validation.rule", rule },
}));
```

After:
```csharp
logger.LogWarning(
    new EventId(0, "validation.failure"),
    "validation.failure for field {validation.field} rule {validation.rule}",
    fieldName,
    rule);
```

With the OpenTelemetry .NET OTLP exporter, `EventId.Name` is exported as the OTLP log `event_name` field. The lower-level `LogRecordData.EventName` / `Logger.EmitLog` bridge API exists in source but is experimental/pre-release-gated, so prefer stable `ILogger` patterns unless the project has opted into the bridge API.

## Ruby

### Exception Recording

Before:
```ruby
span.record_exception(exception)
span.status = OpenTelemetry::Trace::Status.error(exception.message)
```

After:
```ruby
logger = OpenTelemetry.logger_provider.logger(name: 'my-gem')
logger.on_emit(
  event_name: exception_event_name,
  severity_number: OpenTelemetry::Logs::SeverityNumber::SEVERITY_NUMBER_ERROR,
  body: 'exception',
  attributes: {
    'exception.type' => exception.class.name,
    'exception.message' => exception.message,
    'exception.stacktrace' => exception.full_message
  }
)
span.status = OpenTelemetry::Trace::Status.error(exception.message)
```

### General Event

Before:
```ruby
span.add_event('retry.attempt', attributes: { 'retry.count' => count })
```

After:
```ruby
logger.on_emit(
  event_name: 'retry.attempt',
  body: 'retry.attempt',
  attributes: { 'retry.count' => count }
)
```

Ruby Logs API 0.4.1 and Logs SDK 0.6.1 expose `Logger#on_emit` with
`event_name:` and use the current context by default. They do not expose an
exception convenience parameter, so populate the applicable `exception.*`
attributes explicitly.

## Key Rules Across All Languages

1. The event name replaces the name from `AddEvent`. Use the dedicated event-name API where available -- `record.SetEventName(...)` (Go), `.setEventName(...)` (Java `LogRecordBuilder`), `eventName:` (JS `logger.emit`), `event_name=` (Python `Logger.emit`), `EventId.Name` (.NET `ILogger`), or `event_name:` (Ruby `Logger#on_emit`) -- which maps to the `event_name` LogRecord field. Only set an `event.name` attribute when the target SDK/exporter has no dedicated event-name path.
2. All original attributes transfer to the log record attributes, preserving the
   original attribute keys unless an intentional mapping is documented and tested.
3. The log record automatically inherits the active span context from `ctx` / the current context -- this is how trace correlation is maintained.
4. `span.SetStatus` (or equivalent) is still set on the span for error cases -- the migration only moves event emission, not status.
5. Timestamps are set automatically by the SDK if not specified explicitly.
