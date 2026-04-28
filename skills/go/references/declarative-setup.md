# Go SDK Setup with otelconf

Configure the OpenTelemetry SDK in Go via `otelconf`, the declarative YAML configuration
package.

For the YAML configuration schema, load the `otel-declarative-config` skill.

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config`
skill. For Go-specific facts:

| Fact | Fetch |
|---|---|
| Latest `otelconf` module tag | `gh api repos/open-telemetry/opentelemetry-go-contrib/git/matching-refs/tags/otelconf -q '.[-1].ref'` |
| `otelconf` CHANGELOG (which schema rc each release supports, breaking changes) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-contrib/main/CHANGELOG.md` |
| Current API surface (`SDK` methods, options) | `WebFetch https://pkg.go.dev/go.opentelemetry.io/contrib/otelconf` |
| Latest `go.opentelemetry.io/otel` core release | `gh api repos/open-telemetry/opentelemetry-go/releases/latest -q '.tag_name'` |

## Choosing an Import Path

The `otelconf` repo ships two flavors of the package:

- **Root** `go.opentelemetry.io/contrib/otelconf` — tracks the current schema (file_format
  `1.0.0-rc.x` or later). Active development. Includes `sdk.Propagator()` for installing
  propagators from YAML.
- **Schema-pinned** `go.opentelemetry.io/contrib/otelconf/v0.3.0` — frozen at file_format
  `"0.3"` (a 2024 schema). Has no `Propagator()` method; propagators must be installed
  manually via `otel.SetTextMapPropagator(...)`.

Fetch the `otelconf` CHANGELOG (see Sources of Truth) to confirm which schema version the
latest module tag supports.

## Migration notes

- `go.opentelemetry.io/contrib/config` is deprecated. Use `go.opentelemetry.io/contrib/otelconf`.
- If migrating from the schema-pinned `otelconf/v0.3.0` import, the YAML must also migrate to
  the new schema (fetch `examples/otel-sdk-migration-config.yaml` from the schema repo for
  before/after pairs).

## Dependencies

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/log/global"
    "go.opentelemetry.io/otel/propagation"
    semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
    otelconf "go.opentelemetry.io/contrib/otelconf"
)
```

## YAML Config

For the canonical structure, fetch `examples/otel-sdk-config.yaml` from the schema repo
(see the `otel-declarative-config` skill's Sources of Truth). The minimal example below
illustrates the Go-specific quirk; the exact `file_format` string and exporter field names
should come from upstream.

```yaml
# file_format: pick from language-support-status.md based on your otelconf version
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-my-service}"
    - name: deployment.environment.name
      value: "${DEPLOY_ENV:-development}"

# Tracer/meter/logger provider blocks: structure per the canonical example.
# Go-specific quirk: os.ExpandEnv resolves ${VAR} substitutions before the YAML is
# parsed, so any ${VAR} in this file works when the loader uses os.ExpandEnv.
```

## Key API Facts

- **`otelconf.ParseYAML([]byte) (*OpenTelemetryConfiguration, error)`** parses a YAML config file.
- **`otelconf.NewSDK(opts ...) (SDK, error)`** builds the SDK from a parsed configuration. Options include `WithContext(ctx)` and `WithOpenTelemetryConfiguration(*conf)`.
- **`sdk.TracerProvider()`, `sdk.MeterProvider()`, `sdk.LoggerProvider()`** return the configured providers.
- **`sdk.Propagator()`** (root package, `otelconf v0.20.0+`, fixes [open-telemetry/opentelemetry-go-contrib#6712](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712)) returns the propagator built from the YAML `propagator:` block. Install it via `otel.SetTextMapPropagator(sdk.Propagator())`. Not available on the schema-pinned `otelconf/v0.3.0` subpackage.
- **`sdk.Shutdown(ctx) error`** flushes and closes all providers.
