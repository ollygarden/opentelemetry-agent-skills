# OpenTelemetry Go API

**Module**: `go.opentelemetry.io/otel`

## Import Paths
```go
import (
    "go.opentelemetry.io/otel"                    // Global API
    "go.opentelemetry.io/otel/trace"              // Tracing API
    "go.opentelemetry.io/otel/metric"             // Metrics API
    "go.opentelemetry.io/otel/log"                // Logs API (stable as of v1.34.0)
    "go.opentelemetry.io/otel/log/global"         // Global LoggerProvider
    "go.opentelemetry.io/otel/attribute"          // Attributes
    "go.opentelemetry.io/otel/codes"              // Status codes
    "go.opentelemetry.io/otel/propagation"        // Context propagation
    "go.opentelemetry.io/otel/baggage"            // Baggage API
)
```

## Global API Access

### Provider Registration
```go
// Set global providers (typically in main())
otel.SetTracerProvider(tracerProvider)
otel.SetMeterProvider(meterProvider)
otel.SetTextMapPropagator(propagator)

// Get providers from global
tp := otel.GetTracerProvider()
mp := otel.GetMeterProvider()
```

### Convenience Accessors
```go
// Get a tracer from the global TracerProvider
tracer := otel.Tracer("github.com/user/pkg")

// Get a meter from the global MeterProvider
meter := otel.Meter("github.com/user/pkg")
```

### Error Handling
```go
// Set global error handler
otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
    // Log or handle errors
}))
```

## Tracing API

### Core Types
```go
trace.TracerProvider    // Creates Tracers
trace.Tracer           // Creates Spans
trace.Span             // Represents a unit of work
trace.SpanContext      // Immutable span identifier
trace.TraceID          // [16]byte unique trace identifier
trace.SpanID           // [8]byte unique span identifier
```

### Creating Tracers and Spans
```go
// Get tracer (use import path as name)
tracer := otel.Tracer("github.com/user/pkg")

// Start span with automatic context propagation
ctx, span := tracer.Start(ctx, "verb object",
    trace.WithAttributes(
        attribute.String("key", "value"),
    ),
    trace.WithSpanKind(trace.SpanKindServer),
)
defer span.End()

// Add attributes during execution
span.SetAttributes(attribute.Int64("items.count", 42))

// Set status on error
if err != nil {
    span.SetStatus(codes.Error, err.Error())
}
```

### Span Options
```go
trace.WithAttributes(attrs...)      // Initial attributes
trace.WithSpanKind(kind)            // Server, Client, Producer, Consumer, Internal
trace.WithNewRoot()                 // Start new trace
trace.WithLinks(links...)           // Link to other spans
```

### Context Utilities
```go
// Extract span from context
span := trace.SpanFromContext(ctx)

// Get current span context
sc := span.SpanContext()
traceID := sc.TraceID()
spanID := sc.SpanID()
```

## Metrics API

### Core Types
```go
metric.MeterProvider    // Creates Meters
metric.Meter           // Creates Instruments
// Synchronous instruments
metric.Int64Counter
metric.Float64Counter
metric.Int64UpDownCounter
metric.Float64UpDownCounter
metric.Int64Histogram
metric.Float64Histogram
metric.Int64Gauge
metric.Float64Gauge
// Asynchronous instruments (callbacks)
metric.Int64ObservableCounter
metric.Float64ObservableCounter
metric.Int64ObservableUpDownCounter
metric.Float64ObservableUpDownCounter
metric.Int64ObservableGauge
metric.Float64ObservableGauge
```

### Creating Meters and Instruments
```go
// Get meter (use import path as name)
meter := otel.Meter("github.com/user/pkg")

// Counter - monotonic sum
counter, _ := meter.Int64Counter("requests.total",
    metric.WithDescription("Total requests"),
    metric.WithUnit("1"),
)
counter.Add(ctx, 1,
    metric.WithAttributes(attribute.String("method", "GET")))

// UpDownCounter - non-monotonic sum
updown, _ := meter.Int64UpDownCounter("queue.size")
updown.Add(ctx, -1)

// Histogram - distribution of values
histogram, _ := meter.Float64Histogram("request.duration",
    metric.WithUnit("ms"),
)
histogram.Record(ctx, 123.45)

// Gauge - current value
gauge, _ := meter.Float64Gauge("cpu.usage",
    metric.WithUnit("%"),
)
gauge.Record(ctx, 75.5)

// Observable (async) gauge with callback
_, _ = meter.Int64ObservableGauge("memory.usage",
    metric.WithCallback(func(ctx context.Context, o metric.Int64Observer) error {
        o.Observe(getCurrentMemory())
        return nil
    }),
)

// Check if an instrument is enabled before expensive operations (v1.40.0+)
// Available on all synchronous instruments: Counter, UpDownCounter, Histogram, Gauge
if counter.Enabled(ctx, metric.WithAttributes(attribute.String("method", "GET"))) {
    // Only compute expensive value if the instrument will record
    value := computeExpensiveMetric()
    counter.Add(ctx, value, metric.WithAttributes(attribute.String("method", "GET")))
}
```

