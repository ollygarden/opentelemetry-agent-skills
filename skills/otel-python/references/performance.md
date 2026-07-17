# OpenTelemetry Python Performance Tuning

Performance tuning reference for the OpenTelemetry Python SDK. Covers sampling, batch processing, metric readers, cardinality control via Views, asyncio context propagation, exporter configuration, log-handler cost, and graceful shutdown.

---

## Performance Impact by Signal

| Signal | Unsampled Overhead | Sampled Overhead | Primary Cost |
|--------|-------------------|------------------|--------------|
| Traces | Near-zero (noop span) | Moderate | Object creation, export I/O |
| Metrics | N/A (always collected) | N/A | Aggregation, cardinality |
| Logs | Low if handler level filtered | Low-moderate | Serialization, export I/O |

---

## Default Configuration Values

The SDK reads defaults from environment variables; check the [OpenTelemetry Python SDK changelog](https://github.com/open-telemetry/opentelemetry-python/blob/main/CHANGELOG.md) or env-var spec for current values — do not treat any number here as authoritative.

| Parameter | Environment Variable | Note |
|-----------|---------------------|------|
| BSP max queue size | `OTEL_BSP_MAX_QUEUE_SIZE` | Check SDK default |
| BSP max export batch size | `OTEL_BSP_MAX_EXPORT_BATCH_SIZE` | Check SDK default |
| BSP schedule delay (ms) | `OTEL_BSP_SCHEDULE_DELAY` | Check SDK default |
| BSP export timeout (ms) | `OTEL_BSP_EXPORT_TIMEOUT` | Accepted but not applied by `BatchSpanProcessor` in SDK 1.43.0 |
| Span attribute count limit | `OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT` | Check SDK default |
| Span event count limit | `OTEL_SPAN_EVENT_COUNT_LIMIT` | Check SDK default |
| Span link count limit | `OTEL_SPAN_LINK_COUNT_LIMIT` | Check SDK default |
| Attribute value length limit | `OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT` | Unlimited if unset |
| Metric export interval (ms) | `OTEL_METRIC_EXPORT_INTERVAL` | Check SDK default |
| Metric export timeout (ms) | `OTEL_METRIC_EXPORT_TIMEOUT` | Check SDK default |
| OTLP export timeout (s) | `OTEL_EXPORTER_OTLP_TIMEOUT` | Python reads this as **seconds** (default `10`), deviating from the spec, which defines it in milliseconds |
| Traces sampler | `OTEL_TRACES_SAMPLER` | `parentbased_always_on` if unset |
| Traces sampler arg | `OTEL_TRACES_SAMPLER_ARG` | Ratio for ratio-based samplers |

---

## Sampling

Sampling is the most impactful performance lever for traces. Unsampled spans return a noop span with virtually zero overhead — no attribute storage, no events, no export.

### Head Sampling Configuration

Configure a sampler on the `TracerProvider`:

```python
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.sampling import (
    TraceIdRatioBased,
    ParentBased,
    ALWAYS_ON,
    ALWAYS_OFF,
)

# Sample 10% of root traces
sampler = TraceIdRatioBased(0.1)

# Respect upstream decisions; ratio-sample new roots
sampler = ParentBased(root=TraceIdRatioBased(0.1))

provider = TracerProvider(sampler=sampler)
```

Via environment variables (no code change):

```bash
OTEL_TRACES_SAMPLER=traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
```

Supported `OTEL_TRACES_SAMPLER` values: `always_on`, `always_off`, `traceidratio`, `parentbased_always_on`, `parentbased_always_off`, `parentbased_traceidratio`.

### Sampling Decision Impact

```
ALWAYS_ON                   -> Full span lifecycle: allocation + recording + export
ParentBased(ALWAYS_ON)      -> Same, but respects upstream not-sampled decisions
TraceIdRatioBased(0.1)      -> ~90% of root spans become noops (near-zero cost)
ALWAYS_OFF                  -> All spans noop — useful for benchmarking app overhead
```

`ParentBased` is the recommended production default: it honors upstream sampling decisions propagated via W3C TraceContext while allowing ratio-based sampling at service entry points.

> **Tail sampling**: For sampling decisions based on complete trace data (error status, latency), use the OpenTelemetry Collector's `tail_sampling` processor rather than SDK-level head sampling. SDK head sampling combined with Collector tail sampling is a common production pattern.

---

## BatchSpanProcessor Tuning

`BatchSpanProcessor` buffers completed spans and exports them asynchronously in batches.

```
from opentelemetry.sdk.trace.export import BatchSpanProcessor
```

### How It Works

```
Application thread              Background thread
      |                                |
  span.end() --enqueue-->  queue (max_queue_size)
      |                                |
      |                      schedule_delay_millis elapses
      |                      OR batch reaches max_export_batch_size
      |                                |
      |                       --export batch--> Exporter
```

### Constructor Arguments and Env Vars

| Constructor arg | Environment variable | Purpose |
|----------------|---------------------|---------|
| `max_queue_size` | `OTEL_BSP_MAX_QUEUE_SIZE` | In-memory queue capacity |
| `max_export_batch_size` | `OTEL_BSP_MAX_EXPORT_BATCH_SIZE` | Spans per export call |
| `schedule_delay_millis` | `OTEL_BSP_SCHEDULE_DELAY` | Max wait before export |
| `export_timeout_millis` | `OTEL_BSP_EXPORT_TIMEOUT` | Stored but not applied by `BatchSpanProcessor` in SDK 1.43.0 |

### Tuning for Throughput

For high-volume services:

```python
from opentelemetry.sdk.trace.export import BatchSpanProcessor

bsp = BatchSpanProcessor(
    exporter,
    max_queue_size=8192,          # Absorb bursts
    max_export_batch_size=1024,   # Fewer network calls per flush
    schedule_delay_millis=10_000, # Allow larger batches to accumulate
)
```

### Tuning for Latency

For services where trace delivery speed matters (live debugging, alerting):

```python
bsp = BatchSpanProcessor(
    exporter,
    max_export_batch_size=128,   # Export smaller batches sooner
    schedule_delay_millis=2_000, # Export more frequently
)
```

### Queue-Full Behavior

When the queue is full, adding a new span evicts the oldest queued span and logs
`Queue full, dropping Span.` The SDK prioritizes application throughput over
telemetry completeness. Monitor queue-full warnings and the queue capacity set
by `OTEL_BSP_MAX_QUEUE_SIZE`.

### SimpleSpanProcessor

`SimpleSpanProcessor` exports spans synchronously on `span.end()`, adding exporter latency to every span boundary. Use it for:

- Tests and development (deterministic, no background thread)
- Short-lived CLI tools that must export before exit

```python
from opentelemetry.sdk.trace.export import SimpleSpanProcessor

provider = TracerProvider()
provider.add_span_processor(SimpleSpanProcessor(exporter))
```

Avoid `SimpleSpanProcessor` in production services — it blocks the calling thread on every `span.end()`.

---

## Metric Reader Tuning

### PeriodicExportingMetricReader

`PeriodicExportingMetricReader` collects and exports metrics on a fixed interval.

```python
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader

reader = PeriodicExportingMetricReader(
    exporter,
    export_interval_millis=30_000,  # More frequent than default
    export_timeout_millis=15_000,   # Timeout per export attempt
)
```

Or via environment variables:

```bash
OTEL_METRIC_EXPORT_INTERVAL=30000
OTEL_METRIC_EXPORT_TIMEOUT=15000
```

Interval tradeoffs:

- Longer interval (default): lower overhead, suitable for dashboards and alerting
- 15–30s: near-real-time monitoring, moderately higher overhead
- 5–10s: high-frequency use cases, significant CPU and network overhead

---

## Views for Cardinality Control

Cardinality is the number of unique label-value combinations across all recorded attributes. Unbounded attributes (user IDs, request IDs, trace IDs) produce one time series per unique value and are a common source of memory and storage blowup.

`View` instances filter or transform metrics before they reach storage.

```python
from opentelemetry.sdk.metrics.view import View, DropAggregation
from opentelemetry.sdk.metrics import MeterProvider
```

### Allowlist Attribute Filter

Keep only specific attributes on a metric, dropping everything else:

```python
view = View(
    instrument_name="http.server.request.duration",
    attribute_keys={"http.request.method", "http.response.status_code"},
)
```

`attribute_keys` is a `set[str]`. Any recorded attribute not in the set is discarded before aggregation. An empty set `set()` drops all attributes, producing a single time series for that metric.

### Drop an Entire Metric

```python
drop_view = View(
    instrument_name="debug.*",  # Wildcard matching supported
    aggregation=DropAggregation(),
)
```

### Applying Views

```python
mp = MeterProvider(
    metric_readers=[reader],
    views=[view, drop_view],
)
```

Every matching View creates a metric stream, with two exceptions: a View using
`DropAggregation` matches but produces no stream (it discards the instrument's
measurements), and a View that is incompatible with the instrument (for example,
an explicit-bucket histogram on an asynchronous instrument) is skipped with a
warning. Ordering does not make the first match win. Overlapping Views can
therefore produce multiple streams; when their metric identities conflict the SDK
logs a warning but still emits the conflicting streams. A metric with no matching
View uses the default aggregation.

Attributes like HTTP method (~10 values) or response status code (~50 values) are bounded. Attributes like `user.id`, `request.id`, or `session.id` are unbounded and should be filtered out with Views unless you specifically intend per-user metrics.

---

## asyncio Context Propagation

The Python SDK uses `contextvars.ContextVar` (via the `opentelemetry-api` `Context` type) to propagate the active span and baggage. `asyncio` tasks automatically inherit a copy of the current `contextvars` context when created with `asyncio.create_task()` or `loop.create_task()`.

### Correct Pattern in asyncio

```python
import asyncio
from opentelemetry import trace

tracer = trace.get_tracer(__name__)

async def handle_request():
    with tracer.start_as_current_span("handle_request") as span:
        # Child tasks created here inherit this span's context
        await asyncio.gather(
            fetch_data(),
            fetch_config(),
        )

async def fetch_data():
    # The active span from the parent coroutine is visible here
    with tracer.start_as_current_span("fetch_data"):
        ...
```

### Pitfall: Crossing into Threads

`asyncio.to_thread()` copies the current `contextvars` context into the worker thread, so the active span is available there. Mutations to `ContextVar` inside the thread do not propagate back to the asyncio event loop:

```python
import asyncio
from opentelemetry import trace

tracer = trace.get_tracer(__name__)

def blocking_work():
    # The span from the calling coroutine IS visible here (context was copied in)
    current_span = trace.get_current_span()
    # Any span started here is local to this thread's context copy
    with tracer.start_as_current_span("blocking_work"):
        ...
    # This span does NOT become the active span back in the event loop

async def main():
    with tracer.start_as_current_span("main"):
        await asyncio.to_thread(blocking_work)
        # Active span here is still "main" — the thread's mutations are gone
```

`loop.run_in_executor(...)` and manually spawned threads may not automatically copy the current context. Prefer `asyncio.to_thread()` when possible; otherwise pass the context explicitly for deterministic behavior:

```python
import asyncio
import threading
from contextvars import copy_context
from opentelemetry import context

# run_in_executor: wrap the callable in the copied context
async def main():
    loop = asyncio.get_running_loop()
    ctxvars = copy_context()
    await loop.run_in_executor(None, ctxvars.run, blocking_work)

# manual thread: attach the captured OTel context
ctx = context.get_current()  # Capture context in the calling thread

def worker():
    token = context.attach(ctx)  # Attach the captured context
    try:
        with tracer.start_as_current_span("worker"):
            ...
    finally:
        context.detach(token)

t = threading.Thread(target=worker)
t.start()
```

### Pitfall: Context Lost Across `await` with Manual Context Setting

Avoid manually calling `context.attach()` across `await` boundaries without matching `context.detach()`. Unmatched attaches accumulate context frames and create subtle propagation bugs. Prefer `start_as_current_span()` as a context manager, which handles attach/detach automatically.

---

## Exporter Configuration

### gRPC vs HTTP

| Aspect | gRPC (`otlp.proto.grpc`) | HTTP (`otlp.proto.http`) |
|--------|--------------------------|--------------------------|
| Default port | 4317 | 4318 |
| Connection model | Persistent, multiplexed | HTTP/1.1 or HTTP/2 |
| Compression | Optional gzip | Optional gzip |
| Best for | High throughput, stable connections | Firewalls, load balancers, simpler setup |

```python
# gRPC exporter
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

exporter = OTLPSpanExporter(
    endpoint="http://localhost:4317",
    compression=Compression.Gzip,  # from grpc import Compression
)

# HTTP exporter
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter

exporter = OTLPSpanExporter(
    endpoint="http://localhost:4318/v1/traces",
)
```

### Compression

Enable gzip compression to reduce bandwidth at the cost of CPU:

```bash
OTEL_EXPORTER_OTLP_COMPRESSION=gzip
```

Or programmatically for gRPC:

```python
from grpc import Compression

exporter = OTLPSpanExporter(compression=Compression.Gzip)
```

### Retry and Timeout

The OTLP exporters retry on transient errors (connection refused, 5xx responses). Check the current retry defaults in the [exporter source](https://github.com/open-telemetry/opentelemetry-python/tree/main/exporter) — do not assume specific backoff intervals.

Configure timeout via environment variable or constructor:

```bash
OTEL_EXPORTER_OTLP_TIMEOUT=5   # seconds
```

```python
exporter = OTLPSpanExporter(timeout=5)  # seconds in constructor
```

Note: Python interprets `OTEL_EXPORTER_OTLP_TIMEOUT` and the `timeout=` constructor argument in **seconds** (default `10`). This deviates from the OpenTelemetry specification, which defines the variable in milliseconds (default `10000`). A value of `5` is a 5-second timeout in Python, not 5 milliseconds.

Lower timeout: fail fast and free the batch processor for the next export cycle.
Higher timeout: accommodate large batches or slow backends.

---

## Log Handler Cost

The OpenTelemetry Python SDK integrates with Python's standard `logging` module via `LoggingHandler`. Log records emitted through this handler are converted to OTel log records and processed by the `LoggerProvider`.

```python
import logging
from opentelemetry.sdk._logs import LoggerProvider
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.instrumentation.logging.handler import LoggingHandler

logger_provider = LoggerProvider()
logger_provider.add_log_record_processor(
    BatchLogRecordProcessor(log_exporter)
)

handler = LoggingHandler(level=logging.WARNING, logger_provider=logger_provider)
logging.getLogger().addHandler(handler)
```

Performance considerations:

- **Level filter first**: Set `LoggingHandler(level=logging.WARNING)` to avoid processing DEBUG/INFO records entirely. Python's standard `logging` level check happens before the OTel handler is invoked.
- **BatchLogRecordProcessor vs SimpleLogRecordProcessor**: Prefer `BatchLogRecordProcessor` in production to avoid blocking the logging call on export I/O. `SimpleLogRecordProcessor` exports synchronously on every log record.
- **Body construction cost**: Avoid constructing expensive log message bodies (e.g. via `%`-formatting with heavy objects) at the call site when the level is filtered. Python's `logging` lazy `%`-formatting applies only if the record passes the level filter.

---

## Graceful Shutdown

Proper shutdown ensures buffered telemetry is flushed before the process exits.

```python
import signal
import sys

def shutdown(sig=None, frame=None):
    # Shutdown order: providers flush their processors, then exporters close
    tracer_provider.shutdown()
    meter_provider.shutdown()
    logger_provider.shutdown()
    sys.exit(0)

signal.signal(signal.SIGTERM, shutdown)
signal.signal(signal.SIGINT, shutdown)
```

`shutdown()` blocks until all buffered data is exported or the processor's timeout elapses.

### force_flush()

`force_flush()` synchronously exports all buffered data immediately:

```python
# Force-flush before a checkpoint or test assertion
tracer_provider.force_flush()
meter_provider.force_flush()
logger_provider.force_flush()
```

`force_flush()` is synchronous and adds latency at the call site. Do not call it in the request hot path. Use it for:

- Pre-fork checkpoints (e.g. before `os.fork()`)
- Test teardown (ensure spans are exported before assertions)
- Graceful drain during rolling deploys

### asyncio Shutdown

In an asyncio application, wrap provider shutdown in a coroutine or run it from the event loop's cleanup:

```python
import asyncio

async def shutdown_providers():
    loop = asyncio.get_running_loop()
    # Providers are synchronous; run in executor to avoid blocking the loop
    await loop.run_in_executor(None, tracer_provider.shutdown)
    await loop.run_in_executor(None, meter_provider.shutdown)
    await loop.run_in_executor(None, logger_provider.shutdown)
```

---

## Telemetry Pipeline Reliability

The SDK is designed so that telemetry failures do not crash or slow the application:

- **Span creation never raises** — after provider shutdown it may still return a recording span,
  but the shut-down processors/exporters no longer process or export that span
- **Metric recording never raises** — measurements are silently dropped on failure
- **Export failures are retried** — then dropped after the timeout or max retries
- **Queue overflow drops spans** — the application is not blocked

---

## Monitoring the Pipeline

Watch for SDK-emitted warnings in application logs. The `BatchSpanProcessor` logs when spans are dropped due to queue overflow. Enable SDK debug logging during load testing:

```python
import logging
logging.getLogger("opentelemetry").setLevel(logging.DEBUG)
```

Key signals to watch:

| Indicator | Meaning |
|-----------|---------|
| `Queue full, dropping Span.` warnings | Queue overflow — increase `max_queue_size` or reduce `schedule_delay_millis` |
| Export timeout errors | Backend/exporter too slow or batch too large — tune the exporter timeout or `max_export_batch_size`; BSP `export_timeout_millis` is not applied in SDK 1.43.0 |
| High memory growth | Metric cardinality explosion — add Views to filter attributes |
