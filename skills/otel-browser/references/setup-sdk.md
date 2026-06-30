# Browser SDK setup

Setting up OpenTelemetry in the browser for **traces (spans)** and **events (log records)**. There
is no MeterProvider story in the browser yet.

> **Stability**: `@opentelemetry/browser-sdk` is experimental and may be unpublished — confirm with
> `npm view @opentelemetry/browser-sdk version`. The most settled path today wires the providers
> directly: the **stable** web tracing SDK (`@opentelemetry/sdk-trace-web`, `@opentelemetry/context-zone`)
> for spans, plus the **experimental** Logs SDK (`@opentelemetry/api-logs`, `@opentelemetry/sdk-logs`,
> still on the 0.x line) for events. Both approaches are shown below.

## Core principles

- **Load first, load synchronously.** Instrumentations patch `window`, `fetch`, `XMLHttpRequest`,
  and `history`. Start the SDK before any framework or library patches those globals, or behavior is
  undefined.
- **Two signals, two providers.** Spans go through a `TracerProvider`; events go through a
  `LoggerProvider` (events are emitted via the Logs API as `LogRecord`s).
- **Export over OTLP/HTTP.** gRPC is not available in the browser. Use the OTLP/HTTP (protobuf or
  JSON) exporters.
- **Flush before the page vanishes.** Pages can be closed, backgrounded, or frozen (bfcache) with no
  graceful shutdown. Flush on `visibilitychange`/`pagehide` with `keepalive`, not `unload`. See
  [performance.md](performance.md#page-lifecycle-flush-before-the-page-vanishes).

### A Collector (or vendor edge) in front of browsers

The browser is untrusted and uncontrolled. Routing telemetry through an OpenTelemetry Collector (or
a vendor edge endpoint) rather than directly to a backend store is what makes CORS termination,
sampling, redaction, rate limiting, and trustworthy server-side resource attributes (real client IP,
geo) possible. See [performance.md](performance.md#the-collector-is-your-cost-and-safety-valve).

## Approach A — direct providers

Spans use the **stable** tracing SDK; events use the **experimental** Logs SDK (0.x). There is no
stable browser events path yet — the Logs SDK is the current mechanism.

### Dependencies

```bash
# Tracing (stable)
npm install @opentelemetry/api \
  @opentelemetry/sdk-trace-web \
  @opentelemetry/context-zone \
  @opentelemetry/exporter-trace-otlp-http \
  @opentelemetry/resources \
  @opentelemetry/semantic-conventions

# Events (Logs API) for event-based instrumentations
npm install @opentelemetry/api-logs \
  @opentelemetry/sdk-logs \
  @opentelemetry/exporter-logs-otlp-http \
  @opentelemetry/instrumentation
```

### Tracer provider (spans)

```typescript
import { defaultResource, resourceFromAttributes } from '@opentelemetry/resources';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';
import { WebTracerProvider, BatchSpanProcessor } from '@opentelemetry/sdk-trace-web';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';

// Merge with the default resource so `telemetry.sdk.*` (and any env-detected attributes)
// are preserved. `resourceFromAttributes()` ALONE replaces the default resource and drops
// them — telemetry then arrives with only the attributes you listed. Reuse this one
// `resource` for BOTH the tracer and logger providers so spans and events correlate.
const resource = defaultResource().merge(
  resourceFromAttributes({
    [ATTR_SERVICE_NAME]: 'my-web-app',
    [ATTR_SERVICE_VERSION]: '1.0.0',
  }),
);

const provider = new WebTracerProvider({
  resource,
  spanProcessors: [
    new BatchSpanProcessor(
      new OTLPTraceExporter({ url: 'https://collector.example.com/v1/traces' }),
    ),
  ],
});

provider.register({
  // ZoneContextManager keeps trace context across async user interactions
  // (setTimeout, promises, event handlers). Without it, async callbacks lose the active span.
  contextManager: new ZoneContextManager(),
});
```

> `ZoneContextManager` requires `zone.js` (~1 MB). It is the only reliable way to propagate context
> across async boundaries in the browser, but it is a heavy dependency — see the
> [bundle-size trade-off](performance.md#bundle-size).

### Logger provider (events)

```typescript
import { logs } from '@opentelemetry/api-logs';
import { LoggerProvider, BatchLogRecordProcessor } from '@opentelemetry/sdk-logs';
import { OTLPLogExporter } from '@opentelemetry/exporter-logs-otlp-http';

const loggerProvider = new LoggerProvider({
  // Pass the SAME merged resource as the tracer provider. Without a resource, events carry
  // no `service.name` and cannot be correlated with the spans from the same app.
  resource,
  processors: [
    new BatchLogRecordProcessor(
      new OTLPLogExporter({ url: 'https://collector.example.com/v1/logs' }),
    ),
  ],
});
logs.setGlobalLoggerProvider(loggerProvider);
```

The event-based instrumentations (navigation, web vitals, console, errors, …) emit through this
global `LoggerProvider`. See [instrumentation.md](instrumentation.md).

### Register the instrumentations

```typescript
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { WebVitalsInstrumentation } from '@opentelemetry/browser-instrumentation/experimental/web-vitals';

registerInstrumentations({
  // Span-based instrumentations (fetch, XHR, document-load) resolve their tracer AT THIS CALL:
  // from `tracerProvider` if given, otherwise the global tracer provider. Passing it explicitly
  // is the robust, order-independent choice. Relying on the global instead works only if
  // `provider.register()` ran BEFORE this call — register after, and they bind to the no-op
  // tracer and silently emit nothing.
  tracerProvider: provider,
  instrumentations: [
    new FetchInstrumentation(),     // span-based → uses the tracer provider resolved here
    new WebVitalsInstrumentation(), // event-based → uses the global LoggerProvider set above
  ],
});
```

**Order matters.** Each instrumentation resolves its tracer/logger when `registerInstrumentations`
runs, so `provider.register()` and `logs.setGlobalLoggerProvider()` must run **before** it. Passing
`tracerProvider` explicitly (above) removes the ordering dependency for spans; event-based
instrumentations always read the global logger, so set that first regardless.

### Verify it actually emits

These packages are experimental and their wiring is easy to get subtly wrong (a missing
`tracerProvider`, a provider registered too late, an instrumentation that no-ops silently). After
wiring up, confirm each enabled instrumentation actually produces telemetry rather than assuming it
does:

- Turn on SDK diagnostics during bring-up and watch the console:
  `import { diag, DiagConsoleLogger, DiagLogLevel } from '@opentelemetry/api';`
  `diag.setLogger(new DiagConsoleLogger(), DiagLogLevel.DEBUG);` (remove for production).
- Point at a local Collector with the `debug` exporter (`verbosity: detailed`) and check that the
  span names and event names you expect (e.g. a `fetch` span, an `exception` event) actually arrive
  — exercise each path (navigate, click, fetch, throw) and look for it downstream.
- Treat "no error in the console" as **not** proof of capture: a silently inactive instrumentation
  produces neither errors nor telemetry.

## Approach B — Browser SDK (experimental)

`@opentelemetry/browser-sdk` collapses the boilerplate above into one call. It exports two combined
initializers — `quickStartBrowserSdk` (simplified) and `startBrowserSdk` (deeper control) — plus
per-signal entry points. It is experimental and may be unreleased; config options change frequently,
so re-check the package README (see the Sources of Truth in [SKILL.md](../SKILL.md#sources-of-truth)).

```javascript
import { quickStartBrowserSdk } from '@opentelemetry/browser-sdk';

const sdk = quickStartBrowserSdk({
  serviceName: 'my-web-app',
  serviceVersion: '1.0',
  exportUrl: 'https://collector.example.com', // required
  exportHeaders: { 'x-api-key': '...' },       // optional
});
```

`startBrowserSdk` exposes full control (`resourceAttributes`, `exportConfig`, per-signal `logs` /
`traces` blocks with `spanLimits`/`logRecordLimits`, `contextManager`, `propagators`, `sampler`),
and `await sdk.shutdown()` flushes and stops.

### Per-signal SDKs (tree-shaking)

If you need only one signal, import only that signal's entry point so the bundler drops the other.
Each takes the **full** URL (the signal path is not appended automatically).

| Subpath | Function | Default export URL |
|---|---|---|
| `@opentelemetry/browser-sdk/logs` | `startLogsSdk` | `http://localhost:4318/v1/logs` |
| `@opentelemetry/browser-sdk/traces` | `startTracesSdk` | `http://localhost:4318/v1/traces` |

## Sessions

A **session** correlates the traces and events a single user produces over a time window, expressed
as `session.*` attributes (e.g. `session.id`) attached to every span/log. The Browser SDK ships a
session manager under `@opentelemetry/browser-sdk/session` (`createSessionManager`,
`createLocalStorageSessionStore`, `createSessionSpanProcessor`, `createSessionLogRecordProcessor`),
with configurable `maxDuration` and `inactivityTimeout`.

> **Processor ordering**: register the session processor **before** the export (batch) processor so
> `session.id` is stamped before export. If you set `processors` explicitly you own the pipeline —
> include both the session processor and an export processor.

`session.*` follows the
[session semantic conventions](https://github.com/open-telemetry/semantic-conventions/blob/main/docs/general/session.md).

## Connecting frontend to backend traces

To stitch a browser trace to the backend spans it triggers, the browser must inject the W3C
`traceparent` header on outgoing requests, and the backend must accept it.

- `instrumentation-fetch` / `instrumentation-xml-http-request` inject `traceparent` automatically.
- Set `propagateTraceHeaderCorsUrls` so the header is sent on **cross-origin** requests — it is
  **not** sent cross-origin by default.
- The server must list `traceparent` (and `tracestate`/`baggage` if used) in
  `Access-Control-Allow-Headers`, or the preflight fails and the request is blocked.

```typescript
new FetchInstrumentation({
  propagateTraceHeaderCorsUrls: [/api\.example\.com/],
});
```

## Setup checklist

- [ ] SDK initialized **before** app/framework code and before libraries patch globals
- [ ] `service.name` (and ideally `service.version`) set as resource attributes
- [ ] Resource built by merging the **default** resource (so `telemetry.sdk.*` survives), and the **same** resource passed to both the tracer and logger providers
- [ ] Exporting OTLP/**HTTP** to a Collector you control (not gRPC, not directly to a backend store)
- [ ] Providers registered globally (`provider.register()`, `logs.setGlobalLoggerProvider()`) **before** `registerInstrumentations`, or `tracerProvider` passed to it explicitly — else span instrumentations resolve the no-op tracer
- [ ] `ZoneContextManager` registered if you need trace context across async boundaries
- [ ] `BatchSpanProcessor` / `BatchLogRecordProcessor` (not `Simple*`) for export efficiency
- [ ] Session processor registered **before** the export processor
- [ ] `propagateTraceHeaderCorsUrls` set and server `Access-Control-Allow-Headers` includes `traceparent`
- [ ] Telemetry flushed on `visibilitychange`/`pagehide` (keepalive), not `unload`
- [ ] **Verified** each enabled instrumentation actually emits to the Collector (diag logging + `debug` exporter), not just assumed

## Anti-patterns

| Anti-pattern | Why it's wrong | Fix |
|---|---|---|
| Initializing the SDK after the framework boots | Instrumentations patch globals the framework already wrapped; spans/events go missing | Load the SDK in a synchronous entry module imported first |
| Exporting straight to a backend store from the browser | Leaks credentials, no CORS control, no edge sampling/redaction, unbounded cost | Export to a Collector / vendor edge endpoint |
| Expecting `traceparent` on cross-origin calls by default | The browser only propagates same-origin unless told otherwise | Set `propagateTraceHeaderCorsUrls` **and** server `Access-Control-Allow-Headers` |
| `SimpleSpanProcessor` / `SimpleLogRecordProcessor` in production | One network request per record | Use the Batch processors |
| Flushing on `unload` | Unreliable on mobile/bfcache; blocks navigation | Flush on `visibilitychange`/`pagehide` with `keepalive` |
| Session processor after the batch processor | Records exported before `session.id` is attached | Put the session processor first |
| Shipping `sdk-trace-web` without a context manager | Async user interactions lose parent context | Register `ZoneContextManager` |
| `resource: resourceFromAttributes({...})` as the whole resource | Replaces the default resource; drops `telemetry.sdk.*` and env-detected attributes | Merge: `defaultResource().merge(resourceFromAttributes({...}))` |
| Building a `LoggerProvider` with no `resource` | Events carry no `service.name`; can't be attributed or correlated with spans | Pass the same merged resource to the logger provider |
| `registerInstrumentations` running before `provider.register()` (and no explicit `tracerProvider`) | Span instrumentations resolve the no-op tracer at registration time and emit nothing | Register the provider (or pass `tracerProvider`) **before** registering instrumentations |
