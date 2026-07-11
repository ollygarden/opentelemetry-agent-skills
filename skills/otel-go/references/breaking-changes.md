# Go OpenTelemetry Breaking Changes & Gotchas

Audit reference for upgrading existing Go OpenTelemetry code. For current SDK/contrib
version selection, fetch from the Sources of Truth table in the `otel-go` skill (or
`go.opentelemetry.io/otel` and `go.opentelemetry.io/contrib` release tags directly).

## API Deprecations

- `span.RecordError()` and `span.AddEvent()` are deprecated. Use the Logs API to emit events and record exceptions within the context of the active span.
- `go.opentelemetry.io/contrib/config` is deprecated (contrib v1.35.0). Use `go.opentelemetry.io/contrib/otelconf` instead. The API is identical.
- `WithMetricAttributesFn` deprecated in otelhttp (contrib v1.41.0). Use `Labeler` instead.
- otelgrpc: prefer stats handlers (`NewServerHandler` / `NewClientHandler`) over deprecated interceptors for new code.
- `attribute.INVALID` deprecated (core v1.43.0) — an empty value is now valid; use `attribute.EMPTY` instead.
- `attribute.Value.Emit` deprecated (core v1.44.0). Use `attribute.Value.String` instead.
- `OTEL_EXPERIMENTAL_CONFIG_FILE` deprecated in `otelconf` (contrib v1.43.0). Use `OTEL_CONFIG_FILE`.

## Removed APIs (contrib v1.40.0+)

- otelhttp removed `DefaultClient`, `Get`, `Head`, `Post`, `PostForm`, `WithPublicEndpoint`, `WithRouteTag`. Always create a custom client with `otelhttp.NewTransport`.
- HTTP instrumentation sets the `error.type` attribute instead of `exception` span events.

## Semantic Convention Renames (contrib v1.40.0+)

RPC attribute changes:

- `rpc.system` → `rpc.system.name`
- `rpc.method` + `rpc.service` merged into `rpc.method`
- `rpc.client.duration` / `rpc.server.duration` → `rpc.client.call.duration` / `rpc.server.call.duration` (unit changed to seconds)
- `rpc.grpc.status_code` → `rpc.response.status_code`

## otelgrpc (contrib v1.42.0)

- No longer emits `rpc.message` span events, `rpc.*.request/response.size` metrics, or `network.*` attributes — even with `WithMessageEvents`.

## Metric SDK cardinality limit (core v1.44.0)

- `sdk/metric` now enforces a **default cardinality limit of 2000** series per instrument. Series past the limit are dropped into an overflow series (`otel.metric.overflow=true`). Breaking change from the previous unlimited default. Restore unlimited with `sdkmetric.WithCardinalityLimit(0)`; the `OTEL_GO_X_CARDINALITY_LIMIT` env var is deprecated. Per-instrument-kind limits: `sdkmetric.WithCardinalityLimitSelector` (v1.43.0).

## OTLP exporter request-size cap (core v1.44.0)

- All OTLP exporters cap requests at **64 MiB** by default (applied before compression); oversized requests become non-retryable errors. Configure with `WithMaxRequestSize`.

## Baggage limits (core v1.44.0)

- The 8192-byte baggage size limit is now enforced during extraction/parsing (`otel/baggage`, `otel/propagation`); malformed/oversized baggage headers are rejected rather than silently accepted.

## HTTP instrumentation behaviour (contrib v1.44.0)

- Unknown or empty HTTP methods now report `_OTHER` instead of `GET` across all HTTP instrumentations (otelhttp, otelmux).
- The default server span name is now `{method} {route}` (e.g. `GET /foo/{id}`) when a route is available, or `{method}` otherwise — conforming to HTTP semconv.

## Removed / renamed contrib APIs (contrib v1.43.0–v1.44.0)

- otelgrpc removed the deprecated `WithSpanOptions` option (v1.44.0).
- `otelconf`: experimental config types moved from `otelconf` to the `otelconf/x` subpackage (v1.43.0). The `host` resource detector no longer sets `host.id` (v1.43.0).
- otelgrpc added `OTEL_SEMCONV_STABILITY_OPT_IN` (values `rpc` default, `rpc/dup`, `rpc/old`) to stage RPC semconv migration (v1.43.0).

## Log bridges attach errors via SetErr (contrib v1.43.0–v1.44.0)

- otelslog, otelzap, otellogrus, and otellogr now record fields implementing `error` (e.g. `zap.Error`, `slog` error values) via `log.Record.SetErr` instead of emitting them as plain or `exception.*` attributes. The SDK then derives the `exception.*` attributes from the record error.

## Behavioural Notes

- `trace.SpanFromContext()` never returns nil — no nil checks or `IsRecording()` guards needed.
- `log.Record.SetErr(err)` (v1.42.0) — the SDK automatically sets `exception.type` and `exception.message` attributes.

## Resolved Gotchas (historical reference)

- **Propagator not installed by `otelconf`** ([open-telemetry/opentelemetry-go-contrib#6712](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712)) — fixed in `otelconf v0.20.0` (released 2026-02-02). The root `otelconf` package now exposes `sdk.Propagator()`; install via `otel.SetTextMapPropagator(sdk.Propagator())`. The schema-pinned `otelconf/v0.3.0` subpackage still requires manual installation: `otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))`.
