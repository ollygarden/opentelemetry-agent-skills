# Migration Patterns: Before and After

Language-specific patterns for migrating from `AddEvent` / `RecordException` to the Logs API.

When applying these patterns, always check the project's actual SDK imports and version. The examples below use idiomatic API calls; adapt to the project's style.

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
logger := otel.GetLoggerProvider().Logger("my-package")
record := log.Record{}
record.SetTimestamp(time.Now())
record.SetSeverity(log.SeverityError)
record.AddAttributes(
    log.String("event.name", "exception"),
    log.String("exception.type", fmt.Sprintf("%T", err)),
    log.String("exception.message", err.Error()),
    log.String("exception.stacktrace", captureStack()),
)
logger.Emit(ctx, record)
span.SetStatus(codes.Error, err.Error())
```

### General Event

Before:
```go
span.AddEvent("cache.miss", trace.WithAttributes(
    attribute.String("cache.key", key),
))
```

After:
```go
logger := otel.GetLoggerProvider().Logger("my-package")
record := log.Record{}
record.SetTimestamp(time.Now())
record.AddAttributes(
    log.String("event.name", "cache.miss"),
    log.String("cache.key", key),
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
import logging

logger = logging.getLogger(__name__)
# With the OTel logging bridge configured, this emits a log-based event
logger.exception("exception", exc_info=exc, extra={
    "event.name": "exception",
})
span.set_status(Status(StatusCode.ERROR, str(exc)))
```

### General Event

Before:
```python
span.add_event("retry.attempt", attributes={"retry.count": count})
```

After:
```python
logger.info("retry.attempt", extra={
    "event.name": "retry.attempt",
    "retry.count": count,
})
```

## Java

### Exception Recording

Before:
```java
span.recordException(exception);
span.setStatus(StatusCode.ERROR, exception.getMessage());
```

After:
```java
Logger logger = GlobalOpenTelemetry.getLogsBridge().loggerBuilder("my-class").build();
logger.logRecordBuilder()
    .setSeverity(Severity.ERROR)
    .setAttribute(AttributeKey.stringKey("event.name"), "exception")
    .setAttribute(SemanticAttributes.EXCEPTION_TYPE, exception.getClass().getName())
    .setAttribute(SemanticAttributes.EXCEPTION_MESSAGE, exception.getMessage())
    .setAttribute(SemanticAttributes.EXCEPTION_STACKTRACE, getStackTrace(exception))
    .emit();
span.setStatus(StatusCode.ERROR, exception.getMessage());
```

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
    .setAttribute(AttributeKey.stringKey("event.name"), "state.transition")
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
  attributes: {
    'event.name': 'exception',
    'exception.type': error.name,
    'exception.message': error.message,
    'exception.stacktrace': error.stack,
  },
});
span.setStatus({ code: SpanStatusCode.ERROR, message: error.message });
```

### General Event

Before:
```typescript
span.addEvent('queue.enqueue', { 'queue.name': queueName, 'queue.size': size });
```

After:
```typescript
logger.emit({
  attributes: {
    'event.name': 'queue.enqueue',
    'queue.name': queueName,
    'queue.size': size,
  },
});
```

## .NET

### Exception Recording

Before:
```csharp
activity?.RecordException(ex);
activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
```

After:
```csharp
var logger = loggerFactory.CreateLogger("MyClass");
logger.LogError(ex, "exception");
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
logger.LogWarning("validation.failure for field {Field} rule {Rule}", fieldName, rule);
```

## Key Rules Across All Languages

1. The `event.name` attribute on the log record replaces the event name from `AddEvent`.
2. All original attributes transfer to the log record attributes.
3. The log record automatically inherits the active span context from `ctx` / the current context -- this is how trace correlation is maintained.
4. `span.SetStatus` (or equivalent) is still set on the span for error cases -- the migration only moves event emission, not status.
5. Timestamps are set automatically by the SDK if not specified explicitly.
