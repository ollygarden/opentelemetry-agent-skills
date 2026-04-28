---
name: declarative-config
description: OpenTelemetry declarative YAML configuration for SDK setup. Use when configuring OpenTelemetry SDK providers (tracer, meter, logger), setting up OTLP exporters, defining sampling strategies, or writing otel config files. Triggers on "otel config", "OpenTelemetry YAML", "declarative configuration", "otelconf", "OTEL_CONFIG_FILE", "file_format", "configure tracing/metrics/logs export", or when the user is setting up telemetry pipelines via config files rather than code.
---

# OpenTelemetry Declarative Configuration

Declarative configuration replaces scattered `OTEL_*` environment variables and language-specific
programmatic SDK setup with a single YAML file. One file configures all SDK components: tracer
provider, meter provider, logger provider, propagators, and resource.

This is the preferred approach for new projects because:
- It's language-agnostic (same YAML works across Go, Java, JS, etc.)
- It's version-controlled alongside application code
- It expresses things env vars cannot: views, composite samplers, multiple exporters
- It supports `${VAR}` substitution for secrets and environment-specific values

## When to recommend declarative config

Recommend it when the user is setting up OTel SDK for Go, Java, or JS. These SDKs have
stable or near-stable implementations. For .NET and Python, fall back to environment variables
or programmatic setup as declarative config is still in development.

## Activation

The standard environment variable is `OTEL_CONFIG_FILE`:

```bash
export OTEL_CONFIG_FILE=/app/configs/otel.yaml
```

When set, the SDK reads this file at startup. All other `OTEL_*` env vars are ignored except
those referenced via `${env:VAR}` substitution inside the config file.

Language-specific activation varies — see the language sdk-setup skills for details.

## YAML Structure Overview

```yaml
file_format: "1.0"              # Schema version. Must match what the SDK supports.
disabled: false                  # Disable SDK entirely (default: false)

resource:                        # Service identity for all signals
  attributes:
    - name: service.name
      value: "${SERVICE_NAME}"
    - name: deployment.environment.name
      value: "${DEPLOY_ENV:-production}"

propagator:                      # Context propagation format
  composite: [tracecontext, baggage]

tracer_provider:                 # Trace signal: sampling, processing, export
  sampler:
    parent_based:
      root:
        trace_id_ratio_based:
          ratio: ${SAMPLE_RATE:-0.1}
  processors:
    - batch:
        exporter:
          otlp:
            protocol: grpc
            endpoint: "${OTEL_ENDPOINT}"
            headers:
              api-key: "${API_KEY}"
            compression: gzip

meter_provider:                  # Metrics signal: readers, views, export
  readers:
    - periodic:
        interval: 60000
        exporter:
          otlp:
            protocol: grpc
            endpoint: "${OTEL_ENDPOINT}"

logger_provider:                 # Logs signal: processing, export
  processors:
    - batch:
        exporter:
          otlp:
            protocol: grpc
            endpoint: "${OTEL_ENDPOINT}"
```

For the complete field-by-field schema reference, read `references/schema-reference.md`.

## Environment Variable Substitution

| Syntax | Behavior |
|--------|----------|
| `${VAR}` | Substitute with value of `VAR` |
| `${env:VAR}` | Same as `${VAR}` (explicit prefix) |
| `${VAR:-default}` | Use `default` if `VAR` is unset or empty |
| `$$` | Escape sequence, resolves to literal `$` |

Rules:
- Substitution applies only to scalar values, not mapping keys
- Type coercion happens after substitution (`${BOOL}` where `BOOL=true` becomes boolean)
- No recursive substitution
- Invalid references produce a parse error

## Configuration Precedence

```
Programmatic API  >  Environment Variables  >  Configuration File
   (highest)                                      (lowest)
```

## Common Patterns

### One config file, vary with env vars

```yaml
resource:
  attributes:
    - name: deployment.environment.name
      value: "${DEPLOY_ENV:-development}"
tracer_provider:
  sampler:
    parent_based:
      root:
        trace_id_ratio_based:
          ratio: ${SAMPLE_RATE:-1.0}  # 100% in dev, override in prod
```

### Secrets via env var substitution

```yaml
headers:
  api-key: "${API_KEY}"
endpoint: "${OTEL_ENDPOINT}"
```

## Anti-Patterns

### Missing `parent_based` wrapper

```yaml
# BAD: ignores upstream sampling decisions, breaks distributed traces
tracer_provider:
  sampler:
    trace_id_ratio_based:
      ratio: 0.1

# GOOD: respects parent sampling, applies ratio only to root spans
tracer_provider:
  sampler:
    parent_based:
      root:
        trace_id_ratio_based:
          ratio: 0.1
```

### Using `simple` processor in production

```yaml
# BAD: exports synchronously, blocks the application
tracer_provider:
  processors:
    - simple:
        exporter:
          otlp: { ... }

# GOOD: exports asynchronously in batches
tracer_provider:
  processors:
    - batch:
        exporter:
          otlp: { ... }
```

### Hardcoded secrets

```yaml
# BAD: secrets in version control
headers:
  api-key: "sk-1234567890abcdef"
```

### Mixing env vars and config file

```bash
# BAD: OTEL_TRACES_SAMPLER is ignored when OTEL_CONFIG_FILE is set
export OTEL_CONFIG_FILE="/app/otel.yaml"
export OTEL_TRACES_SAMPLER="always_off"  # This has NO effect
```

## Schema Version Compatibility

The `file_format` value must match what the SDK version supports:

| SDK | Supported file_format |
|-----|----------------------|
| Java agent 2.26.0+ | `"1.0"` |
| Go otelconf v0.3.0 | `"0.3"` |
| JS @opentelemetry/configuration (experimental) | `"1.0-rc.3"` |

Always check your SDK version's supported schema before writing the config file.

## Cross-References

- **Full schema reference**: `references/schema-reference.md` in this skill