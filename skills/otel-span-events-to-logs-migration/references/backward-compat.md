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

### Via Declarative Configuration

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

### Via Code

The bridge is a log record processor. Add a bridge implementation provided by
the target language SDK to the LoggerProvider alongside any export processors.
Do not infer support from the specification alone: the specification says SDKs
SHOULD provide the processor, and implementation availability varies by language.

For example, OpenTelemetry Java 1.64.0 provides the bridge in the incubator SDK
extension:

```java
// Java example -- opentelemetry-sdk-extension-incubator
SdkLoggerProvider loggerProvider = SdkLoggerProvider.builder()
    .addLogRecordProcessor(EventToSpanEventBridge.create())
    .addLogRecordProcessor(BatchLogRecordProcessor.builder(otlpExporter).build())
    .build();
```

The specification does not require a particular ordering between the bridge and
other processors. Follow the target SDK's processor-composition semantics. If
you only want span events and do not want separate log export, configure only
the bridge where the SDK permits that pipeline.

## Migration Path

1. Start with the bridge enabled
2. Update downstream consumers to read from logs instead of span events
3. Once all consumers are migrated, remove the bridge
4. The bridge is a transitional tool, not a permanent fixture

## Reference Implementations

- Specification: [Event to span event bridge](https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/logs/sdk.md#event-to-span-event-bridge)
- Java: `io.opentelemetry.sdk.extension.incubator.logs.EventToSpanEventBridge` in `opentelemetry-sdk-extension-incubator`
- Historical Java contrib bridge: [opentelemetry-java-contrib processors](https://github.com/open-telemetry/opentelemetry-java-contrib/blob/main/processors/README.md#event-to-spanevent-bridge) is deprecated and points to the SDK incubator extension.
