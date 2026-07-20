# OpenTelemetry Python Instrumentation Libraries

## Detecting Existing Instrumentation

Before adding instrumentation, check whether it is already present:

```bash
# Check installed instrumentation packages
pip show opentelemetry-distro opentelemetry-instrumentation 2>/dev/null
pip list | grep opentelemetry-instrumentation-

# Check if the zero-code CLI wrapper is in use
grep -r "opentelemetry-instrument" Dockerfile docker-compose.yml Makefile .env* 2>/dev/null

# Check for per-app instrumentation in source
grep -rn "Instrumentor\|instrument_app\|instrument(" --include="*.py" .
```

If you find `opentelemetry-instrument` in the run command, the zero-code path is active and instrumentors are injected automatically. If you find `*Instrumentor.instrument_app(...)` calls in source, the per-app path is in use.

## Zero-Code Path

The zero-code path wraps the Python process with the `opentelemetry-instrument` CLI. No source changes are required.

### Install

```bash
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap -a install   # detects installed libs; installs matching instrumentors
```

`opentelemetry-bootstrap` reads `pip list`, maps installed packages to known instrumentors, and installs them. Re-run after adding new dependencies.

### Run

```bash
opentelemetry-instrument \
  --service_name my-service \
  python -m uvicorn app:app
```

Without `OTEL_CONFIG_FILE`, configuration comes from environment variables
(`OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_TRACES_SAMPLER`,
etc.) or distro defaults. With `opentelemetry-configuration` installed, the
same CLI can activate declarative YAML through `OTEL_CONFIG_FILE`; see
`declarative-setup.md`.

### Common env vars

| Variable | Example |
|----------|---------|
| `OTEL_SERVICE_NAME` | `my-service` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` |
| `OTEL_TRACES_EXPORTER` | `otlp` |
| `OTEL_METRICS_EXPORTER` | `otlp` |
| `OTEL_LOGS_EXPORTER` | `otlp` |
| `OTEL_PROPAGATORS` | `tracecontext,baggage` |
| `OTEL_PYTHON_DISABLED_INSTRUMENTATIONS` | `requests,logging` |
| `OTEL_SEMCONV_STABILITY_OPT_IN` | `http,database` |
| `OTEL_PYTHON_LOG_AUTO_INSTRUMENTATION` | `false` |

In contrib 0.65b0, the aiopg, asyncpg, Cassandra, pymemcache, pymongo, and
Tortoise ORM instrumentors added database semantic-convention migration
support. Their behavior now follows `OTEL_SEMCONV_STABILITY_OPT_IN`; audit
attributes and downstream queries before enabling `database` or
`database/dup`.

## Per-App Instrumentation

Use this form when you need fine-grained control, or when the CLI wrapper is not suitable (e.g., serverless, custom startup code, or the declarative config path).

### FastAPI (verified)

```python
from fastapi import FastAPI
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

app = FastAPI()
FastAPIInstrumentor.instrument_app(app)   # attaches ASGI middleware to this app instance
```

`instrument_app` is the per-app variant: it attaches the ASGI middleware to a specific app instance and produces `SpanKind.SERVER` spans for incoming requests.

> The API shape differs across instrumentors. FastAPI and Flask define static
> `instrument_app(app)` methods (the upstream Flask usage commonly calls it
> through `FlaskInstrumentor()`); Django uses the inherited instance method
> `DjangoInstrumentor().instrument()` with no app argument. Check the contrib
> package for the exact form.

### Flask

```python
from flask import Flask
from opentelemetry.instrumentation.flask import FlaskInstrumentor

app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)
```

### Django

For Django, call `instrument()` at startup (e.g., in `manage.py` or `wsgi.py`) before the WSGI app is created:

```python
from opentelemetry.instrumentation.django import DjangoInstrumentor

DjangoInstrumentor().instrument()
```

### Requests (HTTP client)

```python
from opentelemetry.instrumentation.requests import RequestsInstrumentor

