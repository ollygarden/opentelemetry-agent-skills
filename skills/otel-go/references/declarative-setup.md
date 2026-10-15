# Go SDK Setup with otelconf

Configure the OpenTelemetry SDK in Go via `otelconf`, the declarative YAML configuration
package.

For the YAML configuration schema, load the `otel-declarative-config` skill.

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config`
skill. For Go-specific facts:

| Fact | Fetch |
|---|---|
| `otelconf` CHANGELOG (which schema rc each release supports, breaking changes) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-contrib/main/CHANGELOG.md` |
| Current API surface (`SDK` methods, options) | `WebFetch https://pkg.go.dev/go.opentelemetry.io/contrib/otelconf` |
| Schema tag the generated types come from (`genjsonschema` target) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-contrib/otelconf/<tag>/Makefile` |
| Latest `go.opentelemetry.io/otel` core release | `gh api repos/open-telemetry/opentelemetry-go/releases/latest -q '.tag_name'` |

## Choosing an Import Path

The `otelconf` repo ships two flavors of the package:

- **Root** `go.opentelemetry.io/contrib/otelconf` — trails the schema repository. Verified
  2026-10-15: `v0.27.0` targets schema `v1.0.0-rc.3` (`file_format: "1.0-rc.3"`) while the schema
  repository is at `v1.2.0` (owner: the `otel-declarative-config`
  [parsers table](../../otel-declarative-config/SKILL.md)). Includes `sdk.Propagator()` for installing
  propagators from YAML.
- **Schema-pinned** `go.opentelemetry.io/contrib/otelconf/v0.3.0` — frozen at file_format
  `"0.3"` (a 2024 schema). Has no `Propagator()` method; propagators must be installed
  manually via `otel.SetTextMapPropagator(...)`.

Newer `otelconf` releases may exist; check the selected tag's Makefile schema tag before deploying.
For "latest", resolve the tag once with the `otelconf` tag command in this skill's `SKILL.md`.

## Migration notes

- `go.opentelemetry.io/contrib/config` was removed in contrib v1.35.0. Use `go.opentelemetry.io/contrib/otelconf`.
- In the root package, use `OTEL_CONFIG_FILE`; `OTEL_EXPERIMENTAL_CONFIG_FILE` is rejected by
  current released `otelconf`.
- Starting with root `otelconf` v0.25.0 (contrib v1.45.0), the resolved resource includes the SDK's default resource attributes; account for those when comparing telemetry across this boundary.
- If migrating from the schema-pinned `otelconf/v0.3.0` import, the YAML must also migrate to
  the new schema (fetch `examples/otel-sdk-migration-config.yaml` from the schema repo for
  before/after pairs).

## Dependencies

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
    otelconf "go.opentelemetry.io/contrib/otelconf"
)
```

## YAML Config

For the canonical structure, fetch `examples/otel-sdk-config.yaml` from the schema repo
(see the `otel-declarative-config` skill's Sources of Truth). The minimal example below
illustrates the Go-specific quirk; the exact `file_format` string and exporter field names
should come from the selected `otelconf` module's package fixtures/source.

```yaml
# file_format: use the exact literal accepted by the selected otelconf module
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-my-service}"
    - name: deployment.environment.name
      value: "${DEPLOY_ENV:-development}"

# Tracer/meter/logger provider blocks: structure per the canonical example.
# The root otelconf parser resolves ${VAR}, ${env:VAR}, and ${VAR:-default}
# before parsing YAML. Use $${VAR} when the literal text must be preserved.
```

## Key API Facts

- **`otelconf.ParseYAML([]byte) (*OpenTelemetryConfiguration, error)`** parses a YAML config file.
- **`otelconf.NewSDK(opts ...) (SDK, error)`** builds the SDK from a parsed configuration. Options include `WithContext(ctx)` and `WithOpenTelemetryConfiguration(conf OpenTelemetryConfiguration)`; after `ParseYAML`, pass `*conf` because `ParseYAML` returns `*OpenTelemetryConfiguration`.
- **`sdk.TracerProvider()`, `sdk.MeterProvider()`, `sdk.LoggerProvider()`** return the configured providers.
- **`sdk.Propagator()`** (root package, `otelconf v0.20.0+`, fixes [open-telemetry/opentelemetry-go-contrib#6712](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712)) returns the propagator built from the YAML `propagator:` block. Install it via `otel.SetTextMapPropagator(sdk.Propagator())`. Not available on the schema-pinned `otelconf/v0.3.0` subpackage.
- **`sdk.Shutdown(ctx) error`** flushes and closes all providers.
