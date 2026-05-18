---
name: otel-go
description: OpenTelemetry in Go — SDK setup, API surface, breaking changes, contrib instrumentation libraries (otelhttp, otelgrpc, otelmongo), and performance tuning. Use when adding, reviewing, or configuring OpenTelemetry in a Go service. Triggers on "setup otel in go", "go telemetry", "go tracing", "otelconf go", "otelhttp", "otelgrpc", "TracerProvider go", "MeterProvider go", or any Go-related OTel question.
---

# OpenTelemetry in Go

Entry point for OpenTelemetry mechanics in Go services. Load a reference below based on the
task; each reference is self-contained.

## References

| File | Use when |
|---|---|
| [`references/declarative-setup.md`](references/declarative-setup.md) | Configuring the SDK via `otelconf` and YAML: providers, propagators, shutdown, env-var substitution. |
| [`references/api.md`](references/api.md) | Looking up import paths, global API access, tracer/meter/logger usage, attributes, propagation, log bridges (zap, slog). |
| [`references/instrumentation-libraries.md`](references/instrumentation-libraries.md) | Picking or wiring contrib libraries (otelhttp, otelgrpc, database, AWS, message queues, propagators, resource detectors), and writing manual instrumentation that follows semconv. |
| [`references/performance.md`](references/performance.md) | Tuning sampling, batch processor, metric reader, exporter compression/retry, attribute allocation, log `Enabled()` short-circuiting, graceful shutdown. |
| [`references/breaking-changes.md`](references/breaking-changes.md) | Auditing existing code for deprecated calls, renamed semantic conventions, and removed APIs across recent SDK / contrib releases. |

## Sources of Truth

For YAML schema details, fetch the upstream sources listed in the `otel-declarative-config` skill.
For Go-specific facts:

| Fact | Fetch |
|---|---|
| Latest `go.opentelemetry.io/otel` core release | `gh api repos/open-telemetry/opentelemetry-go/releases/latest -q '.tag_name'` |
| Latest `go.opentelemetry.io/contrib` release | `gh api repos/open-telemetry/opentelemetry-go-contrib/releases/latest -q '.tag_name'` |
| Latest `otelconf` module tag | `gh api repos/open-telemetry/opentelemetry-go-contrib/git/matching-refs/tags/otelconf -q '.[-1].ref'` |
| Latest semconv package version | `gh api repos/open-telemetry/semantic-conventions/releases/latest -q '.tag_name'` |
| `otel-go` CHANGELOG | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go/main/CHANGELOG.md` |
| `otel-go-contrib` CHANGELOG | `WebFetch https://raw.githubusercontent.com/open-telemetry/opentelemetry-go-contrib/main/CHANGELOG.md` |

## Cross-References

- Schema-level facts: `otel-declarative-config` skill (language-agnostic YAML schema sources).
- SDK version selection across languages: `otel-sdk-versions` skill.
- Semantic conventions lookup: `otel-semantic-conventions` skill.
