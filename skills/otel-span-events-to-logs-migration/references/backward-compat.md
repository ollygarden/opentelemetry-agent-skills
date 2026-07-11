# Backward Compatibility: The SDK Bridge

When downstream systems (backends, dashboards, alerting rules) depend on span events appearing in the same proto envelope as the span, the SDK bridge preserves this behavior after migration.

## What It Is

An SDK-based log processor that:
1. Intercepts log records that represent events (identified by a non-empty `event_name` LogRecord field)
2. Converts them to span events on the current span, when the record's `TraceId`/`SpanId` match the current recording span
3. Copies the record's timestamp and attributes onto the span event

This means the log-based event appears as a traditional span event in the exported span data, while also being available as a log record if a log exporter is configured. Bridging does not remove the record from the normal log pipeline.

The bridge is now specified in the OpenTelemetry Specification as the "Event to span event bridge" [LogRecordProcessor](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/sdk.md#event-to-span-event-bridge) (Status: Development), which defines these exact bridging conditions.

## When to Use It

Use the bridge when:
- a backend requires span events in the span proto (e.g., for timeline visualization)
- alerting rules query span events directly
- dashboards depend on span event fields
- migrating incrementally and some consumers have not adapted yet

Do NOT use the bridge when:
- the backend already supports log-based events correlated to traces
- you are starting fresh with no legacy consumers
- you want a clean break from span events

## How to Configure It

### Via Declarative Configuration (preferred)

When available for the language SDK, add the bridge processor to the log pipeline in the declarative config. The processor key is defined in the [opentelemetry-configuration](https://github.com/open-telemetry/opentelemetry-configuration/blob/main/schema/logger_provider.yaml) schema as `event_to_span_event_bridge/development` (the `/development` suffix marks it experimental):

```yaml
# OpenTelemetry SDK declarative configuration
logger_provider:
  processors:
    - event_to_span_event_bridge/development: {}
    - batch:
        exporter:
          otlp_http:
            endpoint: "http://collector:4318"
```

### Via Code (when declarative config is not available)

The bridge is a log record processor. Add it to the LoggerProvider alongside the batch processor:

```go
// Go example
bridgeProcessor := eventbridge.NewSpanEventBridge()
loggerProvider := log.NewLoggerProvider(
    log.WithProcessor(bridgeProcessor),
    log.WithProcessor(
        log.NewBatchProcessor(otlpExporter),
    ),
)
```

```java
// Java example -- see opentelemetry-java-contrib
// processors module for EventToSpanEventBridge
SdkLoggerProvider loggerProvider = SdkLoggerProvider.builder()
    .addLogRecordProcessor(EventToSpanEventBridge.create())
    .addLogRecordProcessor(BatchLogRecordProcessor.builder(otlpExporter).build())
    .build();
```

## Processor Ordering

The bridge processor should be registered BEFORE the batch/export processor in the chain. This ensures:
1. The log record is first converted to a span event and attached to the span
2. Then the log record is batched and exported as a log (if desired)

If you only want span events and do NOT want separate log export, omit the batch log exporter and only register the bridge.

## Migration Path

1. Start with the bridge enabled
2. Update downstream consumers to read from logs instead of span events
3. Once all consumers are migrated, remove the bridge
4. The bridge is a transitional tool, not a permanent fixture

## Reference Implementation

- Java: [opentelemetry-java-contrib processors](https://github.com/open-telemetry/opentelemetry-java-contrib/blob/main/processors/README.md#event-to-spanevent-bridge)
