---
name: otel-declarative-config
description: OpenTelemetry declarative YAML configuration for SDK setup. Use when configuring OpenTelemetry SDK providers (tracer, meter, logger), setting up OTLP exporters, defining sampling strategies, or writing otel config files. Triggers on "otel config", "OpenTelemetry YAML", "declarative configuration", "otelconf", "OTEL_CONFIG_FILE", "file_format", "configure tracing/metrics/logs export", or when the user is setting up telemetry pipelines via config files rather than code.
---

# OpenTelemetry Declarative Configuration

Declarative configuration replaces scattered `OTEL_*` environment variables and language-specific
programmatic SDK setup with a single YAML file. One file configures all SDK components: tracer
provider, meter provider, logger provider, propagators, and resource.

For the current per-language SDK status, fetch the SDK compatibility matrix (see Sources of Truth).
Use it to understand implementation coverage, not as the only source for YAML literals.

## Sources of Truth

This skill teaches concepts. The schema itself, valid `file_format` strings, field names,
and SDK compatibility evolve per release — fetch from upstream rather than relying on
embedded copies. Cache results for the conversation; refetch only on schema-related errors.

| Fact | Fetch |
|---|---|
| Latest schema release tag | `gh release view --repo open-telemetry/opentelemetry-configuration --json tagName,publishedAt` |
| SDK ↔ schema compatibility matrix (coverage advisory, not authoritative for literal `file_format`) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/main/language-support-status.md` |
| Field-by-field docs for the latest release | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/<schema-release-tag>/schema-docs.md` |
| Compiled JSON Schema (validate generated YAML against this) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/<schema-release-tag>/opentelemetry_configuration.json` |
| Canonical full example | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/<schema-release-tag>/examples/otel-sdk-config.yaml` |
| Migration template (every option, with comments) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/<schema-release-tag>/examples/otel-sdk-migration-config.yaml` |
| Schema CHANGELOG (breaking-change history with migration steps) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-configuration/<schema-release-tag>/CHANGELOG.md` |

**Workflow when generating YAML:**

1. Identify the exact runtime/package/agent version that will parse the file.
2. Fetch that runtime/package source, docs, or test fixtures and confirm the accepted
   `file_format` literal. If this conflicts with `language-support-status.md`, the
   runtime/package wins.
3. Fetch `examples/otel-sdk-config.yaml` → use as the structural template only after
   adapting the `file_format` and fields to the selected runtime/package.
4. Overlay the user's specific values (service name, endpoint, sampling, headers).
5. If validation matters, fetch `opentelemetry_configuration.json` and validate the result,
   then still verify against the selected runtime/package because SDK implementations may
   lag or differ from the schema repository.

Replace `<schema-release-tag>` with the tag returned by the first fetch. Keep the compatibility
matrix on `main`: it tracks language implementation coverage independently of schema releases
and may include work not yet released by an implementation. Do not use schema files or examples
from `main` to generate released-version guidance.

The "Latest supported file format" values in `language-support-status.md` are schema/version
coverage metadata. Do not mechanically copy them into YAML unless the target SDK parser,
agent docs, or package fixtures prove that exact literal is accepted.

Terminology trap: schema coverage identifiers and YAML `file_format` literals are related,
but not interchangeable. A matrix entry uses a full semver-shaped coverage value (e.g.
`1.0.0`, or a pre-release such as `1.0.0-rc.3` for implementations still tracking an older
schema), while stable schema examples use a `MAJOR.MINOR` string such as `1.0` or `1.1`.
Older implementations may accept or require a pre-release literal, and released parsers do
not all enforce versions identically. Generated YAML must use the literal verified in the
target runtime. The current stable schema is v1.1.0 (`file_format: "1.1"`); confirm the exact
release with the `gh release` fetch above.

For language-specific package versions and SDK API surface, see the Sources of Truth section
in each language's `otel-<lang>` skill (`otel-go`, `otel-java`, `otel-js`, `otel-python`).
`otel-dotnet` is listed in Cross-References below but does **not** support declarative YAML config yet — see the .NET note.

**Python note:** declarative config requires `opentelemetry-sdk` 1.43.0 or newer
with the `file-configuration` extra (`opentelemetry-sdk[file-configuration]`).
Released Python SDKs honor `OTEL_CONFIG_FILE` through the SDK configurator; when
set, the file is authoritative and the env-var initialization path is skipped.
Programmatic loading is also available. See the `otel-python` skill and its
`declarative-setup.md` reference.

**\.NET note:** declarative YAML config is **not yet implemented** in OpenTelemetry .NET
(tracked by [`open-telemetry/opentelemetry-dotnet#6380`](https://github.com/open-telemetry/opentelemetry-dotnet/issues/6380)).
.NET configures via the DI/builder API, `OTEL_*` env vars, and `IConfiguration`. Do **not**
use `OTEL_CONFIG_FILE` with .NET runtimes. See the `otel-dotnet` skill and its `setup.md` reference.

## Activation

The standard environment variable is `OTEL_CONFIG_FILE`:

```bash
export OTEL_CONFIG_FILE=/app/configs/otel.yaml
```

When set, the SDK reads this file at startup. In file-config mode, SDKs ignore `OTEL_*`
environment variables except those referenced through environment-variable substitution
inside the config file.

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

## Configuration Interaction

Declarative configuration is a file-config mode, not a lower-precedence layer under
`OTEL_*` environment variables:

| Mode | Behavior |
|------|----------|
| File config (`OTEL_CONFIG_FILE`, `-Dotel.config.file`, or programmatic loader) | The file supplies SDK configuration. `OTEL_*` env vars are ignored unless the file explicitly references them with substitution such as `${OTEL_SERVICE_NAME}`. |
| Env-var config | Applies only when the selected SDK/runtime is not parsing a declarative config file. |
| Programmatic setup | Application code can still decide whether to load a file, override parsed values, or build providers directly. Treat the programmatic code path as the runtime source of truth. |

## Cross-References

- Language-specific setup: `otel-go`, `otel-java`, `otel-js`, `otel-python` (each loads its own `references/declarative-setup.md`); `otel-dotnet` (declarative YAML config not yet supported — see .NET note above).
