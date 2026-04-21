---
name: opentelemetry-go
description: >
  OpenTelemetry Go API, SDK, and instrumentation library mechanics.
  Use when writing, reviewing, or configuring OpenTelemetry instrumentation
  in Go applications. Covers current versions, import paths, breaking changes,
  provider setup, contrib libraries, and performance tuning.
---

# OpenTelemetry Go

Mechanics reference for the OpenTelemetry Go API, SDK, and instrumentation libraries.

## Current Versions

| Component | Version | Notes |
|---|---|---|
| SDK (`go.opentelemetry.io/otel`) | v1.42.0 | Requires Go 1.25+ |
| Contrib (`go.opentelemetry.io/contrib`) | v1.42.0 | Matches SDK version |
| Semconv package | `go.opentelemetry.io/otel/semconv/v1.40.0` | Latest available |

## Gotchas and Breaking Changes

- `span.RecordError()` and `span.AddEvent()` are deprecated. Use the Logs API to emit events and record exceptions within the context of the active span.
- `go.opentelemetry.io/contrib/config` is deprecated (contrib v1.35.0). Use `go.opentelemetry.io/contrib/otelconf/v0.3.0` instead. The API is identical.
- Current SDK version: v1.42.0 (requires Go 1.25+). Current semconv package: `go.opentelemetry.io/otel/semconv/v1.40.0`.
- otelhttp removed `DefaultClient`, `Get`, `Head`, `Post`, `PostForm`, `WithPublicEndpoint`, `WithRouteTag` in contrib v1.40.0. Always create a custom client with `otelhttp.NewTransport`.
- RPC semantic convention renames in contrib v1.40.0+: `rpc.system` -> `rpc.system.name`; `rpc.method` + `rpc.service` merged into `rpc.method`; `rpc.client.duration`/`rpc.server.duration` -> `rpc.client.call.duration`/`rpc.server.call.duration` (unit changed to seconds); `rpc.grpc.status_code` -> `rpc.response.status_code`.
- otelgrpc in contrib v1.42.0: no longer emits `rpc.message` span events, `rpc.*.request/response.size` metrics, or `network.*` attributes. Even with `WithMessageEvents`.
- HTTP instrumentation (contrib v1.40.0+) sets `error.type` attribute instead of `exception` span events.
- otelgrpc: prefer stats handlers (`NewServerHandler`/`NewClientHandler`) over deprecated interceptors for new code.
- `trace.SpanFromContext()` never returns nil — no nil checks or `IsRecording()` guards needed.
- Propagator must be set manually via `otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))` until contrib issue #6712 is resolved.
- `WithMetricAttributesFn` deprecated in otelhttp (contrib v1.41.0). Use `Labeler` instead.
- `log.Record.SetErr(err)` (v1.42.0) — the SDK automatically sets `exception.type` and `exception.message` attributes.

## References

Load these on demand based on the task:

- Setting up the SDK, configuring providers, or initializing telemetry? Read `references/sdk-setup.md`.
- Using the tracing, metrics, or logs API? Read `references/api.md`.
- Adding or configuring instrumentation libraries (otelhttp, otelgrpc, otelgin, otelsql, logging bridges)? Read `references/instrumentation-libraries.md`.
- Tuning batch sizes, sampling, cardinality limits, or export intervals? Read `references/performance.md`.
- Using file-based YAML configuration for the SDK? Use the `opentelemetry-sdk-configuration` skill for the schema, then see `references/sdk-setup.md` for Go-specific loading via `otelconf`.
