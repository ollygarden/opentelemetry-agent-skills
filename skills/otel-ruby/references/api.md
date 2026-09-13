# Ruby API, context, and tests

## Tracing

Application and library code obtains a tracer from the global API. Use a stable instrumentation
scope name (normally the gem or library name), not a request-specific value.

```ruby
require 'opentelemetry'

tracer = OpenTelemetry.tracer_provider.tracer('checkout.pricing', '1.0.0')

tracer.in_span('price.quote', kind: :internal, attributes: { 'product.id' => product_id }) do |span|
  span.add_event('discount.applied', attributes: { 'discount.code' => code })
  calculate_price
end
```

`in_span` makes the span current, records an escaping exception by default, marks the span as
error, re-raises, and finishes it. Avoid recording the same exception twice: either rely on that
behavior or pass `record_exception: false` and handle it deliberately. Use `start_span` only when the span lifetime crosses a block;
then activate its context and always finish it.

Span kinds are `:internal`, `:server`, `:client`, `:producer`, and `:consumer`. Choose attributes
from the released semantic conventions; do not put unbounded identifiers or sensitive payloads
into attributes.

## Propagation and fiber-local context

The default propagators are W3C Trace Context and Baggage. Instrumentation libraries normally
inject and extract automatically. For a custom carrier:

```ruby
headers = {}
OpenTelemetry.propagation.inject(headers)

parent = OpenTelemetry.propagation.extract(incoming_headers)
OpenTelemetry::Context.with_current(parent) do
  tracer.in_span('message.consume', kind: :consumer) { consume }
end
```

Ruby context is stored per `Fiber`. If manual code calls `OpenTelemetry::Context.attach`, pair it
with `detach(token)` in `ensure`; unbalanced calls leak the active context into later work. Verify
context behavior when an application mixes threads, fibers, async schedulers, or process forks.

## Experimental metrics

Metrics remain in separate `0.x` gems. After requiring and configuring the metrics SDK:

```ruby
meter = OpenTelemetry.meter_provider.meter('checkout.pricing', version: '1.0.0')
quotes = meter.create_counter('pricing.quotes', unit: '1')
duration = meter.create_histogram('pricing.quote.duration', unit: 's')

quotes.add(1, attributes: { 'quote.result' => 'accepted' })
duration.record(elapsed, attributes: { 'quote.result' => 'accepted' })
```

Keep metric attributes bounded. Use views to drop or allowlist attributes when third-party
instrumentation would otherwise create excessive cardinality. Check the locked metrics SDK
README for supported instruments, views, exemplars, and temporality because the API is experimental.

## Experimental logs and the Logger bridge

The Logs API/SDK are also separate `0.x` gems. Applications usually bridge an existing logging
library rather than replacing it. For Ruby's standard `Logger`, install and configure
`opentelemetry-instrumentation-logger`; consult its README for options and emitted fields.
Direct `LoggerProvider#logger(...).on_emit(...)` is intended for instrumentation libraries and
advanced integrations, not as a requirement to rewrite application logging.

## Tests

Use in-memory exporters for focused tests and clear them between examples. Assert stable contract
facts—span name, kind, parent, status, and bounded attributes—rather than exporter serialization.
The core repository ships `opentelemetry-test-helpers`,
`OpenTelemetry::SDK::Trace::Export::InMemorySpanExporter`, and analogous in-memory metric readers.

For integration tests, send to a disposable local OTLP receiver and verify the whole path. Never
point a test at a production endpoint.

Sources: [Ruby API 1.11.0](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-api/v1.11.0/api),
[metrics API 0.7.0](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-metrics-api/v0.7.0/metrics_api), and
[logs SDK 0.6.1 examples](https://github.com/open-telemetry/opentelemetry-ruby/tree/opentelemetry-logs-sdk/v0.6.1/examples/logs_sdk).