RequestsInstrumentor().instrument()   # patches the requests library globally
```

## Contrib Catalog

Source of truth: `instrumentation/` directory in the [opentelemetry-python-contrib](https://github.com/open-telemetry/opentelemetry-python-contrib) repo. Do not pin versions in configuration — fetch the exact version from PyPI at install time.

### Web Frameworks

| Library | Package |
|---------|---------|
| FastAPI | `opentelemetry-instrumentation-fastapi` |
| Flask | `opentelemetry-instrumentation-flask` |
| Django | `opentelemetry-instrumentation-django` |
| Starlette | `opentelemetry-instrumentation-starlette` |
| Tornado | `opentelemetry-instrumentation-tornado` |
| Falcon | `opentelemetry-instrumentation-falcon` |
| Pyramid | `opentelemetry-instrumentation-pyramid` |
| ASGI (generic) | `opentelemetry-instrumentation-asgi` |
| WSGI (generic) | `opentelemetry-instrumentation-wsgi` |

### HTTP Clients

| Library | Package |
|---------|---------|
| requests | `opentelemetry-instrumentation-requests` |
| httpx | `opentelemetry-instrumentation-httpx` |
| urllib3 | `opentelemetry-instrumentation-urllib3` |
| urllib (stdlib) | `opentelemetry-instrumentation-urllib` |
| aiohttp client | `opentelemetry-instrumentation-aiohttp-client` |
| aiohttp server | `opentelemetry-instrumentation-aiohttp-server` |

### Databases

| Library | Package |
|---------|---------|
| psycopg (v3) | `opentelemetry-instrumentation-psycopg` |
| psycopg2 | `opentelemetry-instrumentation-psycopg2` |
| SQLAlchemy | `opentelemetry-instrumentation-sqlalchemy` |
| MySQL (mysql-connector) | `opentelemetry-instrumentation-mysql` |
| mysqlclient | `opentelemetry-instrumentation-mysqlclient` |
| PyMySQL | `opentelemetry-instrumentation-pymysql` |
| pymssql | `opentelemetry-instrumentation-pymssql` |
| SQLite3 | `opentelemetry-instrumentation-sqlite3` |
| asyncpg | `opentelemetry-instrumentation-asyncpg` |
| aiopg | `opentelemetry-instrumentation-aiopg` |
| pymongo | `opentelemetry-instrumentation-pymongo` |
| Redis | `opentelemetry-instrumentation-redis` |
| pymemcache | `opentelemetry-instrumentation-pymemcache` |
| Cassandra | `opentelemetry-instrumentation-cassandra` |
| Tortoise ORM | `opentelemetry-instrumentation-tortoiseorm` |
| DB-API 2.0 (generic) | `opentelemetry-instrumentation-dbapi` |

`opentelemetry-instrumentation-elasticsearch` was removed from contrib in
0.65b0 because the supported Elasticsearch client versions provide native OTel
instrumentation. Remove that package when upgrading and follow the client's
instrumentation guidance.

### Messaging

| Library | Package |
|---------|---------|
| Celery | `opentelemetry-instrumentation-celery` |
| kafka-python | `opentelemetry-instrumentation-kafka-python` |
| confluent-kafka | `opentelemetry-instrumentation-confluent-kafka` |
| aiokafka | `opentelemetry-instrumentation-aiokafka` |
| pika (RabbitMQ) | `opentelemetry-instrumentation-pika` |
| aio-pika (RabbitMQ async) | `opentelemetry-instrumentation-aio-pika` |
| boto3 SQS | `opentelemetry-instrumentation-boto3sqs` |
| botocore | `opentelemetry-instrumentation-botocore` |
| Remoulade | `opentelemetry-instrumentation-remoulade` |

### Logging / System

| Library | Package |
|---------|---------|
| logging (stdlib) | `opentelemetry-instrumentation-logging` |
| uncaught exceptions (process/thread/asyncio → logs) | `opentelemetry-instrumentation-exceptions` |
| system-metrics | `opentelemetry-instrumentation-system-metrics` |
| Jinja2 | `opentelemetry-instrumentation-jinja2` |
| asyncio | `opentelemetry-instrumentation-asyncio` |
| threading | `opentelemetry-instrumentation-threading` |
| click | `opentelemetry-instrumentation-click` |
| asyncclick | `opentelemetry-instrumentation-asyncclick` |
| AWS Lambda | `opentelemetry-instrumentation-aws-lambda` |
| gRPC | `opentelemetry-instrumentation-grpc` |

## Manual Instrumentation Patterns

Use these when no contrib instrumentor covers the target. Follow semconv — see the `otel-semantic-conventions` skill for attribute names.

Use the current stable attribute constants from `opentelemetry.semconv.attributes` (e.g. `http.request.method`, `url.full`, `db.collection.name`). These match what the contrib instrumentors emit. The older `opentelemetry.semconv.trace.SpanAttributes.HTTP_*` / `DB_*` names (`http.method`, `http.url`, `db.sql.table`) are deprecated — avoid them so manual and auto-instrumented spans stay consistent.

### HTTP Client Call

```python
import requests
from opentelemetry import trace
from opentelemetry.semconv.attributes import http_attributes, url_attributes

