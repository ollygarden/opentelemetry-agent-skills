---
name: otel-declarative-config
description: OpenTelemetry declarative YAML configuration for SDK setup. Use when configuring OpenTelemetry SDK providers (tracer, meter, logger), setting up OTLP exporters, defining sampling strategies, or writing otel config files. Triggers on "otel config", "OpenTelemetry YAML", "declarative configuration", "otelconf", "OTEL_CONFIG_FILE", "file_format", "configure tracing/metrics/logs export", or when the user is setting up telemetry pipelines via config files rather than code.
---

# OpenTelemetry Declarative Configuration

Declarative configuration replaces scattered `OTEL_*` environment variables and language-specific
programmatic SDK setup with a single YAML file. One file configures all SDK components: tracer
provider, meter provider, logger provider, propagators, and resource.

For the current per-language SDK status, fetch the SDK compatibility matrix (see Sources of Truth).

## Sources of Truth

This skill teaches concepts. The schema itself, valid `file_format` strings, field names,
and SDK compatibility evolve per release — fetch from upstream rather than relying on
embedded copies. Cache results for the conversation; refetch only on schema-related errors.

| Fact | Fetch |
|---|---|
| Latest schema release tag | `gh release view --repo open-telemetry/opentelemetry-configuration --json tagName,publishedAt` |
| SDK ↔ schema compatibility matrix | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/main/language-support-status.md` |
| Field-by-field schema docs | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/main/schema-docs.md` |
| Compiled JSON Schema (validate generated YAML against this) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/main/opentelemetry_configuration.json` |
| Canonical full example | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/main/examples/otel-sdk-config.yaml` |
| Migration template (every option, with comments) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/main/examples/otel-sdk-migration-config.yaml` |
| Schema CHANGELOG (breaking-change history with migration steps) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/main/CHANGELOG.md` |

**Workflow when generating YAML:**

1. Fetch `language-support-status.md` → pick the `file_format` string for the target SDK version.
2. Fetch `examples/otel-sdk-config.yaml` → use as the structural template.
3. Overlay the user's specific values (service name, endpoint, sampling, headers).
4. If validation matters, fetch `opentelemetry_configuration.json` and validate the result.

For language-specific package versions and SDK API surface, see the Sources of Truth section
in each language's `otel-<lang>` skill (`otel-go`, `otel-java`, `otel-js`).

## Activation

The standard environment variable is `OTEL_CONFIG_FILE`:

```bash
export OTEL_CONFIG_FILE=/app/configs/otel.yaml
```

When set, the SDK reads this file at startup. All other `OTEL_*` env vars are ignored except
those referenced via `${env:VAR}` substitution inside the config file.

Language-specific activation varies — see the language `sdk-setup` skills for details.

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

## Cross-References

- Language-specific setup: `otel-go`, `otel-java`, `otel-js` (each loads its own `references/declarative-setup.md`).
