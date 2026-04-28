# JavaScript/Node.js SDK Setup with Declarative Configuration

Configure the OpenTelemetry SDK in Node.js via declarative YAML. The
`@opentelemetry/configuration` package loads SDK settings from a YAML file at startup.

For the YAML configuration schema, load the `otel-declarative-config` skill.

> **Status**: The `@opentelemetry/configuration` package is experimental. The API may change
> between releases.

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config`
skill. For JS-specific facts:

| Fact | Fetch |
|---|---|
| Latest `@opentelemetry/configuration` | `npm view @opentelemetry/configuration version` |
| Latest `@opentelemetry/sdk-node` | `npm view @opentelemetry/sdk-node version` |
| Latest `@opentelemetry/auto-instrumentations-node` | `npm view @opentelemetry/auto-instrumentations-node version` |
| Package status / breaking changes | `WebFetch https://www.npmjs.com/package/@opentelemetry/configuration` |
| `sdk-node` CHANGELOG | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/main/experimental/packages/opentelemetry-sdk-node/CHANGELOG.md` |
| Node.js getting-started docs | `WebFetch https://opentelemetry.io/docs/languages/js/getting-started/nodejs/` |

## Activation

Set the `OTEL_CONFIG_FILE` environment variable pointing to a YAML config file:

```bash
export OTEL_CONFIG_FILE=configs/otel.yaml
node --import ./dist/instrumentation.js ./dist/index.js
```

When `OTEL_CONFIG_FILE` is set, the `@opentelemetry/configuration` package reads the file
at startup and configures all providers. The `createConfigFactory()` function auto-detects
the config source: if a valid YAML file exists at the `OTEL_CONFIG_FILE` path, it uses file
config; otherwise it falls back to environment variables.

The config file must have a `.yaml` or `.yml` extension.

## YAML Config

For the canonical structure and the correct `file_format` string for your `@opentelemetry/configuration`
version, fetch `examples/otel-sdk-config.yaml` and `language-support-status.md` (see the
`otel-declarative-config` skill's Sources of Truth). The minimal example below illustrates
the JS-specific quirk.

```yaml
# file_format: pick from language-support-status.md based on your @opentelemetry/configuration version
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-my-node-service}"
    - name: service.version
      value: "${SERVICE_VERSION:-0.1.0}"
    - name: deployment.environment.name
      value: "${NODE_ENV:-development}"

# Tracer/meter/logger provider blocks: structure per the canonical example.
# JS-specific quirk: the config file must have a .yaml or .yml extension.
```

## Using Tracers and Meters

```typescript
import { trace, metrics } from '@opentelemetry/api';

const SCOPE = 'mycompany.com/myservice';
const tracer = trace.getTracer(SCOPE);
const meter = metrics.getMeter(SCOPE);
```

## ESM vs CommonJS

For ESM projects, use `--import` to ensure instrumentation loads before application code:

```bash
# ESM (Node.js >=18.19.0)
OTEL_CONFIG_FILE=configs/otel.yaml node --import ./dist/instrumentation.js ./dist/index.js

# CommonJS
OTEL_CONFIG_FILE=configs/otel.yaml node --require ./dist/instrumentation.js ./dist/index.js
```

## Key API Facts

- **Import order matters**: Telemetry setup must be imported and started before any other application imports. Auto-instrumentation patches modules at require/import time.
- **`@opentelemetry/sdk-node` is experimental** but is the recommended way to set up OTel in Node.js. It handles context manager, propagator, and provider registration automatically.
- **v2.0 migration**: `Resource` class is no longer exported. Use `resourceFromAttributes()`, `defaultResource()`, `emptyResource()` instead.
