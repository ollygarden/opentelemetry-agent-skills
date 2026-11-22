# Ruby instrumentation libraries

## Discover before adding

Inspect the lockfile and bootstrap first:

```sh
bundle list | rg 'opentelemetry|rails|rack|sinatra|sidekiq|faraday|net-http|pg|mysql2|redis'
rg -n "OpenTelemetry::SDK.configure|\.use_all|OpenTelemetry::Instrumentation" .
```

Avoid double instrumentation. Rails can pull together Rack, Action Pack, Active Record, Active
Job, and other libraries; adding lower-level instrumentation separately is useful only when its
configuration or coverage is intentionally required.

## Installation modes

For a bounded surface, install a gem and select its registered constant:

```ruby
gem 'opentelemetry-instrumentation-rack'
gem 'opentelemetry-instrumentation-sidekiq'

OpenTelemetry::SDK.configure do |c|
  c.use 'OpenTelemetry::Instrumentation::Rack'
  c.use 'OpenTelemetry::Instrumentation::Sidekiq'
end
```

For discovery, install `opentelemetry-instrumentation-all` and call `c.use_all`. It installs only
registered instrumentations whose dependency checks pass. Pass a per-instrumentation options hash
or `{ enabled: false }` for exclusions. Do not mix `use` with `use_all`.

## Find the maintained library

The source of truth is the contrib repository's
[`instrumentation/`](https://github.com/open-telemetry/opentelemetry-ruby-contrib/tree/opentelemetry-instrumentation-all/v0.102.0/instrumentation)
directory. It currently includes categories such as:

| Area | Examples |
|---|---|
| Web and Rails | Rails, Rack, Sinatra, Grape, Action Pack/View/Mailer, Active Support |
| HTTP and RPC | Net::HTTP, Faraday, HTTP.rb, HTTPX, Excon, Ethon, RestClient, gRPC |
| Databases and caches | Active Record, PG, mysql2, Trilogy, Mongo, Redis, Dalli, LMDB |
| Jobs and messaging | Active Job, Sidekiq, Resque, Delayed Job, Que, Bunny, ruby-kafka, Racecar, rdkafka |
| Runtime and testing | concurrent-ruby, Logger, Rake, RSpec, Factory Bot |
| Cloud and AI | AWS SDK/Lambda, Anthropic, OpenAI |

This table is a routing aid, not a package/version inventory. Before recommending a gem, open its
README and gemspec in the locked release to confirm its target-library range, Ruby requirement,
registered constant, options, and emitted semantic-convention mode.

Verified 2026-11-22, the latest relevant releases are Rails `0.42.0` (Ruby >= 3.3, Rails >= 7.1),
Rack `0.32.0`, Sinatra `0.31.0`, Sidekiq `0.29.0`, Logger `0.5.0`, and the `all` bundle `0.102.0`.
The latter five also require Ruby >= 3.3. Do not transfer these constraints to an older lockfile.
Anthropic `0.6.0` only propagates context and skips installation for `anthropic` newer than
`1.78.0` (native OpenTelemetry support from `1.79.0`); see its README.

## Framework ordering

Configure instrumentation before the target library begins serving work. Rails instrumentation
uses a Railtie, but its initializer still needs to load during application boot. Rack middleware
and monkey-patching instrumentations can miss setup work or duplicate spans if installed after an
application/server integration has already initialized.

For background jobs and message consumers, verify both producer injection and consumer extraction.
For database and HTTP instrumentation, verify that sensitive statements, URLs, and headers are not
captured unintentionally. Use the relevant library's filtering and sanitization options rather
than post-processing data blindly.

Review client IPs too: since Rack `0.32.0` (which also produces Sinatra's server spans), server
spans carry `client.address` from `Rack::Request#ip` in the stable and `http/dup` modes, and since
Action Pack `0.19.0`, for requests that reach a controller action, Rails overwrites it on the Rack
span with `remote_ip` in every mode (keeping any Rack value when `remote_ip` raises
`IpSpoofAttackError`); both follow trusted-proxy forwarding headers.

## Semantic-convention changes

Instrumentation gems can change emitted span names and attributes independently of core SDK gems.
During an upgrade, compare the affected gem's changelog and tests, then audit dashboards, alerts,
sampling rules, and processors that depend on legacy names. Use `otel-semantic-conventions` for the
released convention, but use the instrumentation release itself to prove what Ruby actually emits.

Since Net::HTTP `0.30.0`, stable mode records the `http.client.request.duration` histogram when a
metrics SDK is configured; see [performance.md](performance.md#metrics-cardinality-and-cost) to
change its buckets.

## Verification

Exercise one representative operation per installed integration and check:

- exactly one expected client/server/producer/consumer span;
- correct parentage across process boundaries;
- route templates rather than raw high-cardinality paths where supported;
- bounded, non-sensitive attributes;
- no boot warning that an instrumentation is unavailable or incompatible.
