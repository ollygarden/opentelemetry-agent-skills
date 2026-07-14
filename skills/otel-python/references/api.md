# OpenTelemetry Python API

**Sources of Truth:** [opentelemetry-python](https://github.com/open-telemetry/opentelemetry-python) · [opentelemetry-python-contrib](https://github.com/open-telemetry/opentelemetry-python-contrib) · [Python API docs](https://opentelemetry-python.readthedocs.io/)

## Import Paths

### API packages (stable, no SDK dependency)
```python
from opentelemetry import trace               # TracerProvider, get_tracer
from opentelemetry import metrics             # MeterProvider, get_meter
from opentelemetry._logs import get_logger_provider  # LoggerProvider access
from opentelemetry import baggage
from opentelemetry import context
from opentelemetry import propagate           # inject / extract
```

### SDK packages (implementation, required at runtime)
```python
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk._logs import LoggerProvider
from opentelemetry.sdk.resources import Resource
```

SDK packages ship in `opentelemetry-sdk`. The API packages ship in `opentelemetry-api`. Application code should import from the API; SDK imports belong in bootstrap/setup code.

## Global API Access

### Provider registration (bootstrap code)
```python
from opentelemetry import trace, metrics
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk._logs import LoggerProvider
from opentelemetry._logs import set_logger_provider

tracer_provider = TracerProvider(resource=resource)
trace.set_tracer_provider(tracer_provider)

meter_provider = MeterProvider(resource=resource)
metrics.set_meter_provider(meter_provider)

logger_provider = LoggerProvider(resource=resource)
set_logger_provider(logger_provider)
```

### Obtaining instances (application code)
```python
tracer = trace.get_tracer("my.library")
meter  = metrics.get_meter("my.library")
logger = get_logger_provider().get_logger("my.library")
```

Use the instrumenting library's import path or package name as the scope name. Pass `schema_url` to align with a semantic conventions version — see `otel-semantic-conventions`.

## Tracing API

### Creating spans
```python
tracer = trace.get_tracer("my.library")

# Context-manager form — preferred
with tracer.start_as_current_span("operation.name") as span:
    span.set_attribute("key", "value")
    do_work()

# Manual form — use when you need the span across call boundaries
span = tracer.start_span("operation.name")
token = context.attach(trace.set_span_in_context(span))
try:
    do_work()
finally:
    context.detach(token)
    span.end()
```

### Span attributes and status
```python
from opentelemetry.trace import StatusCode

span.set_attribute("http.request.method", "GET")   # prefer semantic conventions
span.set_attributes({"key1": "v1", "key2": 42})

# Record an exception and mark the span as failed
try:
    risky()
except Exception as exc:
    span.record_exception(exc)
    span.set_status(StatusCode.ERROR, description=str(exc))
    raise
```

Attribute names: follow [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/). Cross-reference the `otel-semantic-conventions` skill.

### Span kind
```python
from opentelemetry.trace import SpanKind

tracer.start_as_current_span("http.request", kind=SpanKind.CLIENT)
# Values: INTERNAL (default), SERVER, CLIENT, PRODUCER, CONSUMER
```

### Context utilities
```python
# Get the active span
span = trace.get_current_span()

# Extract trace/span IDs
ctx = span.get_span_context()
trace_id = format(ctx.trace_id, "032x")
span_id  = format(ctx.span_id, "016x")

# Run code in a specific span's context
with trace.use_span(span):
    child_span = tracer.start_span("child")
```

## Metrics API

### Synchronous instruments
```python
meter = metrics.get_meter("my.library")

# Counter — monotonic, additive
counter = meter.create_counter(
    "requests.total",
    description="Total HTTP requests",
    unit="1",
)
counter.add(1, {"http.request.method": "GET", "http.response.status_code": 200})

# UpDownCounter — non-monotonic, additive (e.g., queue depth)
queue_size = meter.create_up_down_counter("queue.size", unit="1")
queue_size.add(1)   # enqueue
queue_size.add(-1)  # dequeue

# Histogram — distribution of values
duration = meter.create_histogram("request.duration", unit="ms")
duration.record(42.5, {"http.request.method": "GET"})
```

### Asynchronous (observable) instruments
```python
import psutil

def cpu_observer(options):
    yield metrics.Observation(psutil.cpu_percent(), {"cpu": "total"})

meter.create_observable_gauge(
    "system.cpu.utilization",
    callbacks=[cpu_observer],
    unit="%",
)

# Also: create_observable_counter, create_observable_up_down_counter
```

The callback receives an `options` argument and must `yield` (or return) `Observation` objects. Callbacks are invoked by the SDK on each collection cycle.

## Attributes

Python attribute values are `str | bool | int | float` or sequences thereof.

```python
span.set_attribute("service.name", "api")
span.set_attribute("http.response.status_code", 200)
span.set_attribute("db.query.parameter.0", "alice")

# Metric attributes are plain dicts
counter.add(1, {"http.request.method": "GET"})
```

Attribute keys must follow semantic conventions where applicable — see `otel-semantic-conventions`.

## Propagation

`opentelemetry.propagate` delegates to the globally registered `TextMapPropagator` (default: W3C TraceContext + Baggage).

```python
from opentelemetry import propagate

# Inject into outgoing headers (e.g., requests dict)
headers = {}
propagate.inject(headers)

# Extract from incoming headers (e.g., WSGI environ or dict)
ctx = propagate.extract(carrier=request.headers)

# Run handler in extracted context
token = context.attach(ctx)
try:
    handle_request()
finally:
    context.detach(token)
```

### Custom / additional propagators
```python
from opentelemetry.propagators.composite import CompositePropagator
from opentelemetry.propagators.b3 import B3MultiFormat
from opentelemetry import propagate

propagate.set_global_textmap(
    CompositePropagator([B3MultiFormat()])
)
```

## Logs / Python Logging Bridge

The OTel Python SDK does not replace `logging` — it bridges it. The bridge converts stdlib `logging.LogRecord` objects into OTel log records and forwards them to the configured `LoggerProvider`.

### Manual wiring

Package required: `opentelemetry-instrumentation-logging`

```python
import logging
from opentelemetry._logs import get_logger_provider
from opentelemetry.instrumentation.logging.handler import LoggingHandler

# Attach to the root logger so all loggers inherit it
logging.getLogger().addHandler(
    LoggingHandler(logger_provider=get_logger_provider())
)
logging.getLogger().setLevel(logging.INFO)

# Now use stdlib logging normally
log = logging.getLogger("my.module")
log.info("handled request")  # emitted as OTel log record with trace context
```

The handler automatically injects the active span's `trace_id` and `span_id` into each log record, correlating logs to traces.

> **Deprecated path:** `opentelemetry.sdk._logs.LoggingHandler` is deprecated as of SDK 1.40.0/0.61b0. Always import from `opentelemetry.instrumentation.logging.handler`.

### `LoggingInstrumentor` behavior

`LoggingInstrumentor().instrument()` (also reachable via the `opentelemetry-instrument` CLI when the logging instrumentation is installed) has two separate behaviors:

- It installs the contrib `LoggingHandler` by default, routing stdlib log records through the global `LoggerProvider`. Disable that with `OTEL_PYTHON_LOG_AUTO_INSTRUMENTATION=false` or `enable_log_auto_instrumentation=False`.
- It can inject `otelTraceID`, `otelSpanID`, `otelTraceSampled`, and `otelServiceName` into stdlib `LogRecord`s for text log correlation. Use `inject_trace_context=True` to add those fields without changing the logging format, or `set_logging_format=True` / `OTEL_PYTHON_LOG_CORRELATION=true` to also call `logging.basicConfig` with a format that prints them.

```python
# Add correlation fields to LogRecord without changing logging.basicConfig
from opentelemetry.instrumentation.logging import LoggingInstrumentor
LoggingInstrumentor().instrument(inject_trace_context=True)
```