## Attributes

### Creating Attributes
```go
// Typed attribute constructors
attribute.String("key", "value")
attribute.Int64("count", 42)
attribute.Float64("ratio", 0.95)
attribute.Bool("enabled", true)
attribute.StringSlice("tags", []string{"a", "b"})
attribute.Int64Slice("ids", []int64{1, 2, 3})
attribute.Float64Slice("values", []float64{1.1, 2.2})
attribute.BoolSlice("flags", []bool{true, false})
```

### Attribute Sets
```go
// Create reusable attribute set
attrs := attribute.NewSet(
    attribute.String("service.name", "api"),
    attribute.String("service.version", "1.0.0"),
)

// Filter attributes
filtered := attrs.Filter(func(kv attribute.KeyValue) bool {
    return kv.Key != "service.version"
})
```

## Propagation

### Context Propagation
```go
// Create composite propagator
propagator := propagation.NewCompositeTextMapPropagator(
    propagation.TraceContext{},  // W3C Trace Context
)

// Set global propagator
otel.SetTextMapPropagator(propagator)

// Extract from HTTP headers
ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))

// Inject into HTTP headers
propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))
```

## Logs API

The Logs API became stable in v1.34.0. It provides a bridge for existing logging libraries.

### Core Types
```go
log.LoggerProvider     // Creates Loggers
log.Logger             // Emits log records
log.Record             // Log record with attributes
log.Value              // Typed attribute value
log.Severity           // Log severity level
```

### Creating Loggers and Emitting Records
```go
// Get logger (use import path as name)
logger := global.GetLoggerProvider().Logger("github.com/user/pkg")

// Check if logging is enabled
var params log.EnabledParameters
params.SetSeverity(log.SeverityInfo)
if logger.Enabled(ctx, params) {
    var rec log.Record
    rec.SetTimestamp(time.Now())
    rec.SetSeverity(log.SeverityInfo)
    rec.SetBody(log.StringValue("User logged in"))
    rec.AddAttributes(
        log.String("user.id", userID),
        log.String("session.id", sessionID),
    )
    logger.Emit(ctx, rec)
}

// Attach an error to a log record (v1.42.0+)
// The SDK automatically sets exception attributes (exception.type, exception.message)
var rec log.Record
rec.SetSeverity(log.SeverityError)
rec.SetBody(log.StringValue("operation failed"))
rec.SetErr(err) // Attaches error; SDK sets exception attributes automatically
logger.Emit(ctx, rec)

// Read back the error from a record
if recordErr := rec.Err(); recordErr != nil {
    // Handle the error
}
```

### Severity Levels
```go
log.SeverityTrace1 through log.SeverityTrace4
log.SeverityDebug1 through log.SeverityDebug4
log.SeverityInfo1 through log.SeverityInfo4
log.SeverityWarn1 through log.SeverityWarn4
log.SeverityError1 through log.SeverityError4
log.SeverityFatal1 through log.SeverityFatal4
```

### Log Values
```go
log.StringValue("text")
log.IntValue(42)
log.Int64Value(int64(42))
log.Float64Value(3.14)
log.BoolValue(true)
log.BytesValue([]byte("data"))
log.SliceValue(log.StringValue("a"), log.StringValue("b"))
log.MapValue(log.String("key", "value"))
```

## Logging Bridges

Logging bridges connect existing logging libraries to OpenTelemetry.

> **Note (contrib v1.35.0):** otelzap and otelslog emit `code.function` with the full package path-qualified function name (e.g., `github.com/user/pkg.MyFunc`). The `code.namespace` attribute is no longer emitted.

```go
import (
    // For zap logging
    "go.opentelemetry.io/contrib/bridges/otelzap"

    // For slog logging (Go 1.21+)
    "go.opentelemetry.io/contrib/bridges/otelslog"

    // For logrus logging
    "go.opentelemetry.io/contrib/bridges/otellogrus"

    // For zerolog logging
    "go.opentelemetry.io/contrib/bridges/otelzerolog"

    // For logr logging
    "go.opentelemetry.io/contrib/bridges/otellogr"
)
```

### Example: zap bridge
```go
// Create zap logger with OpenTelemetry bridge
core := zapcore.NewTee(
    zapcore.NewCore(encoder, os.Stdout, zapcore.InfoLevel),
    otelzap.NewCore("my-service", otelzap.WithLoggerProvider(loggerProvider)),
)
logger := zap.New(core)
```

### Example: slog bridge
```go
// Create slog handler with OpenTelemetry bridge
handler := otelslog.NewHandler("my-service", otelslog.WithLoggerProvider(loggerProvider))
logger := slog.New(handler)
```
