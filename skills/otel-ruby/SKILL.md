---
name: otel-ruby
description: OpenTelemetry in Ruby — SDK setup, manual tracing, experimental metrics and logs, Bundler integration, contrib instrumentation for Rails/Rack/Sinatra/Sidekiq and other gems, propagation, performance, and upgrades. Use when adding, reviewing, configuring, or troubleshooting OpenTelemetry in a Ruby or Rails service. Triggers on "setup otel in ruby", "ruby telemetry", "Rails OpenTelemetry", "OpenTelemetry::SDK.configure", "opentelemetry-instrumentation-all", "Ruby tracing", "Rack otel", "Sidekiq otel", or any Ruby-related OTel question.
---

# OpenTelemetry in Ruby

Entry point for OpenTelemetry mechanics in Ruby applications. Load only the reference that matches
the task; each reference is self-contained.

## References

| File | Use when |
|---|---|
| [`references/setup.md`](references/setup.md) | Installing gems, configuring `OpenTelemetry::SDK.configure`, selecting OTLP exporters, setting resources and environment variables, and handling process shutdown. |
| [`references/api.md`](references/api.md) | Writing manual traces, metrics, or logs; managing context and propagation; testing emitted telemetry. |
| [`references/instrumentation-libraries.md`](references/instrumentation-libraries.md) | Selecting and configuring contrib instrumentation for Rails, Rack, Sinatra, HTTP clients, databases, jobs, messaging, and other gems. |
| [`references/performance.md`](references/performance.md) | Tuning sampling, batching, cardinality, forked-process behavior, exporters, and lifecycle handling. |
| [`references/breaking-changes.md`](references/breaking-changes.md) | Reviewing Ruby/runtime compatibility, independently versioned gems, experimental signal changes, and core/contrib upgrades. |

## Signal stability matters

Tracing uses the stable `opentelemetry-api` and `opentelemetry-sdk` gems. Ruby metrics and logs are
separate `0.x` API, SDK, and exporter gems and remain experimental. Do not assume a tracing gem's
version exists for metrics, logs, semantic conventions, exporters, or contrib instrumentation.
Resolve each gem independently with Bundler, inspect its changelog, and keep the resulting
`Gemfile.lock` change under review.

Released snapshot verified 2026-09-09: tracing API `1.11.0`, tracing SDK `1.13.0`, trace OTLP
exporter `0.35.1`; metrics API `0.7.0`, SDK `0.17.0`, OTLP exporter `0.11.0`; logs API `0.4.1`, SDK
`0.6.1`, OTLP exporter `0.5.1`. Treat this as an audit anchor, not a version-alignment rule; use the
lookups below for newer releases.

## Sources of truth

| Fact | Fetch |
|---|---|
| Latest core releases and per-gem tags | `gh api repos/open-telemetry/opentelemetry-ruby/releases --paginate` |
| Latest contrib instrumentation releases | `gh api repos/open-telemetry/opentelemetry-ruby-contrib/releases --paginate` |
| Released and locked gem dependencies | `gem specification <gem> dependencies --remote --yaml`; inspect `Gemfile.lock` for the locked dependency set |
| Core implementation and changelogs | [open-telemetry/opentelemetry-ruby](https://github.com/open-telemetry/opentelemetry-ruby) |
| Contrib catalog, compatibility, and per-gem options | [open-telemetry/opentelemetry-ruby-contrib](https://github.com/open-telemetry/opentelemetry-ruby-contrib) |
| Ruby language docs | [OpenTelemetry Ruby documentation](https://opentelemetry.io/docs/languages/ruby/) |

## Cross-references

- SDK version selection across languages: `otel-sdk-versions` skill.
- Semantic convention names and stability: `otel-semantic-conventions` skill.
- OTLP receiver/export validation: `otel-collector` and `otel-telemetrygen` skills.
