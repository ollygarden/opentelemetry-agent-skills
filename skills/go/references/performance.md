# OpenTelemetry Go Performance Tuning

Performance tuning reference for OpenTelemetry Go SDK. Covers sampling, batch processing, metric readers, attribute allocation, exporter configuration, and pipeline reliability.

---

## Performance Impact by Signal

| Signal | Unsampled Overhead | Sampled Overhead | Primary Cost |
|--------|-------------------|------------------|--------------|
| Traces | Near-zero (noop span) | Moderate | Allocations, export I/O |
| Metrics | N/A (always collected) | N/A | Aggregation, cardinality |
| Logs | Near-zero if `Enabled()` false | Low-moderate | Serialization, export I/O |

## Default Configuration Values

| Parameter | Default | Environment Variable |
|-----------|---------|---------------------|
| Batch span queue size | 2048 | -- |
| Batch span export size | 512 | `OTEL_TRACES_EXPORTER_MAX_EXPORT_BATCH_SIZE` |
| Batch schedule delay | 5s | `OTEL_TRACES_EXPORTER_BATCH_SCHEDULE_DELAY` |
| Batch export timeout | 30s | `OTEL_TRACES_EXPORTER_TIMEOUT` |
| Span attribute limit | 128 | `OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT` |
| Span event limit | 128 | `OTEL_SPAN_EVENT_COUNT_LIMIT` |
| Span link limit | 128 | `OTEL_SPAN_LINK_COUNT_LIMIT` |
| Attribute value length limit | Unlimited | `OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT` |
| Metric export interval | 60s | `OTEL_METRIC_EXPORT_INTERVAL` |
| Metric export timeout | 30s | `OTEL_METRIC_EXPORT_TIMEOUT` |
| OTLP export timeout | 10s | `OTEL_EXPORTER_OTLP_TIMEOUT` |

---

## Sampling

Sampling is the single most impactful performance lever for traces. Unsampled spans create noop instances with virtually zero overhead -- no allocations for attributes, events, or links.

### Head Sampling Configuration

Configure a sampler at the `TracerProvider` level to decide before span processing whether to record:

```go
// Sample 10% of traces
sampler := sdktrace.TraceIDRatioBased(0.1)

// Composite: respect upstream sampling decisions, ratio-sample the rest
sampler := sdktrace.ParentBased(
    sdktrace.TraceIDRatioBased(0.1),
    // Remote parent sampled -> always sample (honor upstream)
    // Remote parent not sampled -> never sample
)

tp := sdktrace.NewTracerProvider(
    sdktrace.WithSampler(sampler),
    sdktrace.WithBatcher(exporter),
)
```

### AlwaysRecord Sampler (v1.40.0+)

The `AlwaysRecord` sampler wraps another sampler and ensures spans are always recorded (attributes, events, status are captured) even when the inner sampler decides not to sample (export). This is useful with tail sampling in a Collector: spans carry full data for the Collector to make sampling decisions, but unsampled spans are not exported by the SDK.

```go
// Record all spans but only export 10% -- useful with Collector tail sampling
sampler := sdktrace.AlwaysRecord(sdktrace.TraceIDRatioBased(0.1))
```

### Sampling Decision Impact

```
AlwaysSample       -> Full span lifecycle: allocation + recording + export
AlwaysRecord(0.1)  -> All spans recorded, 10% exported
TraceIDRatioBased(0.1) -> 90% of spans become noop (near-zero cost)
NeverSample        -> All spans noop (useful for load testing)
```

`AlwaysSample` records and exports every span. `TraceIDRatioBased(0.1)` makes 90% of spans noop with near-zero cost. `ParentBased` wrapping respects upstream sampling decisions. `NeverSample` makes all spans noop, which is useful for benchmarking application-only performance.

> **Tail sampling**: For decision-making based on complete traces (error status, latency thresholds), use the Collector's tail sampling processor instead of SDK-level sampling. SDK head sampling combined with Collector tail sampling is a common production pattern.

