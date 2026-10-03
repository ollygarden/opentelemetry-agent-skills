# Go OpenTelemetry Breaking Changes & Gotchas

Audit reference for upgrading existing Go OpenTelemetry code. For current SDK/contrib
version selection, fetch from the Sources of Truth table in the `otel-go` skill (or
`go.opentelemetry.io/otel` and `go.opentelemetry.io/contrib` release tags directly).

After applying an upgrade, verify it locally with `go mod tidy -diff`, `go build ./...`,
and `go test ./...`. For exporter URL or retry behavior, use an `httptest.Server` or
another disposable local receiver; never probe a deployment endpoint merely to confirm
the SDK migration.

## API Deprecations

- `go.opentelemetry.io/contrib/config` was removed in contrib v1.35.0. Use `go.opentelemetry.io/contrib/otelconf` instead.
- `WithMetricAttributesFn` deprecated in otelhttp v0.66.0 (contrib v1.41.0 release). Use `Labeler` instead.
- otelgrpc's deprecated client/server interceptors were removed in v0.60.0. Use stats handlers (`NewServerHandler` / `NewClientHandler`).
- `attribute.INVALID` deprecated (core v1.43.0) — an empty value is now valid; use `attribute.EMPTY` instead.
- `attribute.Value.Emit` deprecated (core v1.44.0). Use `attribute.Value.String` instead.
- `sdk/log.WithExportBufferSize` is deprecated and has no effect as of log SDK v0.21.0 (core v1.45.0 release); the batch processor no longer keeps a separate export-request buffer.
- `OTEL_EXPERIMENTAL_CONFIG_FILE` is no longer supported by root `otelconf` v0.23.0+ (contrib v1.43.0+). Use `OTEL_CONFIG_FILE`.
- `go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho` was deprecated in v0.71.0 (contrib v1.46.0) and removed in v0.72.0 (contrib v1.47.0). Use `github.com/labstack/echo-opentelemetry`.
- `go.opentelemetry.io/otel/log/global` is deprecated (core v1.47.0). Use `otel.Logger`, `otel.GetLoggerProvider`, and `otel.SetLoggerProvider` from `go.opentelemetry.io/otel`.
- Deprecated as no longer supported in contrib v1.47.0, with no contrib replacement: `instrumentation/github.com/aws/aws-sdk-go-v2/otelaws` (v0.72.0), `propagators/aws`, and `samplers/probability/consistent`.
- otelhttp v0.72.0 deprecates `ReadBytesKey`, `ReadErrorKey`, `WroteBytesKey`, and `WriteErrorKey`; `WithMessageEvents` no longer emits `http.read_bytes` / `http.wrote_bytes`. Total body sizes remain on the server span as `http.request.body.size` / `http.response.body.size`.

## Removed APIs (instrumentation v0.65.0; contrib v1.40.0 release)

- otelhttp removed `DefaultClient`, `Get`, `Head`, `Post`, `PostForm`, `WithPublicEndpoint`, `WithRouteTag`. Always create a custom client with `otelhttp.NewTransport`.
- HTTP instrumentation sets the `error.type` attribute instead of `exception` span events.

## Semantic Convention Renames (instrumentation v0.65.0; contrib v1.40.0 release)

RPC attribute changes:

- `rpc.system` → `rpc.system.name`
- `rpc.method` + `rpc.service` merged into `rpc.method`
- `rpc.client.duration` / `rpc.server.duration` → `rpc.client.call.duration` / `rpc.server.call.duration` (unit changed to seconds)
- `rpc.grpc.status_code` → `rpc.response.status_code`

## otelgrpc v0.67.0 (contrib v1.42.0 release)

- No longer emits `rpc.message` span events, `rpc.*.request/response.size` metrics, or `network.*` attributes — even with `WithMessageEvents`.

## Metric SDK cardinality limit (core v1.44.0)

- `sdk/metric` now enforces a **default cardinality limit of 2000** series per instrument. Series past the limit are dropped into an overflow series (`otel.metric.overflow=true`). Breaking change from the previous unlimited default. Restore unlimited with `sdkmetric.WithCardinalityLimit(0)`; the `OTEL_GO_X_CARDINALITY_LIMIT` env var is deprecated. Per-instrument-kind limits: `sdkmetric.WithCardinalityLimitSelector` (v1.43.0).

## OTLP exporter request-size cap (core v1.44.0)

- All OTLP exporters cap requests at **64 MiB** by default (applied before compression); oversized requests become non-retryable errors. Configure with `WithMaxRequestSize`.

## Baggage limits (core v1.44.0)

- The 8192-byte baggage size limit is now enforced during extraction/parsing (`otel/baggage`, `otel/propagation`): baggage strings over the limit are rejected outright, while malformed members are skipped — valid members are retained and an error is reported.

## Log value API replacement (log API/SDK v0.21.0; core v1.45.0 release)

