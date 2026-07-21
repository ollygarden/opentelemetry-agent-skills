# Python SDK Setup with Declarative File Configuration

Configure the OpenTelemetry SDK in Python via a YAML file processed by
the `opentelemetry-configuration` package (`opentelemetry.configuration`).

> **Experimental public package.** Release 1.44.0 / 0.65b0 moved declarative
> configuration from the private `opentelemetry.sdk._configuration.*` modules
> into the public `opentelemetry-configuration` package. The package is still
> experimental (Development Status: Alpha), so its API and models may change
> between minor releases. Pin and audit the changelog when upgrading.

For the YAML configuration schema, load the `otel-declarative-config` skill.

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the
`otel-declarative-config` skill. For Python-specific facts:

| Fact | Fetch |
|---|---|
| Latest `opentelemetry-sdk` / `opentelemetry-api` | `WebFetch https://pypi.org/pypi/opentelemetry-sdk/json` (`.info.version`) |
| Latest `opentelemetry-configuration` | `WebFetch https://pypi.org/pypi/opentelemetry-configuration/json` |
| Latest `opentelemetry-distro` | `WebFetch https://pypi.org/pypi/opentelemetry-distro/json` |
| Latest OTLP exporter | `WebFetch https://pypi.org/pypi/opentelemetry-exporter-otlp/json` |
| Declarative config support (released 1.44.0) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/v1.44.0/opentelemetry-configuration/README.rst` |
| Vendored schema (released 1.44.0) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/v1.44.0/opentelemetry-configuration/src/opentelemetry/configuration/schema.json` |
| SDK CHANGELOG through 1.44.0 | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/v1.44.0/CHANGELOG.md` |

## Install

Install the separate experimental package directly. It installs matching
`opentelemetry-api` and `opentelemetry-sdk` versions plus the YAML/schema
dependencies:

```bash
pip install opentelemetry-configuration
```

`opentelemetry-sdk[file-configuration]` remains as a deprecated compatibility
alias that installs `opentelemetry-configuration`; prefer the direct package for
new setups.

Add exporters and instrumentation libraries as needed:

```bash
pip install opentelemetry-exporter-otlp opentelemetry-distro \
            opentelemetry-instrumentation-fastapi \
            opentelemetry-instrumentation-logging
```

## Activation

As of SDK **1.44.0 / 0.65b0** there are two supported activation paths.

### Zero-code: `OTEL_CONFIG_FILE`

Point the SDK configurator at a config file and it loads it in place of the
env-var init path — no bootstrap module required. This works with the
`opentelemetry-instrument` CLI (and any distro that runs the SDK configurator):

```bash
OTEL_CONFIG_FILE=otel.yaml opentelemetry-instrument python -m uvicorn app:app
```

When `OTEL_CONFIG_FILE` is set, the file is the **sole** source of SDK
construction: spec-defined `OTEL_*` variables that have schema equivalents are
ignored. Env vars are still read via `${VAR}` / `${VAR:-default}` substitution
inside the file and by components the file enables (e.g. resource detectors).
Python-specific `OTEL_PYTHON_*` configuration extensions are also bypassed
because the regular env-var initialization path is skipped; express equivalent
behavior in the file or in code.

### Programmatic: `configure_sdk`

`configure_sdk` is the single entry point that takes a parsed
`OpenTelemetryConfiguration`, builds the resource, and applies the
tracer/meter/logger providers and propagator globally — honoring the top-level
`disabled` flag. A bootstrap module must be imported before any application code.

```python
"""bootstrap.py — import before any application code."""
from opentelemetry.configuration import configure_sdk, load_config_file

configure_sdk(load_config_file("otel.yaml"))
```

`load_config_file` parses YAML/JSON, performs `${VAR}` and `${VAR:-default}` substitution,
validates against the vendored schema, and returns a **fully-typed**
`OpenTelemetryConfiguration` — nested fields (`resource`, `tracer_provider`,
`meter_provider`, `logger_provider`) are typed dataclasses, not raw dicts. (Prior
to 1.43.0 the loader returned raw dicts for nested fields and callers had to
build the dataclass tree by hand.) Both `load_config_file` and `configure_sdk`
are exported from `opentelemetry.configuration`.

For finer control, the per-signal factories (`configure_tracer_provider`,
`configure_meter_provider`, `configure_logger_provider`, `configure_propagator`,
`create_resource`) are exported from `opentelemetry.configuration.file`.

### Application entry point

```python
import bootstrap  # noqa: F401 — must be first import

from opentelemetry import trace, metrics
# ... rest of app imports
```

The import order matters. Any OTel API call that happens before `bootstrap` is
imported (or before the CLI activates the config) uses no-op providers.

## YAML Config

`file_format` is required and must be a string version. Release 1.44.0 validates it
per the configuration spec: unsupported major versions are rejected; newer minor
versions with the same major version are accepted with a warning. Use `"1.0"`
unless you have checked the currently vendored schema and SDK loader.
The package vendors configuration schema 1.1.0 in this release, while the loader
still declares 1.0 as its supported `file_format` target.

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
          console: {}
logger_provider:
  processors:
    - simple:
        exporter:
          console: {}
```

For OTLP exporters and the full schema, load the `otel-declarative-config` skill.

Env-var substitution uses `${VAR}` and `${VAR:-default}` syntax; it is handled
by `load_config_file` before the YAML is further processed. Use `$$` for a
literal dollar sign.

## Instrumentor Activation

Release 1.44.0 can activate installed contrib instrumentors from the
`instrumentation/development.python` section. Keys are
`opentelemetry_instrumentor` entry-point names; `enabled: false` skips one:

```yaml
instrumentation/development:
  python:
    requests:
      enabled: true
    urllib3:
      enabled: false
```

Additional keys are passed to `instrument()`. If an instrumentor publishes a
dataclass in its `configuration` attribute, those values are type-coerced and
validated first. Already-active instrumentors are skipped to avoid double
instrumentation; unknown or failing instrumentors are logged and do not stop
the remaining entries.

## Key Facts

- **`configure_sdk` / `configure_*` set globals.** `configure_sdk` applies every
  signal; individually `configure_tracer_provider` calls
  `trace.set_tracer_provider(...)`, `configure_meter_provider` calls
  `metrics.set_meter_provider(...)`, and `configure_logger_provider` calls
  `set_logger_provider(...)`. After the bootstrap runs, `trace.get_tracer(...)` etc.
  return providers backed by the configured SDK.

- **Absent section ⇒ global left unset.** A config section that is absent
  (`None`) leaves the corresponding global untouched — it stays the no-op
  default. Each signal is independent.

- **Configured ID generator is applied.** Release 1.44.0 wires the
  `tracer_provider.id_generator` configuration into `TracerProvider`; resolve
  the supported shape from the vendored schema.

- **`LoggingHandler` import.** The deprecated
  `opentelemetry.sdk._logs.LoggingHandler` (deprecated in 1.40.0/0.61b0) should be
  replaced with `opentelemetry.instrumentation.logging.handler.LoggingHandler`
  (from `opentelemetry-instrumentation-logging`).

- **Env substitution** is handled by `load_config_file`; see `otel-declarative-config`
  for the substitution syntax reference.