---

## Batch Processor Tuning

The `BatchSpanProcessor` buffers spans in a queue and exports them asynchronously in batches.

### How It Works

```
Application goroutine          BatchSpanProcessor goroutine
       |                                |
   span.End() --enqueue-->  channel queue (MaxQueueSize)
       |                                |
       |                         timer fires (ScheduleDelay)
       |                         OR batch full (MaxExportBatchSize)
       |                                |
       |                        --export batch--> Exporter
```

### Tuning for Throughput

For high-volume services (>10k spans/second):

```go
bsp := sdktrace.NewBatchSpanProcessor(exporter,
    sdktrace.WithMaxQueueSize(4096),       // Absorb bursts (default: 2048)
    sdktrace.WithMaxExportBatchSize(1024), // Fewer network calls (default: 512)
    sdktrace.WithBatchTimeout(10*time.Second), // Allow larger batches (default: 5s)
)
```

### Tuning for Latency

For services where trace delivery speed matters (debugging, alerting):

```go
bsp := sdktrace.NewBatchSpanProcessor(exporter,
    sdktrace.WithMaxExportBatchSize(128),       // Export smaller batches faster
    sdktrace.WithBatchTimeout(2*time.Second),   // Export more frequently
)
```

### Queue Full Behavior

When the queue fills, new spans are dropped silently by default. Telemetry loss is preferable to application slowdown in this design.

`WithBlockOnQueueFull()` changes this behavior to backpressure the application goroutine, blocking on `span.End()` until queue space is available.

The SDK exposes internal metrics for monitoring queue depth.

### SimpleSpanProcessor

`SimpleSpanProcessor` exports spans synchronously on `span.End()`. It adds latency to every span ending. It is deterministic, which makes it useful for tests and development. Short-lived CLI tools may also use it to ensure spans are exported before exit.

```go
ssp := sdktrace.NewSimpleSpanProcessor(exporter)
```

---

## Metric Reader Tuning

### PeriodicReader Configuration

The `PeriodicReader` collects and exports metrics on a fixed interval.

```go
reader := sdkmetric.NewPeriodicReader(exporter,
    sdkmetric.WithInterval(30*time.Second), // More frequent than default 60s
    sdkmetric.WithTimeout(15*time.Second),  // Timeout per export
)
```

Interval tradeoffs:
- 60s (default): Standard for dashboards and alerting
- 15-30s: Near-real-time monitoring, higher export overhead
- 5-10s: High-frequency use cases, significant overhead

The reader internally uses `sync.Pool` to recycle `ResourceMetrics` objects, reducing GC pressure during collection cycles.

### Views for Cardinality Control

Views filter attributes before aggregation, reducing the number of unique time series.

#### Attribute Filter

```go
// Drop a high-cardinality attribute from a specific metric
dropUserID := sdkmetric.NewView(
    sdkmetric.Instrument{Name: "http.server.request.duration"},
    sdkmetric.Stream{
        AttributeFilter: func(kv attribute.KeyValue) bool {
            return kv.Key != "user.id" // Exclude user.id from aggregation
        },
    },
)
```

An allowlist-style filter keeps only specified attributes:

```go
sdkmetric.Stream{
    AttributeFilter: func(kv attribute.KeyValue) bool {
        return kv.Key == "http.request.method" || kv.Key == "http.response.status_code"
    },
}
```

#### Drop Metric

```go
// Drop an entire metric
dropMetric := sdkmetric.NewView(
    sdkmetric.Instrument{Name: "runtime.*"},
    sdkmetric.Stream{Aggregation: sdkmetric.AggregationDrop{}},
)
```

#### Applying Views

```go
mp := sdkmetric.NewMeterProvider(
    sdkmetric.WithReader(reader),
    sdkmetric.WithView(dropUserID, dropMetric),
)
```

Metric attribute cardinality is the product of unique values across all recorded attributes. Attributes like user IDs or request IDs are unbounded and produce one time series per unique value. Attributes like HTTP method (~10 values) or status code (~50 values) are bounded.

