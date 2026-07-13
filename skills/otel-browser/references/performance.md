# Browser performance, cost & privacy

In the browser the SDK is shipped to the user and runs on their device. Every byte of SDK and every
exported record competes with page-load speed and observability spend. Three budgets dominate —
**bundle size**, **runtime cost**, and **telemetry volume/cost** — with **PII** as a hard constraint
across all three.

| Budget | Main levers |
|---|---|
| Bundle size | Tree-shaking, per-signal SDK imports, prefer events over spans, skip `zone.js` when you can |
| Runtime cost | Off-main-thread processing, batch processors, idle callbacks |
| Telemetry volume/cost | Fewer instrumentations, restrict resource-timing, sampling, Collector-side trimming |
| Privacy (PII) | URL sanitization, restrict console capture, redact in the Collector |

## Bundle size

The SDK affects page-load performance, so bundle size is a first-class constraint for browser OTel —
more so than for backend SDKs.

- **Import per-signal SDK entry points** so the bundler drops unused signals. If nothing produces
  spans, import only `@opentelemetry/browser-sdk/logs` and skip the tracing SDK (and vice versa). See
  [setup-sdk.md](setup-sdk.md#per-signal-sdks-tree-shaking).
- **Prefer event-based over span-based** instrumentations where both exist — they pull in less SDK
  surface.
- **Use subpath imports** for instrumentations
  (`@opentelemetry/browser-instrumentation/experimental/web-vitals`) rather than a barrel import, so
  unused instrumentations tree-shake away.
- **`zone.js` is ~1 MB.** `ZoneContextManager` is the only reliable way to propagate trace context
  across async boundaries, but it is heavy. A common trade-off is to relax in-browser async tracing
  and rely on **session correlation** instead, adding `zone.js` only when stitched async spans are
  genuinely needed.
- Pin exact versions and watch the bundle in CI — experimental packages change size between minors.

## Runtime cost (don't block the main thread)

The main thread renders the page; telemetry must stay off it.

- **Resource timing** already uses `requestIdleCallback` (Safari `setTimeout` fallback), processes in
  batches, and bounds time per idle callback. Tune `batchSize`, `maxProcessingTime`, and
  `maxQueueSize` if you load many resources. See
  [instrumentation.md](instrumentation.md#resource-timing-browserresource_timing).
- **Always use the Batch processors** (`BatchSpanProcessor`, `BatchLogRecordProcessor`), never the
  `Simple*` variants (one network request per record). Bound the queue and batch:

```typescript
new BatchLogRecordProcessor({
  exporter,
  maxQueueSize: 100,        // illustrative — drop records past this rather than grow unbounded
  maxExportBatchSize: 50,
  scheduledDelayMillis: 5000,
});
```

(Numbers are illustrative, not a standard — tune to your traffic.)

## Page lifecycle: flush before the page vanishes

A browser page can be closed, backgrounded, or frozen (bfcache) at any time with no graceful
shutdown. Telemetry buffered in a batch processor is lost if you don't flush in time.

- **Flush on `visibilitychange`/`pagehide`, not `unload`/`beforeunload`** — the latter are unreliable
  (especially on mobile) and break bfcache. The browser `BatchSpanProcessor` and
  `BatchLogRecordProcessor` install these flush handlers by default unless
  `disableAutoFlushOnDocumentHide` is set.
- The OTLP/HTTP browser exporter uses **`fetch` with `keepalive` when browser limits allow it** (the
  modern replacement for `navigator.sendBeacon`) so in-flight exports can survive the page going
  away.
- Late-finalizing signals (INP, CLS, Web Vitals generally) are only known near the end of the page
  lifecycle — they depend on this flush actually happening.

```typescript
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'hidden') {
    void loggerProvider.forceFlush();
    void tracerProvider.forceFlush();
  }
});
document.addEventListener('pagehide', () => {
  void loggerProvider.forceFlush();
  void tracerProvider.forceFlush();
});
```

## Telemetry volume and cost

RUM volume scales with user traffic and is easy to make ruinously expensive. A browser can emit many
loosely-related events per second, and a single content-heavy page can produce hundreds of
resource-timing records.

- **Enable a minimal set first** (e.g. Web Vitals + Errors + Navigation); add resource timing,
  console, and user actions deliberately once you understand their volume.
- **Restrict resource timing** with `initiatorTypes` (e.g. `['xmlhttprequest', 'fetch']`) and
  `ignoreUrls` instead of capturing every script/image/font and known-noisy endpoints.
- **Restrict console capture** to `['error', 'warn']` in production.
- **Sample.** Apply head sampling in the SDK and/or sampling at the Collector edge. Because clients
  are untrusted and chatty, edge sampling and rate limiting protect both cost and your backend.
- **Watch cardinality.** Rich user/session context is valuable for slicing by affected cohort, but
  high-cardinality attributes drive backend cost — balance richness against spend.

## The Collector is your cost and safety valve

Sending browser telemetry to an OpenTelemetry Collector (or vendor edge endpoint) rather than
directly to a backend store is the only place to enforce policy on an untrusted client:

- **Sampling and rate limiting** to cap RUM spend and absorb hostile/buggy clients.
- **Redaction** of PII the client should not have sent — see the `redaction` processor in the
  `otel-collector` skill.
- **CORS termination**, keeping backend credentials out of client code.
- **Trustworthy resource attributes** (real client IP, geo) the browser cannot self-report.

## Privacy / PII (hard constraint)

PII leaks into browser telemetry through URLs, console output, form fields, and click targets.
Controlling it is partly developer discipline and partly pipeline enforcement.

| Vector | Risk | Mitigation |
|---|---|---|
| URLs (`url.full`) | tokens, emails, IDs in path/query | `sanitizeUrl` / `defaultSanitizeUrl` on the navigation instrumentation |
| `console.log`/`info`/`debug` | apps log user data to the console | capture only `['error', 'warn']` in prod |
| `data-otel-*` attributes | exported verbatim | keep to non-PII business keys only |
| Free-form custom attributes | accidental PII | review `applyCustomAttributes` / `applyCustomLogRecordData` hooks |
| Anything the client sends | the client is untrusted | redact in the Collector as a backstop |

## Performance checklist

- [ ] Per-signal SDK imports; unused signal SDK tree-shaken out
- [ ] Subpath imports for instrumentations (no barrel import)
- [ ] `zone.js`/`ZoneContextManager` included **only** if async span stitching is required
- [ ] Batch processors with bounded queue/batch sizes (no `Simple*` processors)
- [ ] Browser batch processor auto-flush on document hide left enabled, or equivalent
      `forceFlush()` wired to `visibilitychange`/`pagehide`; export uses `keepalive` when possible
- [ ] Minimal instrumentation set enabled; resource timing restricted by `initiatorTypes` /
      `ignoreUrls`
- [ ] Console capture limited to `error`/`warn` in production
- [ ] Sampling configured (SDK and/or Collector edge)
- [ ] URL sanitization on; PII redaction enforced in the Collector
- [ ] Exporting to a Collector, not directly to a backend store

## Background

These durable learnings are corroborated by OpenTelemetry community talks:

- **Coming soon to a browser near you: OpenTelemetry** — Browser SIG panel, Oct 2025
  ([recap](https://embrace.io/blog/browser-opentelemetry-panel-recap/)). Events-vs-spans model,
  sessions as the correlation primitive, the `zone.js` trade-off, bundle-size strategy.
- **From RUM to Front-End Observability with OpenTelemetry** — Purvi Kanal, KubeCon EU 2024
  ([video](https://www.youtube.com/watch?v=l2_wsvv-Rhs)). Connecting browser spans to backend traces.
- **A Practical Guide to Debugging Browser Performance with OpenTelemetry** — Purvi Kanal,
  KubeCon NA 2023 ([video](https://www.youtube.com/watch?v=0J1z599tmfY)). Core Web Vitals, page/
  resource timing, long tasks.

> The `browser.*` semantic conventions are still evolving and vendors differ in how they represent
> Core Web Vitals. Confirm attribute names and thresholds against the current
> [OpenTelemetry browser semantic conventions](https://opentelemetry.io/docs/specs/semconv/browser/)
> before encoding them.
