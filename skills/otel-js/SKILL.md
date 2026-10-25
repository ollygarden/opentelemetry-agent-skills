---
name: otel-js
description: OpenTelemetry in Node.js / JavaScript / TypeScript — NodeSDK, declarative YAML configuration, auto-instrumentations, ESM vs CJS import patterns. Use when adding, reviewing, or configuring OpenTelemetry in a Node.js service. Triggers on "setup otel in node", "js telemetry", "node tracing setup", "NodeSDK", "auto instrumentation node", "TracerProvider node", or any Node.js-related OTel question.
---

# OpenTelemetry in Node.js

Entry point for OpenTelemetry mechanics in Node.js / JavaScript / TypeScript services.
Load a reference below based on the task; each reference is self-contained.

## References

| File | Use when |
|---|---|
| [`references/declarative-setup.md`](references/declarative-setup.md) | Configuring the SDK via declarative YAML: `OTEL_CONFIG_FILE`, `@opentelemetry/configuration`, `@opentelemetry/sdk-node`, ESM/CJS import order, 2.x and 3.0 migration notes. |

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config` skill.
For Node.js-specific facts:

| Fact | Fetch |
|---|---|
| Latest `@opentelemetry/configuration` | `npm view @opentelemetry/configuration version` |
| Latest `@opentelemetry/sdk-node` | `npm view @opentelemetry/sdk-node version` |
| Latest `@opentelemetry/auto-instrumentations-node` | `npm view @opentelemetry/auto-instrumentations-node version` |
| Released package status / breaking changes (`3.0.0` / `0.300.0`, verified 2026-10-25) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v3.0.0/experimental/packages/configuration/README.md` and `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v3.0.0/experimental/packages/opentelemetry-sdk-node/README.md` |
| Experimental CHANGELOG through `0.300.0` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v3.0.0/experimental/CHANGELOG.md` |
| ESM / CJS preload mechanics at `3.0.0` | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v3.0.0/doc/esm-support.md` |
| Auto-instrumentations register entry point at `0.82.0` (first release on SDK 3.x / API 1.10) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js-contrib/auto-instrumentations-node-v0.82.0/packages/auto-instrumentations-node/README.md` |
| 2.x → 3.x migration (removed packages, Node.js floor) | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-js/v3.0.0/doc/3.x/migration-guide.md` |
| Node.js getting-started docs | `WebFetch https://opentelemetry.io/docs/languages/js/getting-started/nodejs/` |

## Cross-References

- Schema-level facts: `otel-declarative-config` skill (language-agnostic YAML schema sources).
- SDK version selection across languages: `otel-sdk-versions` skill.
- Semantic conventions lookup: `otel-semantic-conventions` skill.
