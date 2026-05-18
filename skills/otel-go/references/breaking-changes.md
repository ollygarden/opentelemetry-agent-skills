# Go OpenTelemetry Breaking Changes & Gotchas

Audit reference for upgrading existing Go OpenTelemetry code. For current SDK/contrib
version selection, fetch from the Sources of Truth table in the `otel-go` skill (or
`go.opentelemetry.io/otel` and `go.opentelemetry.io/contrib` release tags directly).

## API Deprecations

- `span.RecordError()` and `span.AddEvent()` are deprecated. Use the Logs API to emit events and record exceptions within the context of the active span.
- `go.opentelemetry.io/contrib/config` is deprecated (contrib v1.35.0). Use `go.opentelemetry.io/contrib/otelconf` instead. The API is identical.
- `WithMetricAttributesFn` deprecated in otelhttp (contrib v1.41.0). Use `Labeler` instead.
- otelgrpc: prefer stats handlers (`NewServerHandler` / `NewClientHandler`) over deprecated interceptors for new code.

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

## Behavioural Notes

- `trace.SpanFromContext()` never returns nil — no nil checks or `IsRecording()` guards needed.
- `log.Record.SetErr(err)` (v1.42.0) — the SDK automatically sets `exception.type` and `exception.message` attributes.

## Resolved Gotchas (historical reference)

- **Propagator not installed by `otelconf`** ([open-telemetry/opentelemetry-go-contrib#6712](https://github.com/open-telemetry/opentelemetry-go-contrib/issues/6712)) — fixed in `otelconf v0.20.0` (released 2026-02-02). The root `otelconf` package now exposes `sdk.Propagator()`; install via `otel.SetTextMapPropagator(sdk.Propagator())`. The schema-pinned `otelconf/v0.3.0` subpackage still requires manual installation: `otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))`.