- Log bodies and record attributes now use `go.opentelemetry.io/otel/attribute.Value` and `attribute.KeyValue`. The former `log.Kind`, `log.Value`, `log.KeyValue`, their constructors, and conversion helpers were removed. Replace `log.StringValue(...)` / `log.String(...)` with `attribute.StringValue(...)` / `attribute.String(...)` (and the corresponding typed constructors).
- `sdk/log/logtest.RecordFactory` no longer has `AttributeValueLengthLimit` or `AttributeCountLimit`; test records now keep attribute limits disabled.

## OTLP HTTP endpoint URLs (core v1.45.0)

- `otlptracehttp.WithEndpointURL` and `otlpmetrichttp.WithEndpointURL` no longer append `/v1/traces` or `/v1/metrics` when the URL has no path. They now use `/`, matching signal-specific endpoint environment variables and the log exporter. Join the signal path explicitly to preserve the old behavior.

## Core v1.47.0 (logs API/SDK v1.47.0; log exporters v0.23.0)

- `go.opentelemetry.io/otel/log` and `go.opentelemetry.io/otel/sdk/log` are stable and now tagged on the v1.x line (`@v1.47.0`); the OTLP/stdout log exporters and `logtest` modules continue on v0.x (v0.23.0).
- Requires Go 1.26 or later (Go 1.25 support dropped in core and contrib v1.47.0).
- Nested attribute values are capped at depth 64 by default (tuning: [performance.md](performance.md#span-attribute-limits)).
- The experimental `OTEL_GO_X_METRIC_EXPORT_BATCH_SIZE` variable was removed; use `sdkmetric.WithMaxExportBatchSize` on the `PeriodicReader`.
- Empty `OTEL_TRACES_SAMPLER` / `OTEL_TRACES_SAMPLER_ARG` values are treated as unset.
- gRPC OTLP exporters ignore HTTP URL paths when deriving targets from `OTEL_EXPORTER_OTLP_ENDPOINT` / signal-specific endpoint variables.

## otelhttp client metrics labeler (instrumentation v0.72.0; contrib v1.47.0 release)

- Client metrics no longer include attributes from the server-side `Labeler`. Code that used `ContextWithLabeler` to add custom *client* metric attributes must switch to `ContextWithClientLabeler` (read back with `ClientLabelerFromContext`). Server-side `ContextWithLabeler` / `LabelerFromContext` is unchanged.

## Log exporter shutdown (log SDK v0.22.0; core v1.46.0 release)

- `sdk/log` adds `ErrExporterShutdown`. The built-in stdout and OTLP log exporters now return it when `Export` is called after `Shutdown`; custom exporters should do the same. Calls to `Shutdown` or `ForceFlush` after shutdown remain no-ops that return `nil`.

## HTTP instrumentation behaviour (instrumentation v0.69.0; contrib v1.44.0 release)

- Unknown or empty HTTP methods now report `_OTHER` instead of `GET` across all HTTP instrumentations (otelhttp, otelmux).
- The default server span name is now `{method} {route}` (e.g. `GET /foo/{id}`) when a route is available, or `{method}` otherwise — conforming to HTTP semconv.

In otelhttp v0.70.0 (contrib v1.45.0), client spans and request metrics derive `network.protocol.name` / `network.protocol.version` from the negotiated response protocol rather than the request's initial `Proto` value.

## Removed / renamed contrib APIs (v1.43.0–v1.44.0 release lines)

- otelgrpc removed the deprecated `WithSpanOptions` option in v0.69.0.
- `otelconf`: experimental config types moved from `otelconf` to the `otelconf/x` subpackage in v0.23.0. The `host` resource detector no longer sets `host.id` in v0.23.0.
- otelgrpc added `OTEL_SEMCONV_STABILITY_OPT_IN` (values `rpc` default, `rpc/dup`, `rpc/old`) in v0.68.0 to stage RPC semconv migration.
- Contrib v1.45.0 updates instrumentation and detectors to semconv v1.43.0; use the exact versioned semconv package and its generated migration file rather than mixing helpers from older versions.

## Log bridges attach errors via SetErr (bridge v0.18.0–v0.19.0)

- otelslog, otelzap, otellogrus, and otellogr now record fields implementing `error` (e.g. `zap.Error`, `slog` error values) via `log.Record.SetErr` instead of emitting them as plain or `exception.*` attributes. The SDK then derives the `exception.*` attributes from the record error.

## Behavioural Notes

- `trace.SpanFromContext()` never returns nil — no nil checks or `IsRecording()` guards needed.
- `span.RecordError()` and `span.AddEvent()` are still present in the released tracing API; do not remove existing uses as deprecated APIs.
- `log.Record.SetErr(err)` (`otel/log` v0.18.0; released with core v1.42.0) — the SDK automatically sets `exception.type` and `exception.message` attributes.

## Resolved Gotchas (historical reference)

- **Propagator not installed by `otelconf`** ([open-telemetry/opentelemetry-go-contrib#6712](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712)) — fixed in `otelconf v0.20.0` (released 2026-02-02). The root `otelconf` package now exposes `sdk.Propagator()`; install via `otel.SetTextMapPropagator(sdk.Propagator())`. The schema-pinned `otelconf/v0.3.0` subpackage still requires manual installation: `otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))`.