---

## Attribute Allocation Patterns

Attributes are one of the primary sources of allocations in instrumented code.

### Pre-allocating Attribute Sets

For attributes used across many measurements, construct them once:

```go
var (
    methodGET  = attribute.String("http.request.method", "GET")
    methodPOST = attribute.String("http.request.method", "POST")
    status2xx  = attribute.Int("http.response.status_code", 200)
    status4xx  = attribute.Int("http.response.status_code", 400)
    status5xx  = attribute.Int("http.response.status_code", 500)
)

func handleRequest(ctx context.Context, method string, status int) {
    // Reuse pre-allocated attributes -- no allocation per call
    counter.Add(ctx, 1, metric.WithAttributes(methodGET, status2xx))
}
```

Constructing `attribute.String(...)` or `attribute.Int(...)` inline on every call allocates on every call.

### Slice Capacity

When building dynamic attribute sets, pre-allocating slice capacity eliminates repeated reallocations:

```go
attrs := make([]attribute.KeyValue, 0, 4)
attrs = append(attrs, attribute.String("service.name", svcName))
attrs = append(attrs, attribute.String("deployment.environment.name", env))
if region != "" {
    attrs = append(attrs, attribute.String("cloud.region", region))
}
```

A `nil` slice (`var attrs []attribute.KeyValue`) reallocates on the first and subsequent appends as it grows.

### Span Attribute Limits

The SDK enforces configurable limits on span attributes, events, and links. When limits are exceeded, the oldest items are evicted (FIFO):

```go
tp := sdktrace.NewTracerProvider(
    sdktrace.WithSpanLimits(sdktrace.SpanLimits{
        AttributeCountLimit:         64,  // Default: 128
        EventCountLimit:             32,  // Default: 128
        LinkCountLimit:              16,  // Default: 128
        AttributePerEventCountLimit: 16,  // Default: 128
        AttributePerLinkCountLimit:  16,  // Default: 128
        AttributeValueLengthLimit:   256, // Default: Unlimited. Truncates long strings.
    }),
)
```

`AttributeValueLengthLimit` truncates string values (stack traces, SQL queries, request bodies) that would otherwise consume unbounded memory.

---

## Exporter Configuration

### gRPC vs HTTP

| Aspect | gRPC | HTTP/protobuf |
|--------|------|---------------|
| Connection overhead | Single long-lived connection | New connection per export (or HTTP/2 reuse) |
| Default port | 4317 | 4318 |
| Compression | Optional gzip | Optional gzip |
| Best for | Persistent connections, high throughput | Firewalls, load balancers, simpler infra |

### Compression

Compression reduces bandwidth at the cost of CPU.

gRPC:

```go
import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

exporter, err := otlptracegrpc.New(ctx,
    otlptracegrpc.WithCompressor("gzip"),
)
```

HTTP:

```go
import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

exporter, err := otlptracehttp.New(ctx,
    otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
)
```

### Retry Defaults

The OTLP exporters use exponential backoff with jitter:

| Parameter | Default |
|-----------|---------|
| Enabled | true |
| Initial interval | 5s |
| Max interval | 30s |
| Max elapsed time | 1min |

Retries honor `Retry-After` headers from the backend.

### Timeout Tuning

```go
// Lower timeout: fail fast
exporter, err := otlptracegrpc.New(ctx,
    otlptracegrpc.WithTimeout(5*time.Second), // Default: 10s
)

// Higher timeout: allow more time for large batches
exporter, err := otlptracegrpc.New(ctx,
    otlptracegrpc.WithTimeout(30*time.Second),
)
```

---

## Log Signal Performance

### Enabled() Early Exit

The Logs API supports an `Enabled()` check for skipping expensive log construction when the log level is below threshold or when a processor filters it:

