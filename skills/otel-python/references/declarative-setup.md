# Python SDK Setup with Declarative File Configuration

Configure the OpenTelemetry SDK in Python via a YAML file processed by
`opentelemetry.sdk._configuration.file`.

> **Experimental / private API.** The `opentelemetry.sdk._configuration.*`
> modules are private (leading underscore) and experimental. They may change
> or be removed without a deprecation notice between SDK releases. Check the
> SDK CHANGELOG (see Sources of Truth) before upgrading.

For the YAML configuration schema, load the `otel-declarative-config` skill.

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the
`otel-declarative-config` skill. For Python-specific facts:

| Fact | Fetch |
|---|---|
| Latest `opentelemetry-sdk` / `opentelemetry-api` | `WebFetch https://pypi.org/pypi/opentelemetry-sdk/json` (`.info.version`) |
| Latest `opentelemetry-distro` | `WebFetch https://pypi.org/pypi/opentelemetry-distro/json` |
| Latest OTLP exporter | `WebFetch https://pypi.org/pypi/opentelemetry-exporter-otlp/json` |
| Declarative config support + vendored schema version | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/main/opentelemetry-sdk/src/opentelemetry/sdk/_configuration/README.md` |
| SDK CHANGELOG | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/main/CHANGELOG.md` |

## Install

The `file-configuration` extras group is required; plain `opentelemetry-sdk`
lacks `pyyaml` and `jsonschema` and `load_config_file` will raise `ImportError`.

```bash
pip install "opentelemetry-sdk[file-configuration]" opentelemetry-api
```

Add exporters and instrumentation libraries as needed:

```bash
pip install opentelemetry-exporter-otlp opentelemetry-distro \
            opentelemetry-instrumentation-fastapi \
            opentelemetry-instrumentation-logging
```

## Activation

**Programmatic only** — there is no `OTEL_CONFIG_FILE` environment variable
wiring. A bootstrap module must be imported before any application code.

### The dataclass-conversion gotcha

`load_config_file()` returns an `OpenTelemetryConfiguration` dataclass, but
its nested fields (`resource`, `tracer_provider`, `meter_provider`,
`logger_provider`) are plain `dict` objects, **not** the typed dataclasses
from `models.py`. The `configure_*` functions call attribute access (e.g.
`.processors`, `.sampler`) on these fields and will fail if they receive raw
dicts.

**Workaround:** use `load_config_file` only for YAML parsing and env-var
substitution, then build the dataclass tree manually before calling
`configure_*`.

### Minimal bootstrap pattern

```python
"""bootstrap.py — import before any application code."""
import os

from opentelemetry.sdk._configuration.file import load_config_file
from opentelemetry.sdk._configuration.models import (
    AttributeNameValue,
    Resource as ResourceConfig,
    TracerProvider as TracerProviderConfig,
    SpanProcessor, SimpleSpanProcessor, SpanExporter,
    MeterProvider as MeterProviderConfig,
    MetricReader, PeriodicMetricReader,
    PushMetricExporter, ConsoleMetricExporter,
    LoggerProvider as LoggerProviderConfig,
    LogRecordProcessor, SimpleLogRecordProcessor, LogRecordExporter,
)
from opentelemetry.sdk._configuration._resource import create_resource
from opentelemetry.sdk._configuration._tracer_provider import configure_tracer_provider
from opentelemetry.sdk._configuration._meter_provider import configure_meter_provider
from opentelemetry.sdk._configuration._logger_provider import configure_logger_provider
from opentelemetry.sdk._configuration._propagator import configure_propagator

# Parses YAML + performs env-var substitution. Nested fields are plain dicts.
_raw = load_config_file(os.environ.get("OTEL_PY_CONFIG", "otel.yaml"))

# Build Resource from the dict returned by the loader.
_resource_data = _raw.resource or {}
_attrs = [
    AttributeNameValue(name=a["name"], value=a["value"])
    for a in (_resource_data.get("attributes") or [])
]
_resource = create_resource(ResourceConfig(attributes=_attrs or None))

# Configure each provider by constructing typed dataclasses explicitly.
configure_tracer_provider(
    TracerProviderConfig(
        processors=[SpanProcessor(simple=SimpleSpanProcessor(exporter=SpanExporter(console={})))]
    ),
    _resource,
)

configure_meter_provider(
    MeterProviderConfig(
        readers=[MetricReader(periodic=PeriodicMetricReader(
            interval=5000,
            exporter=PushMetricExporter(console=ConsoleMetricExporter()),
        ))]
    ),
    _resource,
)

configure_logger_provider(
    LoggerProviderConfig(
        processors=[LogRecordProcessor(simple=SimpleLogRecordProcessor(exporter=LogRecordExporter(console={})))]
    ),
    _resource,
)

# WARNING: do NOT call configure_propagator(None).
# Passing None installs an empty CompositePropagator whose extract() returns
# None. context.attach(None) then corrupts the active context, crashing ASGI
# middleware during span creation.
# Only configure propagators when explicitly present in the config file.
if _raw.propagator is not None:
    configure_propagator(_raw.propagator)
```

### Application entry point

```python
import bootstrap  # noqa: F401 — must be first import

from opentelemetry import trace, metrics
# ... rest of app imports
```

The import order matters. Any OTel API call that happens before `bootstrap` is
imported will use no-op providers.

## YAML Config

`file_format` must be the literal string `'1.0'`. Omitting it or using a
different value causes the parser to reject the file.

Minimal verified skeleton (all three signals, console exporters):

```yaml
file_format: "1.0"
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-my-service}"
    - name: service.version
      value: "0.1.0"
tracer_provider:
  processors:
    - simple:
        exporter:
          console: {}
meter_provider:
  readers:
    - periodic:
        interval: 5000
        exporter:
          console: {}  # Must be ConsoleMetricExporter() in code, not empty dict
logger_provider:
  processors:
    - simple:
        exporter:
          console: {}
```

For OTLP exporters and the full schema, load the `otel-declarative-config` skill.

Env-var substitution uses `${VAR}` and `${VAR:-default}` syntax; it is handled
by `load_config_file` before the YAML is further processed.

## Key Facts

- **`configure_*` sets globals.** `configure_tracer_provider` calls
  `trace.set_tracer_provider(...)`, `configure_meter_provider` calls
  `metrics.set_meter_provider(...)`, and `configure_logger_provider` calls
  `set_logger_provider(...)`. After the bootstrap runs, `trace.get_tracer(...)` etc.
  return providers backed by the configured SDK.

- **Absent section ⇒ global left unset.** If you omit `tracer_provider` from the
  YAML (and skip the corresponding `configure_tracer_provider` call), the global
  tracer provider remains the no-op default. Each signal is independent.

- **`ConsoleMetricExporter` must be a typed dataclass.** For metrics,
  `PushMetricExporter(console=ConsoleMetricExporter())` is required. Passing
  `{}` (which works for trace/log console exporters) fails because the meter
  provider factory calls `.temporality_preference` on the exporter object.

- **`LoggingHandler` import.** The deprecated
  `opentelemetry.sdk._logs.LoggingHandler` (deprecated in 0.63b1+) should be
  replaced with `opentelemetry.instrumentation.logging.handler.LoggingHandler`
  (from `opentelemetry-instrumentation-logging`).

- **Env substitution** is handled by `load_config_file`; see `otel-declarative-config`
  for the substitution syntax reference.