tracer = trace.get_tracer(__name__)

def call_external_api(url: str) -> dict:
    with tracer.start_as_current_span(
        "GET",
        kind=trace.SpanKind.CLIENT,
        attributes={
            http_attributes.HTTP_REQUEST_METHOD: "GET",
            url_attributes.URL_FULL: url,
        },
    ) as span:
        resp = requests.get(url)
        span.set_attribute(
            http_attributes.HTTP_RESPONSE_STATUS_CODE, resp.status_code
        )
        return resp.json()
```

### Database Call

```python
from opentelemetry import trace
from opentelemetry.semconv.attributes import db_attributes

tracer = trace.get_tracer(__name__)

def fetch_user(conn, user_id: str) -> dict:
    with tracer.start_as_current_span(
        "SELECT users",
        kind=trace.SpanKind.CLIENT,
        attributes={
            db_attributes.DB_SYSTEM_NAME: "postgresql",
            db_attributes.DB_OPERATION_NAME: "SELECT",
            db_attributes.DB_COLLECTION_NAME: "users",
        },
    ) as span:
        row = conn.execute("SELECT * FROM users WHERE id = %s", (user_id,)).fetchone()
        span.set_attribute("db.rows_returned", 1 if row else 0)
        return row
```

### Background Job

```python
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode

tracer = trace.get_tracer(__name__)

def process_batch(batch_id: str, items: list) -> None:
    with tracer.start_as_current_span("process_batch") as span:
        span.set_attribute("batch.id", batch_id)
        span.set_attribute("batch.size", len(items))
        processed = 0
        for item in items:
            try:
                handle_item(item)
                processed += 1
            except Exception as exc:
                span.record_exception(exc)
        span.set_attribute("batch.processed", processed)
        span.set_attribute("batch.failed", len(items) - processed)
        if processed < len(items):
            span.set_status(Status(StatusCode.ERROR, "some items failed"))
```

## Enriching Auto-Instrumented Spans

When a contrib instrumentor already creates a span (e.g., FastAPI's `GET /work` span), retrieve it from context and add attributes — do not create a new span.

```python
from opentelemetry import trace

@app.get("/orders")
def list_orders(customer_id: str):
    # Span created by FastAPIInstrumentor; enrich it instead of creating a child
    span = trace.get_current_span()
    span.set_attribute("customer.id", customer_id)
    span.set_attribute("business.operation", "list_orders")

    orders = db.query_orders(customer_id)
    span.set_attribute("result.count", len(orders))
    return orders
```

`trace.get_current_span()` returns the active span or a no-op span if none is active — it is always safe to call.
