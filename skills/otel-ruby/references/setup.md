# Ruby SDK setup

## Choose gems by signal

For stable tracing, applications normally need the SDK, an exporter, and only the
instrumentation they use:

```ruby
# Gemfile
gem 'opentelemetry-sdk'
gem 'opentelemetry-exporter-otlp'
gem 'opentelemetry-instrumentation-rails'
gem 'opentelemetry-instrumentation-sidekiq'
gem 'opentelemetry-instrumentation-logger'
```

Libraries that create spans should depend only on `opentelemetry-api`. Exporters and SDK setup
belong to the application. The convenience gem `opentelemetry-instrumentation-all` enables
runtime discovery with `use_all`, but explicit instrumentation gems plus `use` make the installed
surface easier to audit.

Metrics and logs require separate experimental gems. Add only the signals the application will
emit and export:

```ruby
gem 'opentelemetry-metrics-api'
gem 'opentelemetry-metrics-sdk'
gem 'opentelemetry-exporter-otlp-metrics'

gem 'opentelemetry-logs-api'
gem 'opentelemetry-logs-sdk'
gem 'opentelemetry-exporter-otlp-logs'
```

Do not give these gems the tracing SDK's version. Let Bundler resolve compatible releases, then
review the lockfile and each changed gem's changelog.

## Configure once, before application work

Require the selected exporter and instrumentation gems before configuration. In Rails, place the
configuration in an initializer; in Rack/Sinatra or a worker, run it during bootstrap.
Use `OTEL_*` environment variables and `OpenTelemetry::SDK.configure`. The separate experimental
`opentelemetry-config` gem (first released `0.1.0`, 2026-09-29) reads an `OTEL_CONFIG_FILE` YAML
file, but only when the application calls `OpenTelemetry::Config.configure` (or
`configure_from_file`); it covers the tracer provider, resource, samplers, and propagators only, and
per its README does not yet give the file precedence over environment variables. For the YAML schema
see the `otel-declarative-config` skill; check the locked gem's README before relying on it.
Requiring the metrics and logs SDKs before
`configure` installs their configurator hooks; their default exporter is OTLP when the matching
exporter gem is loaded.

```ruby
require 'opentelemetry/sdk'
require 'opentelemetry/exporter/otlp'
require 'opentelemetry-metrics-sdk'
require 'opentelemetry/exporter/otlp_metrics'
require 'opentelemetry-logs-sdk'
require 'opentelemetry/exporter/otlp_logs'
require 'opentelemetry/instrumentation/rails'
require 'opentelemetry/instrumentation/sidekiq'
require 'opentelemetry/instrumentation/logger'

OpenTelemetry::SDK.configure do |c|
  c.service_name = 'checkout'
  c.service_version = ENV.fetch('APP_VERSION', 'unknown')
  c.use 'OpenTelemetry::Instrumentation::Rails'
  c.use 'OpenTelemetry::Instrumentation::Sidekiq'
  c.use 'OpenTelemetry::Instrumentation::Logger'
end
```

`use` and `use_all` are mutually exclusive in one SDK configuration. `use_all` installs every
registered instrumentation whose target library is present:

```ruby
OpenTelemetry::SDK.configure do |c|
  c.use_all(
    'OpenTelemetry::Instrumentation::ActiveRecord' => { enabled: false }
  )
end
```

Use each instrumentation's README for its exact constant, supported dependency versions, and
options. Do not infer a constant mechanically from a gem name.

## Environment configuration

Common inputs include:

```sh
OTEL_SERVICE_NAME=checkout
OTEL_RESOURCE_ATTRIBUTES=service.version=abc123,deployment.environment.name=production
OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4318
OTEL_TRACES_EXPORTER=otlp
OTEL_PROPAGATORS=tracecontext,baggage
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
OTEL_LOG_LEVEL=info
```

The released automatic trace and log OTLP paths support `http/protobuf`; an unsupported general
or relevant signal-specific protocol disables the trace or log exporter with a warning. The
metrics path does not read protocol variables and uses its HTTP/protobuf `MetricsExporter`.
The core repository contains an `opentelemetry-exporter-otlp-grpc` prototype, but its changelog
marks it unreleased and not production-ready. Do not recommend it until a released gem says
otherwise. OTLP compression defaults differ by signal: the trace exporter defaults to `none` since
`opentelemetry-exporter-otlp` `0.37.0`, while the metrics and logs exporters default to `gzip`; set
`OTEL_EXPORTER_OTLP_COMPRESSION` (or the signal-specific variable) when a uniform choice matters.
Use signal-specific endpoints and headers when the signals differ. Never place a
secret directly in checked-in configuration.

## Lifecycle

Flush providers during graceful process termination. `shutdown` flushes processors and exporters;
calling it too early makes later telemetry no-op or fail export.

```ruby
at_exit do
  OpenTelemetry.tracer_provider.shutdown
  OpenTelemetry.meter_provider.shutdown if defined?(OpenTelemetry::SDK::Metrics)
  OpenTelemetry.logger_provider.shutdown if defined?(OpenTelemetry::SDK::Logs)
end
```

Framework and server shutdown hooks are preferable when they offer a bounded graceful-shutdown
phase. For short jobs, call `force_flush` before the process exits if the provider remains shared.

## Verify locally

1. Run `bundle install` and inspect `Gemfile.lock` for independently resolved OTel gems.
2. Boot once with `OTEL_LOG_LEVEL=debug` and confirm no missing exporter or instrumentation warning.
3. Send a request or execute a job against a disposable local receiver.
4. Confirm the expected resource, parent/child relationship, and shutdown export.

Sources: [core SDK 1.13.1](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-sdk/v1.13.1/sdk),
[`opentelemetry-config` 0.1.0](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-config/v0.1.0/config),
[Ruby getting started](https://opentelemetry.io/docs/languages/ruby/getting-started/), and
[contrib instrumentation catalog at the `all` 0.97.0 release](https://github.com/open-telemetry/opentelemetry-ruby-contrib/tree/opentelemetry-instrumentation-all/v0.97.0/instrumentation).
