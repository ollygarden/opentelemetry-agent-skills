# Python SDK Setup with Declarative File Configuration

Configure the OpenTelemetry SDK in Python via a YAML file processed by
`opentelemetry.sdk._configuration.file`.

> **Experimental / private API.** The `opentelemetry.sdk._configuration.*`
> modules are private (leading underscore) and experimental. They may change
> or be removed without a deprecation notice between SDK releases. Check the
> SDK CHANGELOG (see Sources of Truth) before upgrading.
>
> **Main-branch note (post-1.43.0):** upstream `main` has moved the
> declarative configuration implementation into an unreleased
> `opentelemetry-configuration` package. Keep released guidance on
> `opentelemetry.sdk._configuration.*` until that package is released.

For the YAML configuration schema, load the `otel-declarative-config` skill.

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the
`otel-declarative-config` skill. For Python-specific facts:

| Fact | Fetch |
|---|---|
| Latest `opentelemetry-sdk` / `opentelemetry-api` | `WebFetch https://pypi.org/pypi/opentelemetry-sdk/json` (`.info.version`) |
| Latest `opentelemetry-distro` | `WebFetch https://pypi.org/pypi/opentelemetry-distro/json` |
| Latest OTLP exporter | `WebFetch https://pypi.org/pypi/opentelemetry-exporter-otlp/json` |
| Declarative config support + vendored schema version (released 1.43.0) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-python/v1.43.0/opentelemetry-sdk/src/opentelemetry/sdk/_configuration/README.md` |
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

As of SDK **1.43.0 / 0.64b0** there are two supported activation paths.

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

### Programmatic: `configure_sdk`

`configure_sdk` is the single entry point that takes a parsed
`OpenTelemetryConfiguration`, builds the resource, and applies the
tracer/meter/logger providers and propagator globally — honoring the top-level
`disabled` flag. A bootstrap module must be imported before any application code.

```python
"""bootstrap.py — import before any application code."""
from opentelemetry.sdk._configuration.file import load_config_file, configure_sdk

configure_sdk(load_config_file("otel.yaml"))
```

`load_config_file` parses YAML/JSON, performs `${VAR}` and `${VAR:-default}` substitution,
validates against the vendored schema, and returns a **fully-typed**
`OpenTelemetryConfiguration` — nested fields (`resource`, `tracer_provider`,
`meter_provider`, `logger_provider`) are typed dataclasses, not raw dicts. (Prior
to 1.43.0 the loader returned raw dicts for nested fields and callers had to
build the dataclass tree by hand.) Both `load_config_file` and `configure_sdk`
are exported from `opentelemetry.sdk._configuration.file`.

For finer control, the per-signal factories (`configure_tracer_provider`,
`configure_meter_provider`, `configure_logger_provider`, `configure_propagator`,
`create_resource`) are also exported from the same package.

### Application entry point

```python
import bootstrap  # noqa: F401 — must be first import

from opentelemetry import trace, metrics
# ... rest of app imports
```

The import order matters. Any OTel API call that happens before `bootstrap` is
imported (or before the CLI activates the config) uses no-op providers.

## YAML Config

`file_format` is required and must be a string version. SDK 1.43.0 validates it
per the configuration spec: unsupported major versions are rejected; newer minor
versions with the same major version are accepted with a warning. Use `"1.0"`
unless you have checked the currently vendored schema and SDK loader.

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
by `load_config_file` before the YAML is further processed.

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

- **`LoggingHandler` import.** The deprecated
  `opentelemetry.sdk._logs.LoggingHandler` (deprecated in 1.40.0/0.61b0) should be
  replaced with `opentelemetry.instrumentation.logging.handler.LoggingHandler`
  (from `opentelemetry-instrumentation-logging`).

- **Env substitution** is handled by `load_config_file`; see `otel-declarative-config`
  for the substitution syntax reference.