```go
logger := otellog.GetLoggerProvider().Logger("my-service")

if logger.Enabled(ctx, log.EnabledParameters{Severity: log.SeverityInfo}) {
    var rec log.Record
    rec.SetBody(log.StringValue(expensiveStringBuild()))
    rec.SetSeverity(log.SeverityInfo)
    logger.Emit(ctx, rec)
}
```

### log.Record Value Type

`log.Record` is a struct value type (not an interface). Passing it by value keeps it stack-allocated, avoiding heap allocation:

```go
var rec log.Record
rec.SetBody(log.StringValue("operation completed"))
rec.SetSeverity(log.SeverityInfo)
rec.AddAttributes(log.String("component", "handler"))
logger.Emit(ctx, rec)
```

---

## Context Propagation Performance

### Extract Once per Boundary

W3C Trace Context propagation involves parsing and formatting fixed-format headers. The overhead is minimal. Extract once per request boundary:

```go
ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
```

Extracting multiple times for the same request performs redundant parsing with no benefit.

### Baggage Overhead

Baggage is propagated across all service boundaries via HTTP headers. Every key-value pair in baggage adds to every outgoing request's header size:

```go
bag, _ := baggage.New(
    baggageMember("tenant.id", tenantID),
)
ctx = baggage.ContextWithBaggage(ctx, bag)
```

Large values (e.g., full request bodies) or PII in baggage are propagated to every downstream service in every request.

---

## Graceful Shutdown

Proper shutdown ensures buffered telemetry is flushed before the process exits. Shutdown order: providers first (which flush their processors), then exporters.

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := tp.Shutdown(ctx); err != nil {
    slog.Error("trace provider shutdown failed", "error", err)
}
if err := mp.Shutdown(ctx); err != nil {
    slog.Error("metric provider shutdown failed", "error", err)
}
if err := lp.Shutdown(ctx); err != nil {
    slog.Error("log provider shutdown failed", "error", err)
}
```

---

## Telemetry Pipeline Reliability

The OpenTelemetry SDK is designed so that telemetry failures do not crash or block the application:

- **Span creation never returns an error** -- it returns a noop span on failure
- **Metric recording never returns an error** -- measurements are silently dropped on failure
- **Export failures are retried** -- then dropped after max elapsed time
- **Queue overflow drops spans** -- the application is not blocked (unless `WithBlockOnQueueFull` is used)

---

## Monitoring the Pipeline

The SDK exposes internal metrics about its own health:

| Metric | Meaning |
|--------|---------|
| `otel.sdk.trace.spans_exported` | Spans successfully exported |
| `otel.sdk.trace.spans_dropped` | Spans dropped (queue full, export failure) |
| Exporter errors in logs | Export failures with error details |

### ForceFlush

`ForceFlush` synchronously exports all buffered data:

```go
if err := tp.ForceFlush(ctx); err != nil {
    slog.Warn("failed to flush traces", "error", err)
}
```

As of v1.42.0, `TracerProvider.ForceFlush` iterates through all `SpanProcessor` instances and joins their errors together, instead of stopping at the first error. All processors get a chance to flush even if one fails.

`ForceFlush` in the request hot path synchronously exports all buffered data, adding latency to that request.

---

## Synchronous Instrument Enabled()

As of v1.40.0, all synchronous metric instruments (`Int64Counter`, `Float64Counter`, `Int64UpDownCounter`, `Float64UpDownCounter`, `Int64Histogram`, `Float64Histogram`, `Int64Gauge`, `Float64Gauge`) expose an `Enabled()` method. This allows skipping expensive metric value computation when the instrument is not recording:

```go
histogram, _ := meter.Float64Histogram("expensive.metric")

attrs := metric.WithAttributes(attribute.String("region", region))
if histogram.Enabled(ctx, attrs) {
    value := computeExpensiveValue() // Only called when instrument is active
    histogram.Record(ctx, value, attrs)
}
```

This is the metric equivalent of `Logger.Enabled()` for logs. When combined with Views that drop metrics, `Enabled()` returns `false` for dropped instruments.