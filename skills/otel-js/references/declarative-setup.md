# JavaScript/Node.js SDK Setup with Declarative Configuration

Configure the OpenTelemetry SDK in Node.js via declarative YAML. The
`@opentelemetry/configuration` package parses YAML or environment-derived SDK settings;
`@opentelemetry/sdk-node` applies them during startup.

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
| Released package status / breaking changes (`2.11.0` / `0.222.0`) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v2.11.0/experimental/packages/configuration/README.md` and `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v2.11.0/experimental/packages/opentelemetry-sdk-node/README.md` |
| SDK declarative startup helper at `0.222.0` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v2.11.0/experimental/packages/opentelemetry-sdk-node/src/start.ts` |
| Released file config parser (`file_format` acceptance) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v2.11.0/experimental/packages/configuration/src/FileConfigFactory.ts` |
| Released file config fixtures | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v2.11.0/experimental/packages/configuration/test/fixtures/sdk-config.yaml` |
| Experimental CHANGELOG through `0.222.0` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v2.11.0/experimental/CHANGELOG.md` |
| ESM / CJS preload mechanics at `2.11.0` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v2.11.0/doc/esm-support.md` |
| Auto-instrumentations register entry point at `0.80.0` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js-contrib/auto-instrumentations-node-v0.80.0/packages/auto-instrumentations-node/README.md` |
| Node.js getting-started docs | `WebFetch https://opentelemetry.io/docs/languages/js/getting-started/nodejs/` |

## Activation

Set `OTEL_CONFIG_FILE` to a YAML config file, then preload a telemetry startup module
before application code:

```typescript
// instrumentation.ts
import { startNodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';

startNodeSDK({
  instrumentations: [getNodeAutoInstrumentations()],
});
```

```bash
export OTEL_CONFIG_FILE=configs/otel.yaml
node --import ./dist/instrumentation.js ./dist/index.js
```

When `OTEL_CONFIG_FILE` is set, `@opentelemetry/configuration` reads that path as YAML.
`@opentelemetry/sdk-node`'s experimental `startNodeSDK()` helper uses the resulting model
to configure providers. If `OTEL_CONFIG_FILE` is unset, `createConfigFactory()` falls back
to environment variables. If it is set but unreadable or invalid, `startNodeSDK()` logs a
diagnostic error and returns a no-op SDK rather than falling back to env vars.

Use `.yaml` or `.yml` filenames in examples and docs; verify extension enforcement against
the current `FileConfigFactory.ts` source before claiming the parser rejects other suffixes.

## YAML Config

For the canonical structure, fetch `examples/otel-sdk-config.yaml` (see the
`otel-declarative-config` skill's Sources of Truth), then check the installed
`@opentelemetry/configuration` package or its matching source/fixtures for runtime support.
Verified 2026-10-15: `@opentelemetry/configuration` 0.223.0 generates its config types from
schema v1.1.0, as has every release since the pinned 0.220.0 line, so write `file_format: "1.1"`. It accepts schema major `1` including `1.0`, warns
on newer minor versions such as `"1.2"`, and rejects other major versions.

```yaml
file_format: "1.1"
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-my-node-service}"
      type: string
    - name: service.version
      value: "${SERVICE_VERSION:-0.1.0}"
      type: string
    - name: deployment.environment.name
      value: "${NODE_ENV:-development}"
      type: string

# Tracer/meter/logger provider blocks: structure per the canonical example.
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

If the service relies on auto-instrumentation patching ESM imports, include the current
OpenTelemetry loader hook as described in `doc/esm-support.md`:

```bash
node --experimental-loader=@opentelemetry/instrumentation/hook.mjs --import ./dist/instrumentation.js ./dist/index.js
```

The zero-code `@opentelemetry/auto-instrumentations-node/register` entry point starts
`NodeSDK` from environment variables; use `startNodeSDK()` when the task specifically needs
declarative YAML via `OTEL_CONFIG_FILE`.

## Key API Facts

- **Import order matters**: Telemetry setup must be imported and started before any other application imports. Auto-instrumentation patches modules at require/import time.
- **`@opentelemetry/sdk-node` is experimental**. `new NodeSDK()` is its currently recommended startup mechanism; the declarative `startNodeSDK()` helper is also experimental. Both handle context manager, propagator, and provider registration automatically.
- **v2.0 migration**: `Resource` class is no longer exported. Use `resourceFromAttributes()`, `defaultResource()`, `emptyResource()` instead.
- **2.9.0 / 0.220.0 migration note**: `@opentelemetry/sdk-node` now uses `@opentelemetry/sdk-trace`; the `node` and `tracing` namespace re-exports are deprecated. Import trace SDK types/classes directly from their packages.
- **2.10.0 / 0.221.0 propagator note**: Selecting `jaeger` through `OTEL_PROPAGATORS` or declarative config emits a deprecation warning. Use `tracecontext`; the Jaeger propagator still works in this release but is planned for removal.
- **2.11.0 / 0.222.0 declarative migration note**: `startNodeSDK()` now fails fast while creating invalid resource, tracer-provider, meter-provider, or propagator configuration and returns a no-op SDK. This can also affect environment-derived configuration—for example, `OTEL_NODE_RESOURCE_DETECTORS=all` currently includes the unsupported declarative `container` detector. This does not affect `new NodeSDK()`.
