---
name: js-sdk-setup-declarative-config
description: Set up OpenTelemetry SDK in JavaScript/Node.js applications using declarative YAML configuration. Use when initializing tracing, metrics, or logging in a Node.js service, adding OpenTelemetry to a JS project, or configuring OTel providers in Node.js. Triggers on "setup otel in node", "js telemetry", "node tracing setup", "NodeSDK otel", "express observability", "TracerProvider node", or when working on a Node.js/TypeScript project that needs observability.
---

# JavaScript/Node.js SDK Setup with Declarative Configuration

Set up OpenTelemetry in Node.js using declarative YAML configuration. This approach uses
the `@opentelemetry/configuration` package to load SDK settings from a YAML file instead of
programmatic construction or scattered environment variables.

For the YAML configuration schema, read the `general/declarative-config` skill.

> **Status**: The `@opentelemetry/configuration` package is experimental. The API may change
> between releases. If stability is critical, use the programmatic NodeSDK fallback below.

## Setup with Declarative Config

### Dependencies

```bash
npm install @opentelemetry/api @opentelemetry/sdk-node @opentelemetry/configuration
npm install @opentelemetry/auto-instrumentations-node
npm install @opentelemetry/semantic-conventions
```

### Activation

Set the `OTEL_CONFIG_FILE` environment variable pointing to a YAML config file:

```bash
export OTEL_CONFIG_FILE=configs/otel.yaml
node --import ./dist/instrumentation.js ./dist/index.js
```

When `OTEL_CONFIG_FILE` is set, the `@opentelemetry/configuration` package reads the file
at startup and configures all providers. The `createConfigFactory()` function auto-detects
the config source: if a valid YAML file exists at the `OTEL_CONFIG_FILE` path, it uses file
config; otherwise it falls back to environment variables.

### YAML Config

```yaml
file_format: "1.0-rc.3"
resource:
  attributes:
    - name: service.name
      value: "${SERVICE_NAME:-my-node-service}"
    - name: service.version
      value: "${SERVICE_VERSION:-0.1.0}"
    - name: deployment.environment.name
      value: "${NODE_ENV:-development}"

propagator:
  composite: [tracecontext, baggage]

tracer_provider:
  sampler:
    parent_based:
      root:
        trace_id_ratio_based:
          ratio: ${SAMPLE_RATE:-1.0}
  processors:
    - batch:
        exporter:
          otlp:
            protocol: http/protobuf
            endpoint: "${OTEL_ENDPOINT:-http://localhost:4318}"

meter_provider:
  readers:
    - periodic:
        interval: 30000
        exporter:
          otlp:
            protocol: http/protobuf
            endpoint: "${OTEL_ENDPOINT:-http://localhost:4318}"

logger_provider:
  processors:
    - batch:
        exporter:
          otlp:
            protocol: http/protobuf
            endpoint: "${OTEL_ENDPOINT:-http://localhost:4318}"
```

The config file must have a `.yaml` or `.yml` extension.

### Project Structure

```
src/
├── telemetry/
│   ├── constants.ts      # Service scope and telemetry constants
│   ├── setup.ts          # SDK initialization
│   └── index.ts          # Re-exports
├── index.ts              # App entry point (imports telemetry first)
configs/
└── otel.yaml             # Declarative configuration
```

### Instrumentation File (`src/telemetry/setup.ts`)

```typescript
import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';

// When OTEL_CONFIG_FILE is set, NodeSDK reads config from that file.
// Exporters, processors, samplers, and resource are all configured via YAML.
export const sdk = new NodeSDK({
  instrumentations: [
    getNodeAutoInstrumentations({
      '@opentelemetry/instrumentation-fs': { enabled: false },
      '@opentelemetry/instrumentation-dns': { enabled: false },
      '@opentelemetry/instrumentation-net': { enabled: false },
    }),
  ],
});
```

### Entry Point (`src/index.ts`)

```typescript
// IMPORTANT: Import and start telemetry BEFORE any other imports
import { sdk } from './telemetry/setup';
sdk.start();

// Now import the rest of the application
import { app } from './app';

const PORT = process.env.PORT ?? 3000;
const server = app.listen(PORT, () => {
  console.log(`Server listening on port ${PORT}`);
});

// Graceful shutdown
process.on('SIGTERM', () => {
  server.close(async () => {
    await sdk.shutdown();
    process.exit(0);
  });
});
```

### Using Tracers and Meters

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

## Fallback: Programmatic NodeSDK Setup

If declarative config is not suitable (e.g., need dynamic runtime config or older SDK version),
use programmatic setup:

```typescript
import { NodeSDK } from '@opentelemetry/sdk-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { OTLPMetricExporter } from '@opentelemetry/exporter-metrics-otlp-http';
import { OTLPLogExporter } from '@opentelemetry/exporter-logs-otlp-http';
import { PeriodicExportingMetricReader } from '@opentelemetry/sdk-metrics';
import { BatchLogRecordProcessor } from '@opentelemetry/sdk-logs';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { resourceFromAttributes } from '@opentelemetry/resources';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';

const resource = resourceFromAttributes({
  [ATTR_SERVICE_NAME]: 'my-service',
  [ATTR_SERVICE_VERSION]: process.env.SERVICE_VERSION ?? '0.0.0',
});

export const sdk = new NodeSDK({
  resource,
  traceExporter: new OTLPTraceExporter(),
  metricReader: new PeriodicExportingMetricReader({
    exporter: new OTLPMetricExporter(),
    exportIntervalMillis: 30_000,
  }),
  logRecordProcessors: [
    new BatchLogRecordProcessor(new OTLPLogExporter()),
  ],
  instrumentations: [
    getNodeAutoInstrumentations({
      '@opentelemetry/instrumentation-fs': { enabled: false },
      '@opentelemetry/instrumentation-dns': { enabled: false },
      '@opentelemetry/instrumentation-net': { enabled: false },
    }),
  ],
});
```

## Key Details

- **Import order matters**: Telemetry setup must be imported and started before any other application imports. Auto-instrumentation patches modules at require/import time.
- **`@opentelemetry/sdk-node` is experimental** (v0.200.0+) but is the recommended way to set up OTel in Node.js. It handles context manager, propagator, and provider registration automatically.
- **v2.0 migration**: `Resource` class is no longer exported. Use `resourceFromAttributes()`, `defaultResource()`, `emptyResource()` instead.
- **Disable noisy instrumentations**: `fs` and `dns` instrumentations generate high volumes of low-value spans. Disable them unless specifically needed.
- **SIGTERM handler**: Always register a shutdown handler to flush buffered telemetry before the process exits.
